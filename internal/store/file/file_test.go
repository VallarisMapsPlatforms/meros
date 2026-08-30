package file

import (
	"context"
	"errors"
	"testing"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// Three points: two inside central Hiroshima, one far away on Miyajima.
const sampleGeoJSON = `{
	"type": "FeatureCollection",
	"features": [
		{"type": "Feature", "id": "a", "geometry": {"type": "Point", "coordinates": [132.4536, 34.3956]}, "properties": {"name": "Atomic Bomb Dome"}},
		{"type": "Feature", "id": "b", "geometry": {"type": "Point", "coordinates": [132.4593, 34.4027]}, "properties": {"name": "Hiroshima Castle"}},
		{"type": "Feature", "id": "c", "geometry": {"type": "Point", "coordinates": [132.3198, 34.2959]}, "properties": {"name": "Itsukushima Shrine"}}
	]
}`

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewFromGeoJSON(map[string][]byte{"landmarks": []byte(sampleGeoJSON)})
	if err != nil {
		t.Fatalf("NewFromGeoJSON: %v", err)
	}
	return s
}

func TestGetItems(t *testing.T) {
	central := core.BBox{132.44, 34.38, 132.49, 34.41}

	tests := []struct {
		name    string
		query   core.Query
		wantIDs []string
	}{
		{"all items", core.Query{Limit: 10}, []string{"a", "b", "c"}},
		{"bbox filters to central hiroshima", core.Query{Limit: 10, BBox: &central}, []string{"a", "b"}},
		{"limit pages", core.Query{Limit: 2}, []string{"a", "b"}},
		{"offset continues", core.Query{Limit: 2, Offset: 2}, []string{"c"}},
		{"offset beyond end", core.Query{Limit: 2, Offset: 99}, nil},
	}

	s := newTestStore(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc, err := s.GetItems(context.Background(), "landmarks", tt.query)
			if err != nil {
				t.Fatalf("GetItems: %v", err)
			}
			if len(fc.Features) != len(tt.wantIDs) {
				t.Fatalf("returned %d features, want %d", len(fc.Features), len(tt.wantIDs))
			}
			for i, want := range tt.wantIDs {
				if got := fc.Features[i].ID; got != want {
					t.Errorf("feature[%d].ID = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestGetItemsUnknownCollection(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetItems(context.Background(), "nope", core.Query{Limit: 10})
	if !errors.Is(err, core.ErrCollectionNotFound) {
		t.Fatalf("err = %v, want ErrCollectionNotFound", err)
	}
}

func TestGetItem(t *testing.T) {
	s := newTestStore(t)

	f, err := s.GetItem(context.Background(), "landmarks", "b")
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if f.Properties["name"] != "Hiroshima Castle" {
		t.Errorf("name = %v, want Hiroshima Castle", f.Properties["name"])
	}

	if _, err := s.GetItem(context.Background(), "landmarks", "zzz"); !errors.Is(err, core.ErrFeatureNotFound) {
		t.Fatalf("err = %v, want ErrFeatureNotFound", err)
	}
}
