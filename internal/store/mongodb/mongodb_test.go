package mongodb

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// TestAdapterAgainstRealMongo is an integration test. It creates its own
// throwaway collection, so it never depends on demo seed data. Run with:
//
//	MEROS_TEST_MONGODB_URI=mongodb://localhost:27017 go test ./internal/store/mongodb/
func TestAdapterAgainstRealMongo(t *testing.T) {
	uri := os.Getenv("MEROS_TEST_MONGODB_URI")
	if uri == "" {
		t.Skip("set MEROS_TEST_MONGODB_URI to run the MongoDB integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const (
		db   = "meros_test"
		coll = "landmarks_it"
	)

	// Seed a throwaway collection with a 2dsphere index.
	seedClient, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer seedClient.Disconnect(context.Background())
	c := seedClient.Database(db).Collection(coll)
	_ = c.Drop(ctx)
	_, err = c.InsertMany(ctx, []any{
		bson.M{"_id": "a", "type": "Feature",
			"geometry":   bson.M{"type": "Point", "coordinates": bson.A{132.4536, 34.3956}},
			"properties": bson.M{"name": "Atomic Bomb Dome"}},
		bson.M{"_id": "b", "type": "Feature",
			"geometry":   bson.M{"type": "Point", "coordinates": bson.A{132.4593, 34.4027}},
			"properties": bson.M{"name": "Hiroshima Castle"}},
		bson.M{"_id": "c", "type": "Feature",
			"geometry":   bson.M{"type": "Point", "coordinates": bson.A{132.3198, 34.2959}},
			"properties": bson.M{"name": "Itsukushima Shrine"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "geometry", Value: "2dsphere"}},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Drop(context.Background()) })

	// Exercise the adapter through the FeatureStore contract only.
	store, err := Connect(ctx, uri, map[string]Source{
		"landmarks": {Database: db, Collection: coll},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())

	central := core.BBox{132.44, 34.38, 132.49, 34.41}
	fc, err := store.GetItems(ctx, "landmarks", core.Query{Limit: 10, BBox: &central})
	if err != nil {
		t.Fatal(err)
	}
	if len(fc.Features) != 2 {
		t.Fatalf("bbox query returned %d features, want 2", len(fc.Features))
	}

	f, err := store.GetItem(ctx, "landmarks", "b")
	if err != nil {
		t.Fatal(err)
	}
	if f.Properties["name"] != "Hiroshima Castle" {
		t.Errorf("name = %v, want Hiroshima Castle", f.Properties["name"])
	}

	if _, err := store.GetItem(ctx, "landmarks", "zzz"); !errors.Is(err, core.ErrFeatureNotFound) {
		t.Fatalf("err = %v, want ErrFeatureNotFound", err)
	}
	if _, err := store.GetItems(ctx, "ghost", core.Query{Limit: 1}); !errors.Is(err, core.ErrCollectionNotFound) {
		t.Fatalf("err = %v, want ErrCollectionNotFound", err)
	}
}
