package runtime

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/safeguards"
	remotesessionprotection "github.com/vrooli/vrooli/internal/safeguards/remote-session-protection"
	"github.com/vrooli/vrooli/internal/tools"
	"github.com/vrooli/vrooli/internal/tools/stripe"
)

type handler = hostreqkit.Handler

var customToolHandlers = map[string]func(hostreqkit.ToolManifest) hostreqkit.Handler{
	"stripe": stripe.NewHandler,
}

var customSafeguardHandlers = map[string]func(hostreqkit.SafeguardManifest) hostreqkit.Handler{
	"remote_session_protection": remotesessionprotection.NewHandler,
}

type registry struct {
	tools      map[string]hostreqkit.Handler
	safeguards map[string]hostreqkit.Handler
}

func newRegistry(items ...hostreqkit.Handler) registry {
	r := registry{
		tools:      map[string]hostreqkit.Handler{},
		safeguards: map[string]hostreqkit.Handler{},
	}
	for _, item := range items {
		r.register(item)
	}
	return r
}

func (r *registry) register(item hostreqkit.Handler) {
	if item == nil {
		panic("runtime registry: nil handler")
	}
	name := strings.TrimSpace(item.Name())
	if name == "" {
		panic("runtime registry: handler name is required")
	}
	target := r.handlersForKind(item.Kind())
	if _, exists := target[name]; exists {
		panic(fmt.Sprintf("runtime registry: duplicate %s handler %q", item.Kind(), name))
	}
	target[name] = item
}

func (r registry) lookup(kind hostreq.Kind, name string) hostreqkit.Handler {
	return r.handlersForKind(kind)[strings.TrimSpace(name)]
}

func (r registry) names(kind hostreq.Kind) []string {
	target := r.handlersForKind(kind)
	result := make([]string, 0, len(target))
	for name := range target {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (r registry) handlersForKind(kind hostreq.Kind) map[string]hostreqkit.Handler {
	switch kind {
	case hostreq.KindSafeguard:
		return r.safeguards
	default:
		return r.tools
	}
}

var runtimeRegistry = loadRegistry()

func loadRegistry() registry {
	r := registry{
		tools:      make(map[string]hostreqkit.Handler),
		safeguards: make(map[string]hostreqkit.Handler),
	}
	loadTools(&r, tools.Manifests)
	loadSafeguards(&r, safeguards.Manifests)
	return r
}

func loadTools(r *registry, fsys fs.FS) {
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "tool.json" {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			panic(fmt.Sprintf("read tool manifest %s: %v", path, readErr))
		}
		var manifest hostreqkit.ToolManifest
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			panic(fmt.Sprintf("parse tool manifest %s: %v", path, jsonErr))
		}
		if strings.TrimSpace(manifest.Name) == "" {
			panic(fmt.Sprintf("tool manifest %s has no name", path))
		}
		var h hostreqkit.Handler
		if manifest.Handler != "" {
			ctor, ok := customToolHandlers[manifest.Handler]
			if !ok {
				panic(fmt.Sprintf("tool %q references unknown handler %q", manifest.Name, manifest.Handler))
			}
			h = ctor(manifest)
		} else {
			h = newGenericToolHandler(manifest)
		}
		r.register(h)
		return nil
	})
}

func loadSafeguards(r *registry, fsys fs.FS) {
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "safeguard.json" {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			panic(fmt.Sprintf("read safeguard manifest %s: %v", path, readErr))
		}
		var manifest hostreqkit.SafeguardManifest
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			panic(fmt.Sprintf("parse safeguard manifest %s: %v", path, jsonErr))
		}
		if strings.TrimSpace(manifest.Name) == "" {
			panic(fmt.Sprintf("safeguard manifest %s has no name", path))
		}
		ctor, ok := customSafeguardHandlers[manifest.Handler]
		if !ok {
			panic(fmt.Sprintf("safeguard %q references unknown handler %q", manifest.Name, manifest.Handler))
		}
		r.register(ctor(manifest))
		return nil
	})
}

func lookupHandler(kind hostreq.Kind, name string) hostreqkit.Handler {
	return runtimeRegistry.lookup(kind, name)
}

func HasHandler(kind hostreq.Kind, name string) bool {
	return lookupHandler(kind, name) != nil
}
