// Package catalog loads the catalog file — the single place where
// collections are declared and bound to storage backends — and routes
// each request to the adapter that owns the collection.
package catalog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// Backend names accepted in the catalog file.
const (
	BackendFile    = "file"
	BackendMongoDB = "mongodb"
)

// Config is the parsed catalog.
type Config struct {
	Collections []CollectionConfig `yaml:"collections"`
}

// CollectionConfig declares one collection and how to reach its data.
// Credentials never live here — they come from the environment, so the
// catalog stays safe to commit.
type CollectionConfig struct {
	ID          string            `yaml:"id"`
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Extent      []float64         `yaml:"extent"` // west, south, east, north (CRS84)
	Backend     string            `yaml:"backend"`
	Source      map[string]string `yaml:"source"`
}

// Load reads and validates a catalog file.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse catalog %s: %w", path, err)
	}
	if len(cfg.Collections) == 0 {
		return nil, fmt.Errorf("catalog %s declares no collections", path)
	}
	seen := make(map[string]bool, len(cfg.Collections))
	for _, c := range cfg.Collections {
		switch {
		case c.ID == "":
			return nil, fmt.Errorf("catalog: collection with empty id")
		case seen[c.ID]:
			return nil, fmt.Errorf("catalog: duplicate collection id %q", c.ID)
		case c.Backend != BackendFile && c.Backend != BackendMongoDB:
			return nil, fmt.Errorf("catalog: collection %q: unknown backend %q", c.ID, c.Backend)
		case len(c.Extent) != 0 && len(c.Extent) != 4:
			return nil, fmt.Errorf("catalog: collection %q: extent needs 4 numbers (west, south, east, north), got %d", c.ID, len(c.Extent))
		}
		seen[c.ID] = true
	}
	return &cfg, nil
}

// Get returns the declaration of one collection.
func (c *Config) Get(id string) (CollectionConfig, bool) {
	for _, col := range c.Collections {
		if col.ID == id {
			return col, true
		}
	}
	return CollectionConfig{}, false
}

// Metadata returns the public projection of the catalog — what the
// /collections endpoint serves, with all storage detail removed.
func (c *Config) Metadata() []core.Collection {
	out := make([]core.Collection, 0, len(c.Collections))
	for _, col := range c.Collections {
		meta := core.Collection{
			ID:          col.ID,
			Title:       col.Title,
			Description: col.Description,
		}
		if len(col.Extent) == 4 {
			meta.Extent = &core.BBox{col.Extent[0], col.Extent[1], col.Extent[2], col.Extent[3]}
		}
		out = append(out, meta)
	}
	return out
}
