package api

// The API definition served at /api. OGC API - Features Part 1 requires the
// landing page to link to one (Requirement 2), and clients such as QGIS stop
// at the landing page when the link is absent.
//
// It is written by hand rather than generated so that it stays exactly as
// honest as the server: only the paths Meros serves, only the query
// parameters it actually reads.

const contentTypeOpenAPI = "application/vnd.oai.openapi+json;version=3.0"

func openAPIDoc(base, title, description string) map[string]any {
	jsonResponse := func(desc string) map[string]any {
		return map[string]any{
			"description": desc,
			"content": map[string]any{
				contentTypeJSON: map[string]any{"schema": map[string]any{"type": "object"}},
			},
		}
	}
	geoJSONResponse := func(desc string) map[string]any {
		return map[string]any{
			"description": desc,
			"content": map[string]any{
				contentTypeGeoJSON: map[string]any{"schema": map[string]any{"type": "object"}},
			},
		}
	}
	pathParam := func(name, desc string) map[string]any {
		return map[string]any{
			"name": name, "in": "path", "required": true, "description": desc,
			"schema": map[string]any{"type": "string"},
		}
	}
	notFound := jsonResponse("The requested resource does not exist.")
	badRequest := jsonResponse("A query parameter was malformed.")

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       title,
			"description": description,
			"version":     "1.0.0",
			"license":     map[string]any{"name": "Apache-2.0"},
		},
		"servers": []any{map[string]any{"url": base}},
		"paths": map[string]any{
			"/": map[string]any{"get": map[string]any{
				"summary":     "Landing page",
				"operationId": "getLandingPage",
				"tags":        []any{"Capabilities"},
				"responses":   map[string]any{"200": jsonResponse("Links to the API definition, the conformance declaration and the collections.")},
			}},
			"/api": map[string]any{"get": map[string]any{
				"summary":     "This API definition",
				"operationId": "getApiDefinition",
				"tags":        []any{"Capabilities"},
				"responses":   map[string]any{"200": jsonResponse("This document.")},
			}},
			"/conformance": map[string]any{"get": map[string]any{
				"summary":     "Conformance classes this server implements",
				"operationId": "getConformanceDeclaration",
				"tags":        []any{"Capabilities"},
				"responses":   map[string]any{"200": jsonResponse("The list of conformance classes, and nothing that is merely planned.")},
			}},
			"/collections": map[string]any{"get": map[string]any{
				"summary":     "Feature collections offered by this server",
				"operationId": "getCollections",
				"tags":        []any{"Capabilities"},
				"responses":   map[string]any{"200": jsonResponse("The collections declared in the catalog.")},
			}},
			"/collections/{collectionId}": map[string]any{"get": map[string]any{
				"summary":     "Describe one collection",
				"operationId": "describeCollection",
				"tags":        []any{"Capabilities"},
				"parameters":  []any{pathParam("collectionId", "Identifier of the collection.")},
				"responses":   map[string]any{"200": jsonResponse("Metadata about the collection."), "404": notFound},
			}},
			"/collections/{collectionId}/items": map[string]any{"get": map[string]any{
				"summary":     "Fetch features from a collection",
				"operationId": "getFeatures",
				"tags":        []any{"Data"},
				"parameters": []any{
					pathParam("collectionId", "Identifier of the collection."),
					map[string]any{
						"name": "limit", "in": "query", "required": false,
						"description": "Maximum number of features to return.",
						"schema":      map[string]any{"type": "integer", "minimum": 1, "maximum": maxLimit, "default": defaultLimit},
					},
					map[string]any{
						"name": "offset", "in": "query", "required": false,
						"description": "Number of features to skip.",
						"schema":      map[string]any{"type": "integer", "minimum": 0, "default": 0},
					},
					map[string]any{
						"name": "bbox", "in": "query", "required": false,
						"description": "Only features intersecting this bounding box, in CRS84: west,south,east,north.",
						"explode":     false,
						"schema": map[string]any{
							"type": "array", "minItems": 4, "maxItems": 6,
							"items": map[string]any{"type": "number"},
						},
					},
				},
				"responses": map[string]any{"200": geoJSONResponse("One page of features."), "400": badRequest, "404": notFound},
			}},
			"/collections/{collectionId}/items/{featureId}": map[string]any{"get": map[string]any{
				"summary":     "Fetch a single feature",
				"operationId": "getFeature",
				"tags":        []any{"Data"},
				"parameters": []any{
					pathParam("collectionId", "Identifier of the collection."),
					pathParam("featureId", "Identifier of the feature."),
				},
				"responses": map[string]any{"200": geoJSONResponse("The feature."), "404": notFound},
			}},
		},
	}
}
