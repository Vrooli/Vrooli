// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/SEAMS.md

package main

import (
	"context"
	"fmt"

	shortcutsH "web-console/handlers/shortcuts"
)

// shortcutsAdapter bridges the in-process ShortcutStore to the
// transport-neutral Service interface that the Connect handler depends
// on. [REQ:P1-002a] Shortcut Profile Storage
type shortcutsAdapter struct {
	s *Server
}

func newShortcutsAdapter(s *Server) *shortcutsAdapter {
	return &shortcutsAdapter{s: s}
}

func (a *shortcutsAdapter) Effective(ctx context.Context) []shortcutsH.Shortcut {
	return entriesToTransport(a.s.shortcuts.Effective(ctx))
}

func (a *shortcutsAdapter) List(ctx context.Context) []shortcutsH.Profile {
	profiles := a.s.shortcuts.List(ctx)
	out := make([]shortcutsH.Profile, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, profileToTransport(p))
	}
	return out
}

func (a *shortcutsAdapter) Upsert(ctx context.Context, req shortcutsH.UpsertRequest) (shortcutsH.Profile, error) {
	if req.ID == "" {
		return shortcutsH.Profile{}, fmt.Errorf("%w: profile ID is required", shortcutsH.ErrInvalidArgument)
	}
	if !validScopes[req.Scope] {
		return shortcutsH.Profile{}, fmt.Errorf("%w: scope must be 'service', 'workspace', or 'parent'", shortcutsH.ErrInvalidArgument)
	}
	if req.Name == "" {
		return shortcutsH.Profile{}, fmt.Errorf("%w: profile name is required", shortcutsH.ErrInvalidArgument)
	}
	for i, sc := range req.Shortcuts {
		if sc.Label == "" {
			return shortcutsH.Profile{}, fmt.Errorf("%w: shortcut label is required (entry %d)", shortcutsH.ErrInvalidArgument, i)
		}
		if sc.Command == "" {
			return shortcutsH.Profile{}, fmt.Errorf("%w: shortcut command is required (entry %d)", shortcutsH.ErrInvalidArgument, i)
		}
	}
	entries := entriesFromTransport(req.Shortcuts)
	p := a.s.shortcuts.Upsert(ctx, req.ID, req.Scope, req.Name, entries)
	return profileToTransport(p), nil
}

func (a *shortcutsAdapter) Delete(ctx context.Context, id string) {
	a.s.shortcuts.Delete(ctx, id) // idempotent
}

func entriesToTransport(in []ShortcutEntry) []shortcutsH.Shortcut {
	out := make([]shortcutsH.Shortcut, 0, len(in))
	for _, e := range in {
		out = append(out, shortcutsH.Shortcut{
			Label:       e.Label,
			Command:     e.Command,
			Description: e.Description,
		})
	}
	return out
}

func entriesFromTransport(in []shortcutsH.Shortcut) []ShortcutEntry {
	out := make([]ShortcutEntry, 0, len(in))
	for _, e := range in {
		out = append(out, ShortcutEntry{
			Label:       e.Label,
			Command:     e.Command,
			Description: e.Description,
		})
	}
	return out
}

func profileToTransport(p *ShortcutProfile) shortcutsH.Profile {
	return shortcutsH.Profile{
		ID:        p.ID,
		Scope:     p.Scope,
		Name:      p.Name,
		Shortcuts: entriesToTransport(p.Shortcuts),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
