package installgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/modfile"

	"github.com/vrooli/envkit-go"
)

// A repository package (a package of a module in this checkout) is used in
// place through a replace directive, so what an install can add to the disk is
// the external modules that package's import closure needs. The go command
// answers which those are: `go list -deps` run in the owning module, offline and
// read-only, names the module behind every package it can load. Modules the
// cache already holds cost nothing; a package it cannot load because its module
// is not cached is mapped to the module the owner's go.mod requires and sized
// from the proxy exactly like a published module.

const (
	goListTimeout = 2 * time.Minute
	// goRepositoryInstallAllowance covers the go.mod and go.sum rewrite and the
	// module-cache bookkeeping of an install that downloads nothing.
	goRepositoryInstallAllowance int64 = 1 << 20
	maxGoListBytes                     = 64 << 20
)

// goListRunner runs the go command; tests replace it.
type goListRunner func(ctx context.Context, dir string, env []string, args ...string) ([]byte, error)

var runGoList goListRunner = func(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = envkit.Toolchain(envkit.Env(env), envkit.ToolchainOptions{})
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdout, limit: maxGoListBytes}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

type limitedWriter struct {
	w     io.Writer
	limit int
	n     int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n+len(p) > l.limit {
		return 0, fmt.Errorf("go list output exceeds %d bytes", l.limit)
	}
	l.n += len(p)
	return l.w.Write(p)
}

type goListPackage struct {
	ImportPath string
	Standard   bool
	Module     *struct {
		Path    string
		Version string
		Main    bool
		Replace *struct{ Path string }
	}
	Error *struct{ Err string }
}

// externalModulesFor lists the published modules (path and version) that
// building pkg in the owner module needs.
func externalModulesFor(ctx context.Context, ownerDir string, owner *modfile.File, pkg, modCache string) (map[string]string, error) {
	cache := modCache
	env := overlayInstallEnvironment(os.Environ(), []string{
		"GOMODCACHE=" + cache, "GOPROXY=off", "GOFLAGS=-mod=readonly", "GOWORK=off", "GOTOOLCHAIN=local", "GO111MODULE=on",
	})
	runCtx, cancel := context.WithTimeout(ctx, goListTimeout)
	defer cancel()
	pattern := pkg
	if owner.Module != nil && pkg == owner.Module.Mod.Path {
		// Naming the module itself: its root need not be a package, so cover
		// every package the module holds.
		pattern += "/..."
	}
	out, err := runGoList(runCtx, ownerDir, env, "list", "-e", "-deps", "-json=ImportPath,Standard,Module,Error", pattern)
	if err != nil {
		return nil, fmt.Errorf("go list could not load the import closure of %s in %s offline: %w", pkg, ownerDir, err)
	}
	replaced := map[string]bool{}
	required := map[string]string{}
	for _, req := range owner.Require {
		required[req.Mod.Path] = req.Mod.Version
	}
	for _, rep := range owner.Replace {
		replaced[rep.Old.Path] = true
	}
	modules := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var listed goListPackage
		if err := decoder.Decode(&listed); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		if listed.Standard {
			continue
		}
		if listed.Module != nil {
			if listed.Module.Main || listed.Module.Replace != nil || replaced[listed.Module.Path] {
				continue // used in place
			}
			if listed.Error == nil {
				modules[listed.Module.Path] = listed.Module.Version
				continue
			}
		}
		if listed.Error == nil {
			continue
		}
		// The package could not be loaded, almost always because its module is
		// not in the cache. The owner's go.mod says which module provides it.
		path, version := longestRequiredPrefix(required, listed.ImportPath)
		if path == "" {
			return nil, fmt.Errorf("package %s (imported by %s) cannot be loaded and no module in %s provides it: %s", listed.ImportPath, pkg, owner.Module.Mod.Path, strings.TrimSpace(listed.Error.Err))
		}
		if replaced[path] {
			continue
		}
		modules[path] = version
	}
	return modules, nil
}

func longestRequiredPrefix(required map[string]string, importPath string) (path, version string) {
	for candidate, v := range required {
		if (importPath == candidate || strings.HasPrefix(importPath, candidate+"/")) && len(candidate) > len(path) {
			path, version = candidate, v
		}
	}
	return path, version
}

// probeRepositoryPackageGrowth sizes what installing a repository package adds:
// the external modules its import closure needs, minus what the module cache
// and the consuming go.mod already provide.
func probeRepositoryPackageGrowth(ctx context.Context, r Resolution, ownerDir string, owner, consumer *modfile.File, proxy, modCache string, client *http.Client) (goModuleProbeResult, error) {
	modules, err := externalModulesFor(ctx, ownerDir, owner, r.PackageName, modCache)
	if err != nil {
		return goModuleProbeResult{}, err
	}
	required := make(map[string]string, len(consumer.Require))
	for _, req := range consumer.Require {
		required[req.Mod.Path] = req.Mod.Version
	}
	walk := &externalGoWalk{ctx: ctx, proxy: proxy, client: client, required: required, modCache: modCache, listed: true, seen: map[string]bool{}}
	paths := make([]string, 0, len(modules))
	for path := range modules {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := walk.visit(path, modules[path], false); err != nil {
			return goModuleProbeResult{}, err
		}
	}
	walk.result.MetadataBytes += goRepositoryInstallAllowance
	return walk.result, nil
}
