package adoptions

import (
	"context"
	"errors"
	"os"
	"strings"

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

// IndexedSlotReader reads the component's indexed slot from the components
// service. The source component.json remains authoritative, but the index is
// the runtime adoption contract.
type IndexedSlotReader struct {
	Components components.Service
}

// Slot resolves componentID through the indexed component row. Empty string
// means the component did not declare a slot and the resolver will use the
// target manifest default.
func (r *IndexedSlotReader) Slot(ctx context.Context, componentID string) (string, error) {
	if r == nil || r.Components == nil {
		return "", errors.New("slot reader: components service not configured")
	}
	c, err := r.Components.Get(ctx, componentID)
	if err != nil {
		return "", err
	}
	return c.Slot, nil
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
	resp := &adoptionsv1.ResolveAdoptionPathResponse{}

	// Legacy single-path resolution needs a concrete scenario. Template-only
	// requests (the code panel's scenario-agnostic preview) skip it; the entry
	// summary is derived from the version placement below instead.
	if req.Msg.Scenario != "" || req.Msg.OverridePath != "" {
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
		resp.Path = out.Path
		resp.Source = toProtoSource(out.Source)
		resp.Slot = out.Slot
		resp.Warnings = out.Warnings
	} else if req.Msg.Template == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("ResolveAdoptionPath: scenario or template is required"))
	}

	// Full file-set placement. Non-fatal: a version lookup failure leaves the
	// response as the single entry path (the summary above) so the caller
	// still gets a usable answer.
	if files := h.resolveVersionFiles(ctx, req.Msg, name, slot); files != nil {
		resp.Files = files.protoFiles
		resp.Template = files.template
		resp.ManifestResolved = files.manifestResolved
		// Template-only mode: derive the entry summary from the placement.
		if resp.Path == "" {
			for _, f := range files.protoFiles {
				if f.IsEntry {
					resp.Path = f.TargetPath
					resp.Source = f.Source
					resp.Slot = f.Slot
					resp.Warnings = append(resp.Warnings, f.Warnings...)
					break
				}
			}
		}
	}
	return connect.NewResponse(resp), nil
}

type versionPlacement struct {
	protoFiles       []*adoptionsv1.ResolvedVersionFile
	template         string
	manifestResolved bool
}

// resolveVersionFiles fetches the version's file set and places each member at
// its slot-derived path. Returns nil when the file set can't be determined
// (the caller then relies on the single entry path). An explicit override or a
// missing Components service short-circuits to nil — overrides only name the
// entry file, so a multi-file tree would be misleading.
func (h *connectHandler) resolveVersionFiles(ctx context.Context, msg *adoptionsv1.ResolveAdoptionPathRequest, name, slot string) *versionPlacement {
	if h.deps.Components == nil || msg.OverridePath != "" {
		return nil
	}
	version := strings.TrimSpace(msg.Version)
	if version == "" {
		if c, err := h.deps.Components.Get(ctx, msg.ComponentId); err == nil {
			version = c.LatestVersion
			if version == "" {
				version = c.Version
			}
		}
	}
	var files []components.ComponentVersionFile
	if version != "" {
		if v, err := h.deps.Components.GetVersion(ctx, msg.ComponentId, version); err == nil {
			files = v.Files
		} else {
			h.deps.Logger.Printf("ResolveAdoptionPath: version %q lookup failed for %q: %v", version, msg.ComponentId, err)
		}
	}

	in := adoptions.VersionResolveInput{
		ComponentName: name,
		ComponentSlot: slot,
		Scenario:      msg.Scenario,
		Template:      msg.Template,
		Feature:       msg.Feature,
	}
	for _, f := range files {
		in.Files = append(in.Files, adoptions.FileInput{Path: f.Path, IsEntry: f.IsEntry, Slot: f.Slot})
	}
	out, err := h.deps.Resolver.ResolveVersion(in)
	if err != nil {
		h.deps.Logger.Printf("ResolveAdoptionPath: version placement failed for %q: %v", msg.ComponentId, err)
		return nil
	}
	placement := &versionPlacement{template: out.Template, manifestResolved: out.ManifestResolved}
	for _, rf := range out.Files {
		placement.protoFiles = append(placement.protoFiles, &adoptionsv1.ResolvedVersionFile{
			LibraryPath: rf.LibraryPath,
			TargetPath:  rf.TargetPath,
			Slot:        rf.Slot,
			Source:      toProtoSource(rf.Source),
			SlotSource:  string(rf.SlotSource),
			IsEntry:     rf.IsEntry,
			Warnings:    rf.Warnings,
		})
	}
	return placement
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
