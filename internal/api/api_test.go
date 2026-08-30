package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VallarisMapsPlatforms/meros/internal/catalog"
	"github.com/VallarisMapsPlatforms/meros/internal/core"
	"github.com/VallarisMapsPlatforms/meros/internal/store/file"
)

const sampleGeoJSON = `{
	"type": "FeatureCollection",
	"features": [
		{"type": "Feature", "id": "a", "geometry": {"type": "Point", "coordinates": [132.4536, 34.3956]}, "properties": {"name": "Atomic Bomb Dome"}},
		{"type": "Feature", "id": "b", "geometry": {"type": "Point", "coordinates": [132.4593, 34.4027]}, "properties": {"name": "Hiroshima Castle"}},
		{"type": "Feature", "id": "c", "geometry": {"type": "Point", "coordinates": [132.3198, 34.2959]}, "properties": {"name": "Itsukushima Shrine"}}
	]
}`

// newTestServer wires the full read path — API over router over the
// file store — with no network dependencies. The test suite is the
// system's second driver: it exercises the same FeatureStore contract
// the HTTP layer uses.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	store, err := file.NewFromGeoJSON(map[string][]byte{"landmarks": []byte(sampleGeoJSON)})
	if err != nil {
		t.Fatal(err)
	}
	// Collection metadata reaches the API as plain core values — the HTTP
	// layer has no idea a catalog file, or a storage backend, exists.
	cols := []core.Collection{{
		ID:          "landmarks",
		Title:       "Hiroshima Landmarks",
		Description: "test data",
	}}
	router := catalog.NewRouter(map[string]core.FeatureStore{"landmarks": store})

	ts := httptest.NewServer(New(cols, router, WithServiceInfo("Meros Test", "test instance")).Handler())
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, url string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return resp, body
}

func TestEndpoints(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantType    string
		wantContain []string
	}{
		{"landing", "/", http.StatusOK, "application/json", []string{`"service-desc"`, `"conformance"`, `"data"`}},
		{"api definition", "/api", http.StatusOK, contentTypeOpenAPI, []string{`"openapi"`, `/collections/{collectionId}/items`}},
		{"conformance", "/conformance", http.StatusOK, "application/json", []string{"ogcapi-features-1/1.0/conf/core"}},
		{"collections", "/collections", http.StatusOK, "application/json", []string{`"landmarks"`, `"itemType":"feature"`}},
		{"collection", "/collections/landmarks", http.StatusOK, "application/json", []string{`"Hiroshima Landmarks"`}},
		{"collection 404", "/collections/ghost", http.StatusNotFound, "application/json", []string{"collection not found"}},
		{"items", "/collections/landmarks/items", http.StatusOK, "application/geo+json", []string{`"FeatureCollection"`, `"numberReturned":3`}},
		{"items bbox", "/collections/landmarks/items?bbox=132.44,34.38,132.49,34.41", http.StatusOK, "application/geo+json", []string{`"numberReturned":2`, "Atomic Bomb Dome"}},
		{"items bad bbox", "/collections/landmarks/items?bbox=1,2,3", http.StatusBadRequest, "application/json", []string{"bbox"}},
		{"items bad limit", "/collections/landmarks/items?limit=0", http.StatusBadRequest, "application/json", []string{"limit"}},
		{"item", "/collections/landmarks/items/b", http.StatusOK, "application/geo+json", []string{"Hiroshima Castle"}},
		{"item 404", "/collections/landmarks/items/zzz", http.StatusNotFound, "application/json", []string{"feature not found"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := get(t, ts.URL+tt.path)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", resp.StatusCode, tt.wantStatus, body)
			}
			if ct := resp.Header.Get("Content-Type"); ct != tt.wantType {
				t.Errorf("content-type = %q, want %q", ct, tt.wantType)
			}
			for _, want := range tt.wantContain {
				if !strings.Contains(string(body), want) {
					t.Errorf("body missing %q\nbody: %s", want, body)
				}
			}
		})
	}
}

func TestPaginationNextLink(t *testing.T) {
	ts := newTestServer(t)

	resp, body := get(t, ts.URL+"/collections/landmarks/items?limit=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var fc struct {
		NumberReturned int `json:"numberReturned"`
		Links          []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
	}
	if err := json.Unmarshal(body, &fc); err != nil {
		t.Fatal(err)
	}
	if fc.NumberReturned != 2 {
		t.Fatalf("numberReturned = %d, want 2", fc.NumberReturned)
	}
	var next string
	for _, l := range fc.Links {
		if l.Rel == "next" {
			next = l.Href
		}
	}
	if next == "" {
		t.Fatal("no next link on a full page")
	}
	if !strings.Contains(next, "offset=2") {
		t.Errorf("next link %q missing offset=2", next)
	}
}

func TestPaginationPrevLink(t *testing.T) {
	ts := newTestServer(t)

	links := func(path string) map[string]string {
		resp, body := get(t, ts.URL+path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d for %s", resp.StatusCode, path)
		}
		var fc struct {
			Links []struct{ Rel, Href string } `json:"links"`
		}
		if err := json.Unmarshal(body, &fc); err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, l := range fc.Links {
			out[l.Rel] = l.Href
		}
		return out
	}

	if _, ok := links("/collections/landmarks/items?limit=2")["prev"]; ok {
		t.Error("first page should have no prev link")
	}

	second := links("/collections/landmarks/items?limit=2&offset=2")
	prev, ok := second["prev"]
	if !ok {
		t.Fatal("second page has no prev link")
	}
	if !strings.Contains(prev, "offset=0") {
		t.Errorf("prev link %q should point back to offset=0", prev)
	}
}
