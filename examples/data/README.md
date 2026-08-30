# Demo data

Two small datasets, enough to make the API do something real.

> **© OpenStreetMap contributors.** Both files take their geometry from
> OpenStreetMap and are therefore licensed under the **Open Database License
> (ODbL) 1.0**, not the Apache-2.0 licence that covers the rest of this
> repository. If you reuse them, keep the attribution and the ODbL terms. See
> <https://www.openstreetmap.org/copyright>.

## `hiroshima-landmarks.geojson`

Eight landmark points around Hiroshima. Coordinates come from OpenStreetMap —
the feature itself in each case (`relation 1506282` for the Atomic Bomb Dome,
`way 145661233` for the castle, and so on), never a nearby information board.
Peace Memorial Park keeps a hand-chosen point inside the park, because OSM
offers no single park polygon under that name.

The ids, English names and categories were written for this demo; only the
coordinates are OSM's.

`examples/mongo-init/seed.js` is **generated from this file** and seeds the same
features into MongoDB, so the two backends can be asked the same question and
give byte-identical answers. Regenerate it whenever this file changes.

## `hiroden-lines.geojson`

The seven named passenger lines of the Hiroshima Electric Railway (広島電鉄),
as **MultiLineString** features.

> **© OpenStreetMap contributors.** This file is derived from OpenStreetMap and
> is therefore licensed under the **Open Database License (ODbL) 1.0**, not the
> Apache-2.0 licence that covers the rest of this repository. If you reuse it,
> keep the attribution and the ODbL terms. See
> <https://www.openstreetmap.org/copyright>.

Retrieved from the Overpass API with:

```
[out:json][timeout:90];
way["railway"="tram"](34.33,132.40,34.42,132.49);
out geom;
```

Then processed: only ways carrying a `name` were kept — the rest are depot
(`service=yard`) and crossover track, not passenger route. Ways were grouped by
line name into one MultiLineString each, and coordinates rounded to six decimal
places (about 10 cm), which keeps the file readable without any visible loss.

To refresh it, re-run the query above and repeat those steps.
