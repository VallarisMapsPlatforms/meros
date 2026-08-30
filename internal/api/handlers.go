package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

const (
	defaultLimit = 10
	maxLimit     = 1000
)

// Conformance classes are declared only after they are actually served —
// the endpoint lists what is true, never what is planned.
var conformsTo = []string{
	"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/core",
	"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/geojson",
}

func (s *Server) landing(w http.ResponseWriter, r *http.Request) {
	base := baseURL(r)
	writeJSON(w, http.StatusOK, contentTypeJSON, landingResponse{
		Title:       s.title,
		Description: s.desc,
		Links: []Link{
			{Href: base + "/", Rel: "self", Type: contentTypeJSON, Title: "this document"},
			{Href: base + "/api", Rel: "service-desc", Type: contentTypeOpenAPI, Title: "the API definition"},
			{Href: base + "/conformance", Rel: "conformance", Type: contentTypeJSON, Title: "OGC API conformance classes"},
			{Href: base + "/collections", Rel: "data", Type: contentTypeJSON, Title: "feature collections"},
		},
	})
}

func (s *Server) apiDefinition(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, contentTypeOpenAPI, openAPIDoc(baseURL(r), s.title, s.desc))
}

func (s *Server) conformance(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, contentTypeJSON, conformanceResponse{ConformsTo: conformsTo})
}

func (s *Server) collections(w http.ResponseWriter, r *http.Request) {
	base := baseURL(r)
	cols := s.cols
	out := make([]collectionResponse, 0, len(cols))
	for _, c := range cols {
		out = append(out, s.collectionResponse(base, c))
	}
	writeJSON(w, http.StatusOK, contentTypeJSON, collectionsResponse{
		Collections: out,
		Links:       []Link{{Href: base + "/collections", Rel: "self", Type: contentTypeJSON}},
	})
}

func (s *Server) collection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	col, ok := s.byID[id]
	if !ok {
		apiError(w, http.StatusNotFound, "collection not found")
		return
	}
	writeJSON(w, http.StatusOK, contentTypeJSON, s.collectionResponse(baseURL(r), col))
}

func (s *Server) collectionResponse(base string, c core.Collection) collectionResponse {
	var extent *extentResponse
	if c.Extent != nil {
		extent = &extentResponse{Spatial: spatialExtent{
			BBox: [][4]float64{*c.Extent},
			CRS:  crs84,
		}}
	}
	return collectionResponse{
		ID:          c.ID,
		Title:       c.Title,
		Description: c.Description,
		Extent:      extent,
		ItemType:    "feature",
		CRS:         []string{crs84},
		Links: []Link{
			{Href: fmt.Sprintf("%s/collections/%s", base, c.ID), Rel: "self", Type: contentTypeJSON},
			{Href: fmt.Sprintf("%s/collections/%s/items", base, c.ID), Rel: "items", Type: contentTypeGeoJSON, Title: c.Title},
		},
	}
}

func (s *Server) items(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.byID[id]; !ok {
		apiError(w, http.StatusNotFound, "collection not found")
		return
	}

	q, err := parseQuery(r.URL.Query())
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}

	fc, err := s.store.GetItems(r.Context(), id, q)
	if err != nil {
		s.storeError(w, err)
		return
	}

	base := baseURL(r)
	links := []Link{{
		Href: base + r.URL.RequestURI(),
		Rel:  "self",
		Type: contentTypeGeoJSON,
	}}
	// A full page means there may be more; the standard requires the next
	// link only in that case. prev is a recommendation, not a requirement,
	// but a reader who can go forward should be able to come back.
	if len(fc.Features) == q.Limit {
		links = append(links, Link{
			Href:  base + pageURI(r.URL, q, q.Offset+q.Limit),
			Rel:   "next",
			Type:  contentTypeGeoJSON,
			Title: "next page",
		})
	}
	if q.Offset > 0 {
		links = append(links, Link{
			Href:  base + pageURI(r.URL, q, max(0, q.Offset-q.Limit)),
			Rel:   "prev",
			Type:  contentTypeGeoJSON,
			Title: "previous page",
		})
	}

	features := make([]featureResponse, 0, len(fc.Features))
	for _, f := range fc.Features {
		features = append(features, toFeatureResponse(f, nil))
	}
	// numberReturned is OGC wire vocabulary — it is computed here, in the
	// protocol adapter, not carried through the storage contract.
	writeJSON(w, http.StatusOK, contentTypeGeoJSON, featureCollectionResponse{
		Type:           "FeatureCollection",
		Features:       features,
		NumberReturned: len(fc.Features),
		TimeStamp:      time.Now().UTC().Format(time.RFC3339),
		Links:          links,
	})
}

func (s *Server) item(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fid := r.PathValue("fid")
	if _, ok := s.byID[id]; !ok {
		apiError(w, http.StatusNotFound, "collection not found")
		return
	}

	f, err := s.store.GetItem(r.Context(), id, fid)
	if err != nil {
		s.storeError(w, err)
		return
	}

	base := baseURL(r)
	links := []Link{
		{Href: fmt.Sprintf("%s/collections/%s/items/%s", base, id, url.PathEscape(fid)), Rel: "self", Type: contentTypeGeoJSON},
		{Href: fmt.Sprintf("%s/collections/%s", base, id), Rel: "collection", Type: contentTypeJSON},
	}
	writeJSON(w, http.StatusOK, contentTypeGeoJSON, toFeatureResponse(*f, links))
}

func toFeatureResponse(f core.Feature, links []Link) featureResponse {
	geom := f.Geometry
	if geom == nil {
		geom = json.RawMessage("null")
	}
	props := f.Properties
	if props == nil {
		props = map[string]any{}
	}
	return featureResponse{
		Type:       "Feature",
		ID:         f.ID,
		Geometry:   geom,
		Properties: props,
		Links:      links,
	}
}

// parseQuery validates the OGC query parameters Meros v1 supports.
func parseQuery(values url.Values) (core.Query, error) {
	q := core.Query{Limit: defaultLimit}

	if raw := values.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxLimit {
			return q, fmt.Errorf("query param 'limit' must be an integer between 1 and %d", maxLimit)
		}
		q.Limit = n
	}

	if raw := values.Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return q, errors.New("query param 'offset' must be a non-negative integer")
		}
		q.Offset = n
	}

	if raw := values.Get("bbox"); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) != 4 && len(parts) != 6 {
			return q, errors.New("query param 'bbox' must have 4 or 6 numbers")
		}
		nums := make([]float64, len(parts))
		for i, p := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return q, errors.New("query param 'bbox' must contain only numbers")
			}
			nums[i] = v
		}
		var b core.BBox
		if len(nums) == 4 {
			b = core.BBox{nums[0], nums[1], nums[2], nums[3]}
		} else {
			// 6 values include elevation; keep the horizontal components.
			b = core.BBox{nums[0], nums[1], nums[3], nums[4]}
		}
		q.BBox = &b
	}

	return q, nil
}

// pageURI rebuilds the current items URI at a different offset, keeping
// every other query parameter the client sent.
func pageURI(u *url.URL, q core.Query, offset int) string {
	values := u.Query()
	values.Set("offset", strconv.Itoa(offset))
	values.Set("limit", strconv.Itoa(q.Limit))
	return u.Path + "?" + values.Encode()
}

func (s *Server) storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrCollectionNotFound):
		apiError(w, http.StatusNotFound, "collection not found")
	case errors.Is(err, core.ErrFeatureNotFound):
		apiError(w, http.StatusNotFound, "feature not found")
	default:
		s.log.Error("store error", "error", err)
		apiError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, contentType string, v any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func apiError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, contentTypeJSON, errorResponse{
		Code:        strconv.Itoa(status),
		Description: msg,
	})
}

// baseURL reconstructs the externally visible origin, honoring a reverse
// proxy's forwarded protocol if present.
func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if v := r.Header.Get("X-Forwarded-Proto"); v != "" {
		scheme = v
	}
	return scheme + "://" + r.Host
}
