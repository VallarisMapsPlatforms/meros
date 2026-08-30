// Command meros serves feature collections declared in a catalog file
// over OGC API - Features (read-only, Part 1).
//
// This is the composition root: the only place that knows every concrete
// type. It reads the catalog, builds one adapter per backend, and hands the
// API two neutral values — the collection metadata and a core.FeatureStore.
// Configuration becomes domain values here so that no package below ever
// learns the config format, or that a config file existed.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VallarisMapsPlatforms/meros/internal/api"
	"github.com/VallarisMapsPlatforms/meros/internal/catalog"
	"github.com/VallarisMapsPlatforms/meros/internal/core"
	"github.com/VallarisMapsPlatforms/meros/internal/store/file"
	"github.com/VallarisMapsPlatforms/meros/internal/store/mongodb"
)

func main() {
	var (
		configPath = flag.String("config", "catalog.yml", "path to the catalog file")
		addr       = flag.String("addr", ":8080", "listen address")
	)
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(*configPath, *addr, log); err != nil {
		log.Error("meros exited", "error", err)
		os.Exit(1)
	}
}

func run(configPath, addr string, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := catalog.Load(configPath)
	if err != nil {
		return err
	}

	// Group collections by backend so each adapter is constructed once.
	fileSources := map[string]string{}
	mongoSources := map[string]mongodb.Source{}
	for _, c := range cfg.Collections {
		switch c.Backend {
		case catalog.BackendFile:
			path := c.Source["path"]
			if path == "" {
				return fmt.Errorf("collection %q: file backend needs source.path", c.ID)
			}
			fileSources[c.ID] = path
		case catalog.BackendMongoDB:
			src := mongodb.Source{
				Database:      c.Source["database"],
				Collection:    c.Source["collection"],
				GeometryField: c.Source["geometry_field"],
			}
			if src.Database == "" || src.Collection == "" {
				return fmt.Errorf("collection %q: mongodb backend needs source.database and source.collection", c.ID)
			}
			mongoSources[c.ID] = src
		}
	}

	stores := map[string]core.FeatureStore{}

	if len(fileSources) > 0 {
		fileStore, err := file.New(fileSources)
		if err != nil {
			return err
		}
		for id := range fileSources {
			stores[id] = fileStore
		}
	}

	if len(mongoSources) > 0 {
		// Credentials come from the environment, never from the catalog.
		// The default matches the local docker-compose instance.
		uri := os.Getenv("MEROS_MONGODB_URI")
		if uri == "" {
			uri = "mongodb://localhost:27017"
		}
		connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		mg, err := mongodb.Connect(connectCtx, uri, mongoSources)
		cancel()
		if err != nil {
			return err
		}
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = mg.Close(closeCtx)
		}()
		for id := range mongoSources {
			stores[id] = mg
		}
	}

	router := catalog.NewRouter(stores)
	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(cfg.Metadata(), router, api.WithLogger(log)).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("meros serving",
		"addr", addr,
		"catalog", configPath,
		"collections", len(cfg.Collections),
	)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("meros stopped")
	return nil
}
