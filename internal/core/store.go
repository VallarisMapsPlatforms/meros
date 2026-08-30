package core

import (
	"context"
	"errors"
)

// FeatureStore is Meros's internal contract: the interface every storage
// adapter implements. It describes what a storage can do — never how.
type FeatureStore interface {
	// GetItems returns one page of features from a collection.
	GetItems(ctx context.Context, collection string, q Query) (*FeatureCollection, error)
	// GetItem returns a single feature by id.
	GetItem(ctx context.Context, collection, id string) (*Feature, error)
}

// Sentinel errors let the API layer map storage outcomes to HTTP status
// codes without knowing any storage details.
var (
	ErrCollectionNotFound = errors.New("collection not found")
	ErrFeatureNotFound    = errors.New("feature not found")
)
