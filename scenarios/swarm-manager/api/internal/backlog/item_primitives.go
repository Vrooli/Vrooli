package backlog

import (
	"fmt"
	"os"
	"strings"
)

// ItemAttacher is the minimal initiative-side hook the single-item
// CreateItem helper needs — it records the new item as a member of an
// initiative without touching the item file (which CreateItem has already
// written). Satisfied by the batch-create handler's InitiativeAssigner and
// by the proposals package's narrower InitiativeAssigner, so both surfaces
// can share a single creation primitive.
type ItemAttacher interface {
	RememberItem(initiativeName, ref string) error
}

// ItemWriter is the narrow store surface CreateItem needs: enough to
// locate the item directory on disk and persist the item file. Satisfied
// by backlog.Store, by FileStore directly, and by the proposals
// package's narrower BacklogStore — so every caller can pass whatever
// they already hold.
type ItemWriter interface {
	ItemDir(kind BacklogKind, name string) string
	SaveItem(item BacklogItem) error
}

// CreateItem writes a new backlog item to disk and, when initiativeName is
// non-empty, attaches it to the initiative through attacher. Creation is
// atomic: if the initiative attach fails, the item dir is removed so no
// orphan item file lingers.
//
// Callers own validation (duplicate check, depends_on fan-out, kind
// parsing, priority bounds). This helper is deliberately dumb — it
// enforces no invariants beyond "make the files consistent on disk."
// The shared creation primitive keeps the add_item proposal op and the
// batch-create handler from drifting on lifecycle fields.
func CreateItem(store ItemWriter, attacher ItemAttacher, item BacklogItem, initiativeName string) error {
	itemDir := store.ItemDir(item.Kind, item.Name)
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		return fmt.Errorf("create item dir: %w", err)
	}
	if err := store.SaveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		return fmt.Errorf("save item: %w", err)
	}

	initiativeName = strings.TrimSpace(initiativeName)
	if initiativeName == "" || attacher == nil {
		return nil
	}
	ref := string(item.Kind) + "/" + item.Name
	if err := attacher.RememberItem(initiativeName, ref); err != nil {
		_ = os.RemoveAll(itemDir)
		return fmt.Errorf("attach %s to initiative %s: %w", ref, initiativeName, err)
	}
	return nil
}

// ItemPatch is a struct-based patch describing updatable BacklogItem
// fields. A nil pointer leaves the field unchanged; a non-nil pointer
// applies the value (including explicit empty values that clear the
// field). This shape is the single source of truth for item mutation used
// by both the HTTP PATCH handler and the proposals.OpUpdateItem path, so
// adding a new updatable field requires touching exactly one place.
type ItemPatch struct {
	Title           *string
	Description     *string
	Status          *string
	Priority        *int
	Tags            *[]string
	DependsOn       *[]string
	Initiative      *string
	Effort          *string
	AcceptanceAllow *[]string
	AcceptanceDeny  *[]string
	SpawnedFrom     *string
	Note            *string
}

// ApplyItemPatch mutates item in-place according to patch. Callers remain
// responsible for dependency validation (Store.ValidateDependencies),
// effort normalization, and status-transition gating — those live above
// this helper because different callers validate differently (PATCH
// handler rejects at request time; proposals rejects at Validate time).
func ApplyItemPatch(item *BacklogItem, patch ItemPatch) {
	if patch.Title != nil {
		item.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		item.Description = *patch.Description
	}
	if patch.Status != nil {
		item.Status = BacklogStatus(*patch.Status)
	}
	if patch.Priority != nil {
		item.Priority = *patch.Priority
	}
	if patch.Tags != nil {
		item.Tags = cloneStringsOrEmpty(*patch.Tags)
	}
	if patch.DependsOn != nil {
		item.DependsOn = cloneStrings(*patch.DependsOn)
	}
	if patch.Initiative != nil {
		item.Initiative = strings.TrimSpace(*patch.Initiative)
	}
	if patch.Effort != nil {
		item.Effort = strings.ToUpper(strings.TrimSpace(*patch.Effort))
	}
	if patch.AcceptanceAllow != nil {
		item.AcceptanceAllow = cloneStrings(*patch.AcceptanceAllow)
	}
	if patch.AcceptanceDeny != nil {
		item.AcceptanceDeny = cloneStrings(*patch.AcceptanceDeny)
	}
	if patch.SpawnedFrom != nil {
		item.SpawnedFrom = strings.TrimSpace(*patch.SpawnedFrom)
	}
	if patch.Note != nil {
		item.Note = strings.TrimSpace(*patch.Note)
	}
}
