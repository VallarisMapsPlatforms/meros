package catalog

import (
	"context"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// Router implements core.FeatureStore by delegating each call to the
// adapter that owns the collection. The API layer holds ONE FeatureStore
// and never learns that routing exists — routing is a storage-side
// concern, hidden behind the same contract as everything else.
type Router struct {
	stores map[string]core.FeatureStore // collection id → adapter
}

var _ core.FeatureStore = (*Router)(nil)

// NewRouter builds a router over a collection→adapter table. The table is
// assembled by the composition root (cmd/meros) from the catalog.
func NewRouter(stores map[string]core.FeatureStore) *Router {
	return &Router{stores: stores}
}

func (r *Router) GetItems(ctx context.Context, collection string, q core.Query) (*core.FeatureCollection, error) {
	s, ok := r.stores[collection]
	if !ok {
		return nil, core.ErrCollectionNotFound
	}
	return s.GetItems(ctx, collection, q)
}

func (r *Router) GetItem(ctx context.Context, collection, id string) (*core.Feature, error) {
	s, ok := r.stores[collection]
	if !ok {
		return nil, core.ErrCollectionNotFound
	}
	return s.GetItem(ctx, collection, id)
}
