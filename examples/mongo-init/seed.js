// GENERATED from examples/data/hiroshima-landmarks.geojson — do not edit by hand.
// The backend-swap demo compares the two side by side, so they must stay identical.
//
// Coordinates © OpenStreetMap contributors, ODbL 1.0.
const db2 = db.getSiblingDB("meros");

db2.landmarks.drop();
db2.landmarks.insertMany([
  {_id: "hs-001", type: "Feature", geometry: {"type": "Point", "coordinates": [132.45358, 34.39547]}, properties: {"name": "Atomic Bomb Dome", "name_ja": "原爆ドーム", "category": "heritage"}},
  {_id: "hs-002", type: "Feature", geometry: {"type": "Point", "coordinates": [132.45283, 34.3919]}, properties: {"name": "Peace Memorial Park", "name_ja": "平和記念公園", "category": "park"}},
  {_id: "hs-003", type: "Feature", geometry: {"type": "Point", "coordinates": [132.45951, 34.40153]}, properties: {"name": "Hiroshima Castle", "name_ja": "広島城", "category": "castle"}},
  {_id: "hs-004", type: "Feature", geometry: {"type": "Point", "coordinates": [132.46769, 34.40041]}, properties: {"name": "Shukkei-en Garden", "name_ja": "縮景園", "category": "garden"}},
  {_id: "hs-005", type: "Feature", geometry: {"type": "Point", "coordinates": [132.47552, 34.39783]}, properties: {"name": "Hiroshima Station", "name_ja": "広島駅", "category": "transport"}},
  {_id: "hs-006", type: "Feature", geometry: {"type": "Point", "coordinates": [132.45469, 34.39566]}, properties: {"name": "Orizuru Tower", "name_ja": "おりづるタワー", "category": "viewpoint"}},
  {_id: "hs-007", type: "Feature", geometry: {"type": "Point", "coordinates": [132.48462, 34.3919]}, properties: {"name": "Mazda Zoom-Zoom Stadium", "name_ja": "マツダスタジアム", "category": "stadium"}},
  {_id: "hs-008", type: "Feature", geometry: {"type": "Point", "coordinates": [132.3193, 34.29654]}, properties: {"name": "Itsukushima Shrine", "name_ja": "厳島神社", "category": "heritage"}},
]);

// The spatial index the bbox filter relies on.
db2.landmarks.createIndex({geometry: "2dsphere"});
