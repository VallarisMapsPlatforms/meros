# Meros

**A composable, read-only [OGC API - Features](https://ogcapi.ogc.org/features/) server in Go.**

Every collection declares its storage in a catalog file. The API speaks one public
contract (OGC API - Features) to clients and one internal contract (`FeatureStore`)
to storage — so a backend can be added or swapped without touching anything above it.

> **What this is:** the code behind a talk at
> [FOSS4G 2026 (Hiroshima)](https://talks.osgeo.org/foss4g-2026/talk/SZ7AKX/).
> The MongoDB adapter is based on the read path of
> [Vallaris Maps](https://vallarismaps.com), a geospatial platform in production;
> the architecture around it was written for this repository.
> Read-only, OGC API - Features Part 1, two conformance classes, two storage adapters.

## Quickstart (no database needed)

Needs Go 1.22 or newer.

```bash
go run ./cmd/meros -config examples/catalog-quickstart.yml
```

```bash
curl localhost:8080/                                   # landing page
curl localhost:8080/api                                # the API definition
curl localhost:8080/conformance                        # what this server actually conforms to
curl localhost:8080/collections                        # collections from the catalog
curl "localhost:8080/collections/landmarks/items?bbox=132.45,34.39,132.46,34.40&limit=5"
curl "localhost:8080/collections/hiroden/items?limit=3"
```

Or open QGIS → *Layer → Add Layer → Add WFS / OGC API - Features* → `http://localhost:8080`.
Set **Version** to *OGC API - Features*; on auto-detect QGIS tries classic WFS first and gives up.

## Two storages behind one API

```bash
docker compose up -d          # local MongoDB, seeds itself with the demo landmarks
go run ./cmd/meros -config examples/catalog.yml
```

That catalog declares three collections, and the `backend:` line is the only thing
that differs between where they live:

| Collection | Storage | Geometry |
|---|---|---|
| `landmarks-mongodb` | MongoDB | points |
| `landmarks-file` | a GeoJSON file | points — the same ones |
| `hiroden` | a GeoJSON file | tram lines |

Ask the first two the same question and you get the same features back; the only
difference is the collection name in the links. One server, one catalog, two
databases, and nothing above the `backend:` line knows which is which.

## Poking at it by hand

[`examples/meros.postman_collection.json`](examples/meros.postman_collection.json)
follows the same path in Postman, with assertions on each step — the landing
page carries the links the standard requires, `/conformance` claims only two
classes, the two landmark collections return identical features, paging offers
`next` and `prev` in the right places, and the error cases name what went wrong.
Import it, set `baseUrl`, and run the folders in order.

It also runs headless:

```bash
newman run examples/meros.postman_collection.json
```

## Architecture

```
Clients (QGIS · browsers · scripts)
   │
OGC API layer          ← public contract (stable for clients)
   │
FeatureStore           ← internal contract (stable for storage)
   │
Catalog / Router       ← "which storage owns this collection?"  (catalog.yml)
   │
Storage adapters       ← MongoDB · GeoJSON file (read off disk)
```

- `internal/core` — the domain model and the `FeatureStore` contract. Imports no
  driver, no framework, no config format.
- `internal/api` — the HTTP adapter. Imports `core` and nothing else: not the
  storage adapters, not the catalog, not the config format.
- `internal/catalog` — catalog loading + a router that itself implements
  `FeatureStore`, so the API never learns that routing exists.
- `internal/store/file` — GeoJSON files, read from disk at startup.
- `internal/store/mongodb` — MongoDB. All MongoDB vocabulary lives here and nowhere else.
- `cmd/meros` — the composition root; the only place that knows every concrete type.

## Configuration

The catalog registers every collection and how to reach it. Credentials never live
in the catalog — the MongoDB URI comes from `MEROS_MONGODB_URI`.

```yaml
collections:
  - id: landmarks
    title: Hiroshima Landmarks
    backend: mongodb            # or: file
    source:
      database: meros
      collection: landmarks
      # geometry_field: geometry   (default)
```

## Adding a storage backend

Implement one interface with two methods:

```go
type FeatureStore interface {
    GetItems(ctx context.Context, collection string, q Query) (*FeatureCollection, error)
    GetItem(ctx context.Context, collection, id string) (*Feature, error)
}
```

Neutral types only — no driver vocabulary in the contract. Register the backend in
`cmd/meros` and declare it in the catalog. Nothing in the OGC layer or the core changes.

## Tests

```bash
go test ./...                                            # unit tests, no database
MEROS_TEST_MONGODB_URI=mongodb://localhost:27017 \
  go test ./internal/store/mongodb/                      # adapter integration test
```

## What it does not do

Writes and transactions (Part 4) · CQL2 filtering (Part 3) · coordinate reference
systems beyond CRS84 (Part 2) · tiles, maps, records. `/conformance` lists only
what the server actually does, which is why it declares two classes and not more.

## Licence and data

Apache-2.0 for the code.

Everything under `examples/data/` takes its geometry from OpenStreetMap and carries
OSM's licence instead — **© OpenStreetMap contributors, ODbL 1.0**. Provenance and
the Overpass queries that produced it are in
[examples/data/README.md](examples/data/README.md).
