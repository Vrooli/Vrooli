package aisearch

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
)

// BacklogReader is the minimum surface aisearch needs to enumerate and load
// backlog items. Implemented by backlog.FileStore via an adapter in the API
// wiring layer.
type BacklogReader interface {
	LoadAll() ([]backlog.BacklogItem, error)
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
}

// InitiativeReader is the minimum surface aisearch needs to enumerate and
// load initiatives. Implemented by initiatives.Service via an adapter.
type InitiativeReader interface {
	List() ([]initiatives.Initiative, error)
	Get(name string) (*initiatives.Initiative, error)
}

// qdrantNamespace is a stable namespace UUID used to derive deterministic
// point IDs for swarm-manager entities. Changing this value would orphan all
// existing vectors; do not modify.
var qdrantNamespace = [16]byte{
	0x6b, 0xa7, 0xb8, 0x10,
	0x9d, 0xad, 0x11, 0xd1,
	0x80, 0xb4, 0x00, 0xc0,
	0x4f, 0xd4, 0x30, 0xc8,
}

// backlogPointID returns the deterministic UUIDv5 Qdrant point ID for a
// backlog item.
func backlogPointID(kind backlog.BacklogKind, name string) string {
	k := strings.TrimSpace(string(kind))
	n := strings.TrimSpace(name)
	if k == "" {
		k = "unknown"
	}
	if n == "" {
		n = "unknown"
	}
	return uuidV5(qdrantNamespace, "swarm-manager:"+k+"/"+n)
}

// initiativePointID returns the deterministic UUIDv5 Qdrant point ID for an
// initiative.
func initiativePointID(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		n = "unknown"
	}
	return uuidV5(qdrantNamespace, "swarm-manager:initiative/"+n)
}

func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], sum[:16])

	// Set version (5) and RFC4122 variant bits.
	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(uuid[:])
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]
}

// composeBacklogText builds the embedding input text for a backlog item.
// Fields are concatenated with blank-line separators so a dense-embedding
// model can weight them roughly equally.
func composeBacklogText(item backlog.BacklogItem) string {
	var parts []string

	if title := strings.TrimSpace(item.Title); title != "" {
		parts = append(parts, title)
	}
	if desc := strings.TrimSpace(item.Description); desc != "" {
		parts = append(parts, desc)
	}
	if len(item.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(item.Tags, ", "))
	}
	if kind := strings.TrimSpace(string(item.Kind)); kind != "" {
		parts = append(parts, "Kind: "+kind)
	}
	if status := strings.TrimSpace(string(item.Status)); status != "" {
		parts = append(parts, "Status: "+status)
	}
	if init := strings.TrimSpace(item.Initiative); init != "" {
		parts = append(parts, "Initiative: "+init)
	}
	if effort := strings.TrimSpace(item.Effort); effort != "" {
		parts = append(parts, "Effort: "+effort)
	}
	if len(item.DependsOn) > 0 {
		parts = append(parts, "Dependencies: "+strings.Join(item.DependsOn, ", "))
	}
	if note := strings.TrimSpace(item.Note); note != "" {
		if len(note) > 2000 {
			note = note[:2000] + "..."
		}
		parts = append(parts, "Note: "+note)
	}

	return strings.Join(parts, "\n\n")
}

// composeInitiativeText builds the embedding input text for an initiative.
func composeInitiativeText(init initiatives.Initiative) string {
	var parts []string

	if title := strings.TrimSpace(init.Title); title != "" {
		parts = append(parts, title)
	}
	if desc := strings.TrimSpace(init.Description); desc != "" {
		parts = append(parts, "Description: "+desc)
	}
	if status := strings.TrimSpace(init.Status); status != "" {
		parts = append(parts, "Status: "+status)
	}
	if len(init.DependsOn) > 0 {
		parts = append(parts, "Dependencies: "+strings.Join(init.DependsOn, ", "))
	}
	if len(init.Items) > 0 {
		parts = append(parts, "Items: "+strings.Join(init.Items, ", "))
	}
	if note := strings.TrimSpace(init.Note); note != "" {
		if len(note) > 2000 {
			note = note[:2000] + "..."
		}
		parts = append(parts, "Note: "+note)
	}

	return strings.Join(parts, "\n\n")
}

// buildBacklogPayload returns the Qdrant payload map for a backlog item.
// Keys here must stay in sync with BacklogPayload in models.go.
func buildBacklogPayload(item backlog.BacklogItem) map[string]interface{} {
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	return map[string]interface{}{
		"kind":       string(item.Kind),
		"name":       item.Name,
		"title":      item.Title,
		"status":     string(item.Status),
		"priority":   item.Priority,
		"tags":       tags,
		"initiative": item.Initiative,
		"effort":     item.Effort,
		"archived":   item.ArchivedAt != nil,
	}
}

// buildInitiativePayload returns the Qdrant payload map for an initiative.
func buildInitiativePayload(init initiatives.Initiative) map[string]interface{} {
	return map[string]interface{}{
		"name":     init.Name,
		"title":    init.Title,
		"status":   init.Status,
		"priority": init.Priority,
		"archived": init.ArchivedAt != nil,
	}
}

// IndexBacklogItem embeds and upserts a backlog item's vector. Safe to call
// from write-through hooks; callers should invoke in a goroutine so CRUD
// latency is not affected by embedding/vector-store latency.
func (s *Service) IndexBacklogItem(ctx context.Context, item backlog.BacklogItem) error {
	if s.embedder == nil || s.backlogStore == nil {
		return fmt.Errorf("aisearch not configured for backlog indexing")
	}

	text := composeBacklogText(item)
	vector, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed backlog %s/%s: %w", item.Kind, item.Name, err)
	}

	id := backlogPointID(item.Kind, item.Name)
	payload := buildBacklogPayload(item)
	if err := s.backlogStore.Upsert(ctx, id, vector, payload); err != nil {
		return fmt.Errorf("upsert backlog %s/%s: %w", item.Kind, item.Name, err)
	}

	slog.Debug("[aisearch] indexed backlog item", "kind", item.Kind, "name", item.Name, "id", id)
	return nil
}

// DeleteBacklogItem removes the vector for a backlog item identified by kind
// and name.
func (s *Service) DeleteBacklogItem(ctx context.Context, kind backlog.BacklogKind, name string) error {
	if s.backlogStore == nil {
		return fmt.Errorf("aisearch not configured for backlog indexing")
	}
	id := backlogPointID(kind, name)
	if err := s.backlogStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete backlog %s/%s: %w", kind, name, err)
	}
	slog.Debug("[aisearch] deleted backlog item from index", "kind", kind, "name", name, "id", id)
	return nil
}

// IndexInitiative embeds and upserts an initiative's vector.
func (s *Service) IndexInitiative(ctx context.Context, init initiatives.Initiative) error {
	if s.embedder == nil || s.initiativeStore == nil {
		return fmt.Errorf("aisearch not configured for initiative indexing")
	}

	text := composeInitiativeText(init)
	vector, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed initiative %s: %w", init.Name, err)
	}

	id := initiativePointID(init.Name)
	payload := buildInitiativePayload(init)
	if err := s.initiativeStore.Upsert(ctx, id, vector, payload); err != nil {
		return fmt.Errorf("upsert initiative %s: %w", init.Name, err)
	}

	slog.Debug("[aisearch] indexed initiative", "name", init.Name, "id", id)
	return nil
}

// DeleteInitiative removes the vector for an initiative.
func (s *Service) DeleteInitiative(ctx context.Context, name string) error {
	if s.initiativeStore == nil {
		return fmt.Errorf("aisearch not configured for initiative indexing")
	}
	id := initiativePointID(name)
	if err := s.initiativeStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete initiative %s: %w", name, err)
	}
	slog.Debug("[aisearch] deleted initiative from index", "name", name, "id", id)
	return nil
}

// reindexAllWithProgress is the underlying worker for ReindexAll. It walks
// both collections sequentially and invokes the progress callback after each
// entity. Any per-entity error is logged and counted; the reindex continues.
func (s *Service) reindexAllWithProgress(
	ctx context.Context,
	progress func(indexed, skipped, errors int),
	setTotal func(total int),
) (*ReindexResponse, error) {
	var indexed, skipped, errs int

	if s.backlogStore != nil {
		if err := s.backlogStore.EnsureCollection(ctx); err != nil {
			return nil, fmt.Errorf("ensure backlog collection: %w", err)
		}
	}
	if s.initiativeStore != nil {
		if err := s.initiativeStore.EnsureCollection(ctx); err != nil {
			return nil, fmt.Errorf("ensure initiative collection: %w", err)
		}
	}

	var backlogItems []backlog.BacklogItem
	var initList []initiatives.Initiative
	if s.backlogReader != nil {
		items, err := s.backlogReader.LoadAll()
		if err != nil {
			return nil, fmt.Errorf("load backlog items: %w", err)
		}
		backlogItems = items
	}
	if s.initiativeReader != nil {
		inits, err := s.initiativeReader.List()
		if err != nil {
			return nil, fmt.Errorf("list initiatives: %w", err)
		}
		initList = inits
	}

	if setTotal != nil {
		setTotal(len(backlogItems) + len(initList))
	}

	for _, item := range backlogItems {
		if err := ctx.Err(); err != nil {
			return &ReindexResponse{
				Indexed: indexed,
				Skipped: skipped,
				Errors:  errs,
				Message: fmt.Sprintf("Reindex canceled after %d items", indexed+skipped+errs),
			}, err
		}
		if err := s.IndexBacklogItem(ctx, item); err != nil {
			slog.Warn("[aisearch] reindex backlog failed", "kind", item.Kind, "name", item.Name, "err", err)
			errs++
		} else {
			indexed++
		}
		if progress != nil {
			progress(indexed, skipped, errs)
		}
	}

	for _, init := range initList {
		if err := ctx.Err(); err != nil {
			return &ReindexResponse{
				Indexed: indexed,
				Skipped: skipped,
				Errors:  errs,
				Message: fmt.Sprintf("Reindex canceled after %d items", indexed+skipped+errs),
			}, err
		}
		if err := s.IndexInitiative(ctx, init); err != nil {
			slog.Warn("[aisearch] reindex initiative failed", "name", init.Name, "err", err)
			errs++
		} else {
			indexed++
		}
		if progress != nil {
			progress(indexed, skipped, errs)
		}
	}

	return &ReindexResponse{
		Indexed: indexed,
		Skipped: skipped,
		Errors:  errs,
		Message: fmt.Sprintf("Reindex complete: %d indexed, %d skipped, %d errors", indexed, skipped, errs),
	}, nil
}

// ReindexAll runs a full reindex synchronously. Callers that want background
// execution should use StartReindex instead.
func (s *Service) ReindexAll(ctx context.Context) (*ReindexResponse, error) {
	return s.reindexAllWithProgress(ctx, nil, nil)
}
