package adoptions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"connectrpc.com/connect"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
	"react-component-library/internal/uimanifest"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
)

// SlotReader looks up the slot declared by a component's library manifest.
// Returns empty string when the component declares no slot (caller will fall
// back to the manifest's defaults.slot).
type SlotReader interface {
	Slot(ctx context.Context, componentID string) (string, error)
}

// FSSlotReader reads `library/components/<slug>/component.json` from disk via
// the existing components service. Decouples the handler from any future slot
// persistence in the DB.
type FSSlotReader struct {
	Components  components.Service
	LibraryRoot string
}

// Slot resolves componentID → slug (via components.Service.Get) → reads
// `<libraryRoot>/components/<slug>/component.json` and returns the `slot`
// field. Empty string when the field is absent.
func (r *FSSlotReader) Slot(ctx context.Context, componentID string) (string, error) {
	if r == nil || r.Components == nil {
		return "", errors.New("slot reader: components service not configured")
	}
	c, err := r.Components.Get(ctx, componentID)
	if err != nil {
		return "", err
	}
	if r.LibraryRoot == "" || c.Slug == "" {
		return "", nil
	}
	path := filepath.Join(r.LibraryRoot, "components", c.Slug, "component.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read component manifest %q: %w", path, err)
	}
	var mf struct {
		Slot string `json:"slot"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		return "", fmt.Errorf("parse component manifest %q: %w", path, err)
	}
	return mf.Slot, nil
}

// ResolveAdoptionPath implements the new RPC. The resolver itself is in
// internal/adoptions/pathresolver.go; this handler glues request → resolver
// → response.
func (h *connectHandler) ResolveAdoptionPath(ctx context.Context, req *connect.Request[adoptionsv1.ResolveAdoptionPathRequest]) (*connect.Response[adoptionsv1.ResolveAdoptionPathResponse], error) {
	if h.deps.Resolver == nil || h.deps.SlotReader == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ResolveAdoptionPath: resolver not configured"))
	}
	// Look up the component's name + slot. We need ComponentName for token
	// substitution; an empty name is a request error.
	comp, err := h.deps.Library.Get(ctx, req.Msg.ComponentId)
	if err != nil {
		return nil, adoptions.ToConnectError(err)
	}
	slot, err := h.deps.SlotReader.Slot(ctx, req.Msg.ComponentId)
	if err != nil {
		h.deps.Logger.Printf("ResolveAdoptionPath: slot lookup failed for %q: %v", req.Msg.ComponentId, err)
		// Fall through with empty slot; resolver will use defaults.slot.
		slot = ""
	}

	// Prefer Slug (PascalCase, file-safe) over DisplayName (may contain
	// spaces). Slug falls back to DisplayName for legacy rows without a slug.
	name := comp.Slug
	if name == "" {
		name = comp.DisplayName
	}
	out, err := h.deps.Resolver.Resolve(adoptions.ResolveInput{
		ComponentSlot: slot,
		ComponentName: name,
		Scenario:      req.Msg.Scenario,
		Override:      req.Msg.OverridePath,
		Feature:       req.Msg.Feature,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&adoptionsv1.ResolveAdoptionPathResponse{
		Path:     out.Path,
		Source:   toProtoSource(out.Source),
		Slot:     out.Slot,
		Warnings: out.Warnings,
	}), nil
}

func toProtoSource(s adoptions.ResolveSource) adoptionsv1.ResolveSource {
	switch s {
	case adoptions.SourceExplicit:
		return adoptionsv1.ResolveSource_RESOLVE_SOURCE_EXPLICIT
	case adoptions.SourceTemplateManifest:
		return adoptionsv1.ResolveSource_RESOLVE_SOURCE_TEMPLATE_MANIFEST
	case adoptions.SourceHeuristic:
		return adoptionsv1.ResolveSource_RESOLVE_SOURCE_HEURISTIC
	case adoptions.SourceFallback:
		return adoptionsv1.ResolveSource_RESOLVE_SOURCE_FALLBACK
	}
	return adoptionsv1.ResolveSource_RESOLVE_SOURCE_UNSPECIFIED
}

// BuildResolver constructs a Resolver from a repo root. Prefers the
// `VROOLI_SOURCE_ROOT` env var (set by the scenario lifecycle) when the
// provided fallback is empty or unreadable — `runtime.Caller`-derived paths
// can be unreliable in trimpath builds, but `VROOLI_SOURCE_ROOT` is always
// set to the actual checkout root.
func BuildResolver(fallbackRepoRoot string) *adoptions.Resolver {
	repoRoot := fallbackRepoRoot
	if env := os.Getenv("VROOLI_SOURCE_ROOT"); env != "" {
		repoRoot = env
	} else if env := os.Getenv("VROOLI_ROOT"); env != "" {
		repoRoot = env
	}
	return adoptions.NewResolver(uimanifest.NewFSLoader(repoRoot), repoRoot)
}
