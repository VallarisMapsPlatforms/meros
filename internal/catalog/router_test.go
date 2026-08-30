package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// stubStore records which collection it was asked for.
type stubStore struct {
	label string
	got   string
}

func (s *stubStore) GetItems(_ context.Context, collection string, _ core.Query) (*core.FeatureCollection, error) {
	s.got = collection
	return &core.FeatureCollection{}, nil
}

func (s *stubStore) GetItem(_ context.Context, collection, _ string) (*core.Feature, error) {
	s.got = collection
	return &core.Feature{ID: s.label}, nil
}

func TestRouterDelegatesPerCollection(t *testing.T) {
	mongo := &stubStore{label: "mongo"}
	mem := &stubStore{label: "mem"}
	r := NewRouter(map[string]core.FeatureStore{
		"rivers":    mongo,
		"landmarks": mem,
	})

	if _, err := r.GetItems(context.Background(), "rivers", core.Query{}); err != nil {
		t.Fatal(err)
	}
	if mongo.got != "rivers" {
		t.Errorf("mongo store got %q, want rivers", mongo.got)
	}

	f, err := r.GetItem(context.Background(), "landmarks", "x")
	if err != nil {
		t.Fatal(err)
	}
	if f.ID != "mem" {
		t.Errorf("routed to %q, want mem", f.ID)
	}
}

func TestRouterUnknownCollection(t *testing.T) {
	r := NewRouter(map[string]core.FeatureStore{})
	if _, err := r.GetItems(context.Background(), "ghost", core.Query{}); !errors.Is(err, core.ErrCollectionNotFound) {
		t.Fatalf("err = %v, want ErrCollectionNotFound", err)
	}
	if _, err := r.GetItem(context.Background(), "ghost", "1"); !errors.Is(err, core.ErrCollectionNotFound) {
		t.Fatalf("err = %v, want ErrCollectionNotFound", err)
	}
}
