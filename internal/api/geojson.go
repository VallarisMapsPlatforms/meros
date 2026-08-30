package api

import "encoding/json"

// Wire types for OGC API - Features responses. Field names and casing
// follow the standard's JSON schemas.

// Link is an RFC 8288 style web link.
type Link struct {
	Href  string `json:"href"`
	Rel   string `json:"rel"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

const (
	contentTypeJSON    = "application/json"
	contentTypeGeoJSON = "application/geo+json"

	// crs84 is the only reference system Meros serves. The standard treats
	// it as the default when a collection declares none, but map clients
	// look for it spelled out, so it is always sent.
	crs84 = "http://www.opengis.net/def/crs/OGC/1.3/CRS84"
)

// spatialExtent is the bounding box of a collection. bbox is an array of
// arrays because the standard allows several boxes; Meros declares one.
type spatialExtent struct {
	BBox [][4]float64 `json:"bbox"`
	CRS  string       `json:"crs"`
}

type extentResponse struct {
	Spatial spatialExtent `json:"spatial"`
}

type landingResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Links       []Link `json:"links"`
}

type conformanceResponse struct {
	ConformsTo []string `json:"conformsTo"`
}

type collectionResponse struct {
	ID          string          `json:"id"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	Extent      *extentResponse `json:"extent,omitempty"`
	ItemType    string          `json:"itemType"`
	CRS         []string        `json:"crs"`
	Links       []Link          `json:"links"`
}

type collectionsResponse struct {
	Collections []collectionResponse `json:"collections"`
	Links       []Link               `json:"links"`
}

type featureResponse struct {
	Type       string          `json:"type"`
	ID         string          `json:"id,omitempty"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties map[string]any  `json:"properties"`
	Links      []Link          `json:"links,omitempty"`
}

type featureCollectionResponse struct {
	Type           string            `json:"type"`
	Features       []featureResponse `json:"features"`
	NumberReturned int               `json:"numberReturned"`
	TimeStamp      string            `json:"timeStamp"`
	Links          []Link            `json:"links"`
}

type errorResponse struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
