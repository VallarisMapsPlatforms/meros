// Package core holds Meros's storage-neutral domain model and the
// FeatureStore contract. Nothing in this package may import a database
// driver, an HTTP framework, or a config format — the neutrality of these
// types is what keeps storage adapters interchangeable.
package core

import "encoding/json"

// BBox is a WGS84 (CRS84) bounding box in lon/lat order:
// west, south, east, north.
type BBox [4]float64

// Query is the storage-neutral read request every adapter understands.
type Query struct {
	BBox   *BBox
	Limit  int
	Offset int
}

// Feature is a single geographic feature. Geometry is carried as raw
// GeoJSON and passed through untouched — the core has no opinion on how
// geometries are encoded beyond "it is GeoJSON".
type Feature struct {
	ID         string
	Geometry   json.RawMessage
	Properties map[string]any
}

// FeatureCollection is one page of features. Counts, links, and other
// wire-format concerns belong to the protocol adapters, not here.
type FeatureCollection struct {
	Features []Feature
}

// Collection is the public metadata of one dataset as declared in the
// catalog. The /collections endpoint is a sanitized projection of this.
//
// Extent is declared metadata, not a query: map clients need it before they
// fetch anything, and the operator already knows it. Keeping it in the
// catalog spares every adapter from computing its own bounds.
type Collection struct {
	ID          string
	Title       string
	Description string
	Extent      *BBox
}
