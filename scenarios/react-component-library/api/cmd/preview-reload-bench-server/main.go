//go:build previewreloadbench

// Command preview-reload-bench-server serves one synthetic component from a
// temporary source root through the production preview and harness handlers.
// It exists only to support tools/bench-preview-reload.py; no repository
// library files or persistent scenario databases are touched.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/schedule"
	componentsH "react-component-library/handlers/components"
	previewH "react-component-library/handlers/preview"
	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	internalpreview "react-component-library/internal/preview"
)

const (
	componentName = "ReloadBenchmark"
	libraryID     = "react-component-library:ReloadBenchmark"
	sourceBytes   = 50 * 1024
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	tempRoot, err := os.OpenRoot(os.TempDir())
	if err != nil {
		return fmt.Errorf("open temporary directory root: %w", err)
	}
	defer tempRoot.Close()

	root, err := os.MkdirTemp(os.TempDir(), "rcl-preview-reload-bench-")
	if err != nil {
		return fmt.Errorf("create temporary component root: %w", err)
	}
	defer func() { _ = tempRoot.RemoveAll(filepath.Base(root)) }()
	fixtureRoot, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("open temporary component root: %w", err)
	}
	defer fixtureRoot.Close()

	if err := fixtureRoot.MkdirAll(filepath.Join("components", componentName, "versions", "1.0.0"), 0o700); err != nil {
		return fmt.Errorf("create temporary fixture: %w", err)
	}
	if err := fixtureRoot.WriteFile(filepath.Join("components", componentName, "component.json"), []byte(`{
  "libraryId": "`+libraryID+`",
  "displayName": "Preview reload benchmark",
  "description": "Disposable 50 KiB latency fixture",
  "tags": [],
  "latest": "1.0.0",
  "deprecatedVersions": []
}`), 0o600); err != nil {
		return fmt.Errorf("write temporary manifest: %w", err)
	}
	if err := fixtureRoot.WriteFile(filepath.Join("components", componentName, "versions", "1.0.0", componentName+".tsx"), []byte(syntheticSource()), 0o600); err != nil {
		return fmt.Errorf("write temporary component source: %w", err)
	}
	if err := fixtureRoot.WriteFile(filepath.Join("components", componentName, "versions", "1.0.0", "story.tsx"), []byte(`import React from "react";
import { ReloadBenchmark } from "./ReloadBenchmark";
export function Default() { return React.createElement(ReloadBenchmark); }
`), 0o600); err != nil {
		return fmt.Errorf("write temporary story harness: %w", err)
	}
	if err := fixtureRoot.WriteFile(filepath.Join("components", componentName, "versions", "1.0.0", "story.json"), []byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "title": "Preview reload benchmark",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{
    "id": "default",
    "name": "Synthetic component",
    "mode": "live",
    "args": {},
    "composition": {"specimen": {"module": "./story.tsx", "export": "Default"}}
  }]
}`), 0o600); err != nil {
		return fmt.Errorf("write temporary story contract: %w", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(root, "benchmark.db"))
	if err != nil {
		return fmt.Errorf("open temporary SQLite database: %w", err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalcomponents.Schema),
	); err != nil {
		return fmt.Errorf("initialize temporary schema: %w", err)
	}
	assets, repo := componentsH.BuildService(db, schedule.System(), root)
	indexer := internalcomponents.NewIndexer(repo, root, nil)
	if _, err := indexer.Run(ctx); err != nil {
		return fmt.Errorf("index temporary component: %w", err)
	}
	component, err := assets.GetByLibraryID(ctx, libraryID)
	if err != nil {
		return fmt.Errorf("resolve temporary component: %w", err)
	}
	if component.ID == "" {
		return fmt.Errorf("temporary component has no registry ID")
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve benchmark server repository root")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../.."))
	previewService := internalpreview.NewServiceWithDepsAtRoot(
		assets,
		internalpreview.NewEsbuilderWithRoot(root),
		nil,
		repoRoot,
	)
	router := mux.NewRouter()
	previewH.ModuleFromService(
		previewService,
		assets,
		log.New(os.Stderr, "preview-benchmark: ", log.LstdFlags),
		repoRoot,
	).Mount(router)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	server := &http.Server{Handler: router, ReadHeaderTimeout: 3 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"url":          "http://" + listener.Addr().String(),
		"asset_id":     libraryID,
		"source_bytes": sourceBytes,
	}); err != nil {
		return fmt.Errorf("write benchmark server descriptor: %w", err)
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve temporary preview: %w", err)
	}
	return nil
}

func syntheticSource() string {
	prefix := `/** @libraryId ` + libraryID + `
 * @version 1.0.0
 */
import React from "react";
const padding = "`
	suffix := `";
export const ReloadBenchmark = () => React.createElement("div", { "data-testid": "synthetic-component" }, padding.slice(0, 32));
`
	filler := sourceBytes - len(prefix) - len(suffix)
	if filler < 0 {
		panic("synthetic source prefix exceeds 50 KiB")
	}
	source := prefix + strings.Repeat("x", filler) + suffix
	if len([]byte(source)) != sourceBytes {
		panic("synthetic source is not exactly 50 KiB")
	}
	return source
}
