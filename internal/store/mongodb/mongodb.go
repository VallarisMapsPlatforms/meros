// Package mongodb is the MongoDB FeatureStore adapter. Every piece of
// MongoDB vocabulary (bson, ObjectID, $-operators) stays inside this
// package; nothing above it ever sees a driver type.
package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// DefaultGeometryField is the GeoJSON geometry field indexed with
// 2dsphere. A collection that stores its geometry under another name
// overrides this in the catalog.
const DefaultGeometryField = "geometry"

// Per-operation ceiling: these are short OLTP-style reads; anything
// slower than this indicates a missing index, and failing fast beats
// hanging a request.
const opTimeout = 5 * time.Second

// Source locates one collection's data inside MongoDB.
type Source struct {
	Database      string
	Collection    string
	GeometryField string // defaults to DefaultGeometryField
}

// Store serves features from MongoDB through one long-lived client.
type Store struct {
	client  *mongo.Client
	sources map[string]Source // collection id → location
}

var _ core.FeatureStore = (*Store)(nil)

// Connect creates the one client shared by all requests. Pooling is this
// adapter's decision alone — it lives in exactly one place, and changing it
// never reaches past the seam.
func Connect(ctx context.Context, uri string, sources map[string]Source) (*Store, error) {
	opts := options.Client().
		ApplyURI(uri).
		// Modest pool: a single small API instance serving read traffic.
		// The driver default (100) would only hold idle sockets; raise
		// this when fronting real concurrency.
		SetMaxPoolSize(25).
		// Fail fast when the server is unreachable rather than hanging.
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(5 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		// Never wrap the URI into errors — it may carry credentials.
		return nil, fmt.Errorf("mongodb: connect: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongodb: ping: %w", err)
	}

	normalized := make(map[string]Source, len(sources))
	for id, src := range sources {
		if src.GeometryField == "" {
			src.GeometryField = DefaultGeometryField
		}
		normalized[id] = src
	}
	return &Store{client: client, sources: normalized}, nil
}

// Close releases the client and its pool.
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *Store) GetItems(ctx context.Context, collection string, q core.Query) (*core.FeatureCollection, error) {
	src, ok := s.sources[collection]
	if !ok {
		return nil, core.ErrCollectionNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	filter := bson.M{}
	if q.BBox != nil {
		filter[src.GeometryField] = geoIntersectsBBox(*q.BBox)
	}

	// A stable order is required for offset pagination to be correct
	// (and it keeps pages identical across backends).
	findOpts := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}})
	if q.Limit > 0 {
		findOpts.SetLimit(int64(q.Limit))
	}
	if q.Offset > 0 {
		findOpts.SetSkip(int64(q.Offset))
	}

	cur, err := s.coll(src).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("mongodb: find %s: %w", collection, err)
	}
	defer cur.Close(ctx)

	var out []core.Feature
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("mongodb: decode %s: %w", collection, err)
		}
		f, err := toFeature(doc, src.GeometryField)
		if err != nil {
			return nil, fmt.Errorf("mongodb: %s: %w", collection, err)
		}
		out = append(out, f)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("mongodb: cursor %s: %w", collection, err)
	}
	return &core.FeatureCollection{Features: out}, nil
}

func (s *Store) GetItem(ctx context.Context, collection, id string) (*core.Feature, error) {
	src, ok := s.sources[collection]
	if !ok {
		return nil, core.ErrCollectionNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	// Ids may be plain strings or ObjectIDs, depending on how the documents
	// were written. Try both.
	res := s.coll(src).FindOne(ctx, bson.M{"_id": id})
	if errors.Is(res.Err(), mongo.ErrNoDocuments) {
		if oid, err := bson.ObjectIDFromHex(id); err == nil {
			res = s.coll(src).FindOne(ctx, bson.M{"_id": oid})
		}
	}
	if errors.Is(res.Err(), mongo.ErrNoDocuments) {
		return nil, core.ErrFeatureNotFound
	}
	if res.Err() != nil {
		return nil, fmt.Errorf("mongodb: find one %s: %w", collection, res.Err())
	}

	var doc bson.M
	if err := res.Decode(&doc); err != nil {
		return nil, fmt.Errorf("mongodb: decode %s: %w", collection, err)
	}
	f, err := toFeature(doc, src.GeometryField)
	if err != nil {
		return nil, fmt.Errorf("mongodb: %s: %w", collection, err)
	}
	return &f, nil
}

func (s *Store) coll(src Source) *mongo.Collection {
	return s.client.Database(src.Database).Collection(src.Collection)
}

// geoIntersectsBBox turns a bbox into the spatial filter MongoDB expects:
// a closed counter-clockwise polygon ring matched with $geoIntersects
// against the 2dsphere index.
func geoIntersectsBBox(b core.BBox) bson.M {
	west, south, east, north := b[0], b[1], b[2], b[3]
	return bson.M{
		"$geoIntersects": bson.M{
			"$geometry": bson.M{
				"type": "Polygon",
				"coordinates": [][][]float64{{
					{west, south},
					{east, south},
					{east, north},
					{west, north},
					{west, south},
				}},
			},
		},
	}
}

// toFeature translates one MongoDB document into the neutral domain type.
// This function is the exact spot where MongoDB's dialect ends.
func toFeature(doc bson.M, geometryField string) (core.Feature, error) {
	f := core.Feature{Properties: map[string]any{}}

	switch id := doc["_id"].(type) {
	case string:
		f.ID = id
	case bson.ObjectID:
		f.ID = id.Hex()
	default:
		f.ID = fmt.Sprint(id)
	}

	if geom, ok := doc[geometryField]; ok && geom != nil {
		raw, err := json.Marshal(geom)
		if err != nil {
			return core.Feature{}, fmt.Errorf("feature %s: encode geometry: %w", f.ID, err)
		}
		f.Geometry = raw
	}

	// Driver v2 decodes embedded documents as bson.D where v1 gave bson.M.
	// Accept either, so this does not depend on how the driver represents
	// a sub-document.
	switch props := doc["properties"].(type) {
	case bson.M:
		f.Properties = map[string]any(props)
	case bson.D:
		m := make(map[string]any, len(props))
		for _, e := range props {
			m[e.Key] = e.Value
		}
		f.Properties = m
	}
	return f, nil
}
