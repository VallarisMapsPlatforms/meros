// Package file is a zero-dependency FeatureStore backed by plain GeoJSON
// files, read from disk at startup and served from memory. It is the second
// implementation that keeps the FeatureStore contract honest, and the fake
// the test suite runs against.
package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// Store serves features read from GeoJSON files.
type Store struct {
	features map[string][]core.Feature // collection id → features
	bounds   map[string][]core.BBox    // per-feature bbox, same index
}

var _ core.FeatureStore = (*Store)(nil)

// New builds a store from GeoJSON files: collection id → file path.
func New(sources map[string]string) (*Store, error) {
	s := newEmpty()
	for id, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("file store: collection %q: %w", id, err)
		}
		if err := s.addCollection(id, raw); err != nil {
			return nil, fmt.Errorf("file store: collection %q: %w", id, err)
		}
	}
	return s, nil
}

// NewFromGeoJSON builds a store from GeoJSON documents already in memory —
// used by tests and embedded samples.
func NewFromGeoJSON(docs map[string][]byte) (*Store, error) {
	s := newEmpty()
	for id, raw := range docs {
		if err := s.addCollection(id, raw); err != nil {
			return nil, fmt.Errorf("file store: collection %q: %w", id, err)
		}
	}
	return s, nil
}

func newEmpty() *Store {
	return &Store{
		features: make(map[string][]core.Feature),
		bounds:   make(map[string][]core.BBox),
	}
}

func (s *Store) GetItems(ctx context.Context, collection string, q core.Query) (*core.FeatureCollection, error) {
	feats, ok := s.features[collection]
	if !ok {
		return nil, core.ErrCollectionNotFound
	}

	matched := make([]core.Feature, 0, len(feats))
	for i, f := range feats {
		if q.BBox != nil && !intersects(s.bounds[collection][i], *q.BBox) {
			continue
		}
		matched = append(matched, f)
	}

	// Page after filtering, in stable declaration order.
	if q.Offset > 0 {
		if q.Offset >= len(matched) {
			matched = nil
		} else {
			matched = matched[q.Offset:]
		}
	}
	if q.Limit > 0 && len(matched) > q.Limit {
		matched = matched[:q.Limit]
	}

	return &core.FeatureCollection{Features: matched}, nil
}

func (s *Store) GetItem(ctx context.Context, collection, id string) (*core.Feature, error) {
	feats, ok := s.features[collection]
	if !ok {
		return nil, core.ErrCollectionNotFound
	}
	for i := range feats {
		if feats[i].ID == id {
			f := feats[i]
			return &f, nil
		}
	}
	return nil, core.ErrFeatureNotFound
}

type geojsonFeatureCollection struct {
	Features []struct {
		ID         any             `json:"id"`
		Geometry   json.RawMessage `json:"geometry"`
		Properties map[string]any  `json:"properties"`
	} `json:"features"`
}

func (s *Store) addCollection(id string, raw []byte) error {
	var fc geojsonFeatureCollection
	if err := json.Unmarshal(raw, &fc); err != nil {
		return fmt.Errorf("parse GeoJSON: %w", err)
	}
	for i, gf := range fc.Features {
		fid := fmt.Sprintf("%s-%d", id, i)
		if gf.ID != nil {
			fid = fmt.Sprint(gf.ID)
		}
		bbox, err := geometryBounds(gf.Geometry)
		if err != nil {
			return fmt.Errorf("feature %s: %w", fid, err)
		}
		props := gf.Properties
		if props == nil {
			props = map[string]any{}
		}
		s.features[id] = append(s.features[id], core.Feature{
			ID:         fid,
			Geometry:   gf.Geometry,
			Properties: props,
		})
		s.bounds[id] = append(s.bounds[id], bbox)
	}
	return nil
}

// geometryBounds computes the bounding box of any coordinates-based
// GeoJSON geometry by walking its nested coordinate arrays.
func geometryBounds(raw json.RawMessage) (core.BBox, error) {
	var g struct {
		Coordinates any `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		return core.BBox{}, fmt.Errorf("parse geometry: %w", err)
	}
	b := core.BBox{180, 90, -180, -90}
	if g.Coordinates == nil {
		// No coordinates (e.g. GeometryCollection): never filtered out.
		return core.BBox{-180, -90, 180, 90}, nil
	}
	walkCoordinates(g.Coordinates, &b)
	return b, nil
}

func walkCoordinates(node any, b *core.BBox) {
	arr, ok := node.([]any)
	if !ok || len(arr) == 0 {
		return
	}
	if x, isNum := arr[0].(float64); isNum {
		// A position: [lon, lat, ...]
		if len(arr) < 2 {
			return
		}
		y, ok := arr[1].(float64)
		if !ok {
			return
		}
		b[0] = min(b[0], x)
		b[1] = min(b[1], y)
		b[2] = max(b[2], x)
		b[3] = max(b[3], y)
		return
	}
	for _, child := range arr {
		walkCoordinates(child, b)
	}
}

func intersects(a, q core.BBox) bool {
	return a[0] <= q[2] && a[2] >= q[0] && a[1] <= q[3] && a[3] >= q[1]
}
