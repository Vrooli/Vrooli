package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	bundleruntime "scenario-to-desktop-runtime"
	"scenario-to-desktop-runtime/manifest"
)

func main() {
	var manifestPath string
	var appData string
	var bundleRoot string
	var dryRun bool

	flag.StringVar(&manifestPath, "manifest", "bundle.json", "Path to bundle.json")
	flag.StringVar(&appData, "app-data", "", "Override app data directory (defaults to OS config dir + app name)")
	flag.StringVar(&bundleRoot, "bundle-root", "", "Root directory containing bundle assets; defaults to manifest directory")
	flag.BoolVar(&dryRun, "dry-run", true, "Skip launching service binaries (control API + validation only)")
	flag.Parse()

	absManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		log.Fatalf("resolve manifest: %v", err)
	}

	m, err := manifest.LoadManifest(absManifest)
	if err != nil {
		log.Fatalf("load manifest: %v", err)
	}

	if err := m.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		log.Fatalf("validate manifest: %v", err)
	}

	if bundleRoot == "" {
		bundleRoot = filepath.Dir(absManifest)
	}

	supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
		AppDataDir: appData,
		Manifest:   m,
		BundlePath: bundleRoot,
		DryRun:     dryRun,
	})
	if err != nil {
		log.Fatalf("init supervisor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := supervisor.Start(ctx); err != nil {
		log.Fatalf("start runtime: %v", err)
	}

	fmt.Printf("runtime ready — IPC listening on %s:%d (dry-run=%v)\n", m.IPC.Host, m.IPC.Port, dryRun)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("shutdown requested")
	_ = supervisor.Shutdown(context.Background())
}
