// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// importChange is the internal representation before converting to proto.
type importChange struct {
	item       string // kind/name
	action     string // "update" or "create"
	details    []string
	createData *createItemData
	updateData *updateItemData
}

// itemMarkerRe matches <!-- item:KIND/NAME --> or <!-- item:NEW -->.
var itemMarkerRe = regexp.MustCompile(`^<!--\s*item:(\S+)\s*-->`)

// clarifyMarkerRe matches <!-- clarify:KIND/NAME -->.
var clarifyMarkerRe = regexp.MustCompile(`^<!--\s*clarify:(\S+)\s*-->`)

// suggestMarkerRe matches <!-- suggest:KIND/NAME -->.
var suggestMarkerRe = regexp.MustCompile(`^<!--\s*suggest:(\S+)\s*-->`)

// notesMarkerRe matches <!-- notes:KIND/NAME -->.
var notesMarkerRe = regexp.MustCompile(`^<!--\s*notes:(\S+)\s*-->`)

// newItemHeadingRe matches ## idea/my-new-idea -- Title Here (supports em-dash and double-dash).
var newItemHeadingRe = regexp.MustCompile(`^##\s+(\w+)/(\S+)\s*(?:—|--)\s*(.+)$`)

// checkboxRe matches [ ] or [x] checkbox lines.
var checkboxRe = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(.+)$`)

// metadataRowRe matches table rows like | **Status** | ready | or | Status | ready |.
var metadataRowRe = regexp.MustCompile(`^\|\s*\**(\w[\w\s]*?)\**\s*\|\s*(.*?)\s*\|`)

// Import handles the POST /api/v1/backlog/import endpoint.
// It parses an edited markdown export and applies (or previews) changes.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apierr.MapError(w, "[backlog] import", apierr.BadRequest("failed to parse multipart form"))
		return
	}

	applyChanges := r.FormValue("apply") == "true"

	file, _, err := r.FormFile("file")
	if err != nil {
		apierr.MapError(w, "[backlog] import", apierr.BadRequest("file field is required"))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		apierr.MapError(w, "[backlog] import", apierr.Internal("failed to read uploaded file"))
		return
	}

	changes, parseErrors := h.parseImportMarkdown(string(content))

	if applyChanges {
		for i := range changes {
			if err := h.applyChange(&changes[i]); err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", changes[i].item, err))
			}
		}
	}

	updatedCount := 0
	createdCount := 0
	for _, c := range changes {
		switch c.action {
		case "update":
			updatedCount++
		case "create":
			createdCount++
		}
	}

	protoChanges := make([]*apipb.ImportChange, 0, len(changes))
	for _, c := range changes {
		protoChanges = append(protoChanges, &apipb.ImportChange{
			Item:    c.item,
			Action:  c.action,
			Details: c.details,
		})
	}

	resp := &apipb.ImportBacklogResponse{
		DryRun:  !applyChanges,
		Changes: protoChanges,
		Errors:  parseErrors,
		Summary: fmt.Sprintf("%d items updated, %d items created, %d errors", updatedCount, createdCount, len(parseErrors)),
	}

	if applyChanges && len(changes) > 0 {
		h.invalidateAllGraphLenses()
	}

	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] import", apierr.Internal("failed to encode response"))
	}
}

// applyChange executes a single import change.
func (h *Handler) applyChange(change *importChange) error {
	switch change.action {
	case "create":
		return h.applyCreate(change)
	case "update":
		return h.applyUpdate(change)
	default:
		return fmt.Errorf("unknown action: %s", change.action)
	}
}

// applyCreate creates a new backlog item from import data.
func (h *Handler) applyCreate(change *importChange) error {
	cd := change.createData
	if cd == nil {
		return fmt.Errorf("no create data")
	}

	itemDir := h.store.ItemDir(cd.kind, cd.name)
	if _, err := os.Stat(itemDir); err == nil {
		return fmt.Errorf("item already exists: %s/%s", cd.kind, cd.name)
	}

	if err := os.MkdirAll(itemDir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tags := cd.tags
	if tags == nil {
		tags = []string{}
	}

	item := BacklogItem{
		Name:        cd.name,
		Title:       cd.title,
		Description: cd.description,
		Status:      StatusBacklog,
		Priority:    cd.priority,
		Tags:        tags,
		Created:     now,
		Updated:     now,
		Kind:        cd.kind,
	}

	if err := h.store.SaveItem(item); err != nil {
		if rmErr := os.RemoveAll(itemDir); rmErr != nil {
			slog.Debug("backlog: rollback imported item dir failed", "err", rmErr, "dir", itemDir)
		}
		return fmt.Errorf("failed to save item: %w", err)
	}

	slog.Info("import created item", "kind", cd.kind, "name", cd.name)
	return nil
}

// applyUpdate applies changes to an existing backlog item.
func (h *Handler) applyUpdate(change *importChange) error {
	ud := change.updateData
	if ud == nil {
		return fmt.Errorf("no update data")
	}

	item := ud.item
	modified := false

	if ud.description != nil {
		item.Description = *ud.description
		modified = true
	}
	if ud.status != nil {
		item.Status = BacklogStatus(*ud.status)
		modified = true
	}
	if ud.priority != nil {
		item.Priority = *ud.priority
		modified = true
	}
	if ud.hasTags {
		item.Tags = ud.tags
		if item.Tags == nil {
			item.Tags = []string{}
		}
		modified = true
	}

	if modified {
		item.Updated = time.Now().UTC().Format(time.RFC3339)
		if err := h.store.SaveItem(item); err != nil {
			return fmt.Errorf("failed to save item: %w", err)
		}
	}

	// Apply clarify changes.
	if len(ud.clarifyAnswers) > 0 || len(ud.clarifyNotes) > 0 {
		questionsPath := filepath.Join(h.store.ItemDir(ud.kind, ud.name), "clarify", "questions.json")
		if err := h.applyClarifyChanges(questionsPath, ud.clarifyAnswers, ud.clarifyNotes); err != nil {
			slog.Warn("import failed to apply clarify changes", "kind", ud.kind, "name", ud.name, "err", err)
		}
	}

	// Apply suggest changes.
	if len(ud.suggestAccepted) > 0 || len(ud.suggestRejection) > 0 {
		suggestionsPath := filepath.Join(h.store.ItemDir(ud.kind, ud.name), "suggest", "suggestions.json")
		if err := h.applySuggestChanges(suggestionsPath, ud.suggestAccepted, ud.suggestRejection); err != nil {
			slog.Warn("import failed to apply suggest changes", "kind", ud.kind, "name", ud.name, "err", err)
		}
	}

	// Apply notes changes.
	if ud.notes != "" {
		notesPath := filepath.Join(h.store.ItemDir(ud.kind, ud.name), "notes.md")
		if err := os.WriteFile(notesPath, []byte(ud.notes+"\n"), 0o600); err != nil {
			slog.Warn("import failed to write notes", "kind", ud.kind, "name", ud.name, "err", err)
		}
	}

	slog.Info("import updated item", "kind", ud.kind, "name", ud.name, "changes", len(change.details))
	return nil
}

// applyClarifyChanges updates questions.json with new answers and notes.
func (h *Handler) applyClarifyChanges(questionsPath string, answers map[int]string, notes map[int]string) error {
	questions, err := loadQuestions(questionsPath)
	if err != nil {
		return err
	}

	modified := false
	for idx, answer := range answers {
		if idx < len(questions) && answer != "" {
			if questions[idx].Answer != answer {
				questions[idx].Answer = answer
				modified = true
			}
		}
	}
	for idx, note := range notes {
		if idx < len(questions) && note != "" {
			if questions[idx].Notes != note {
				questions[idx].Notes = note
				modified = true
			}
		}
	}

	if !modified {
		return nil
	}

	data, err := json.MarshalIndent(questions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal questions: %w", err)
	}
	return os.WriteFile(questionsPath, data, 0o600)
}

// applySuggestChanges updates suggestions.json with acceptance and rejection data.
func (h *Handler) applySuggestChanges(suggestionsPath string, accepted map[int]bool, rejections map[int]string) error {
	suggestions, err := loadSuggestions(suggestionsPath)
	if err != nil {
		return err
	}

	modified := false
	for idx, isAccepted := range accepted {
		if idx < len(suggestions) {
			if suggestions[idx].Accepted != isAccepted {
				suggestions[idx].Accepted = isAccepted
				modified = true
			}
		}
	}
	for idx, reason := range rejections {
		if idx < len(suggestions) && reason != "" {
			if suggestions[idx].RejectionReason != reason {
				suggestions[idx].RejectionReason = reason
				suggestions[idx].Accepted = false
				modified = true
			}
		}
	}

	if !modified {
		return nil
	}

	data, err := json.MarshalIndent(suggestions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal suggestions: %w", err)
	}
	return os.WriteFile(suggestionsPath, data, 0o600)
}
