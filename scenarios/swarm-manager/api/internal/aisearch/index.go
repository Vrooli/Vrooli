package aisearch

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/pathutil"
)

// BacklogReader is the minimum surface aisearch needs to enumerate and load
// backlog items. Implemented by backlog.FileStore via an adapter in the API
// wiring layer.
type BacklogReader interface {
	LoadAll() ([]backlog.BacklogItem, error)
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
}

// GoalReader is the minimum surface aisearch needs to enumerate and load
// goals.
type GoalReader interface {
	List() ([]goals.Goal, error)
	Get(name string) (*goals.Goal, error)
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

// goalPointID returns the deterministic UUIDv5 Qdrant point ID for a goal.
func goalPointID(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		n = "unknown"
	}
	return uuidV5(qdrantNamespace, "swarm-manager:goal/"+n)
}

func capturePointID(id string) string {
	n := strings.TrimSpace(id)
	if n == "" {
		n = "unknown"
	}
	return uuidV5(qdrantNamespace, "swarm-manager:capture/"+n)
}

func composeCaptureText(capture CaptureDocument) string {
	parts := []string{}
	if text := strings.TrimSpace(capture.Text); text != "" {
		parts = append(parts, text)
	}
	if note := strings.TrimSpace(capture.Note); note != "" {
		parts = append(parts, "Note: "+note)
	}
	return strings.Join(parts, "\n\n")
}

func buildCapturePayload(capture CaptureDocument, payloadHash string) map[string]interface{} {
	out := map[string]interface{}{
		"id":         capture.ID,
		"capture_id": capture.ID,
		"text":       capture.Text,
		"note":       capture.Note,
		"status":     "capture",
	}
	if payloadHash != "" {
		out["payload_hash"] = payloadHash
	}
	return out
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

// composePayloadHash returns a short, deterministic content fingerprint of the
// embedding input text plus the searchable payload. The reconciler stores this
// alongside each Qdrant point and compares it against a freshly computed hash
// to skip re-embedding items whose content has not changed.
//
// Format: "sha256:" + hex(sha256(text || "|" || canonicalJSON(payload))[:8]).
// The 8-byte (64-bit) truncation is the "skip-if-unchanged" decision boundary:
// at current scale (~326 items) the birthday-collision probability is ~3×10⁻¹⁵,
// and the cost of a collision is one stale embedding for one item until the
// next genuine edit — non-corrupting. Widen sum[:16] if scale exceeds 10⁵ items.
//
// IMPORTANT: the input payload must NOT contain a "payload_hash" key, otherwise
// the hash includes itself and becomes unreproducible. json.Marshal sorts map
// keys, so the canonical form is stable across goroutines and processes.
func composePayloadHash(text string, payload map[string]interface{}) string {
	canonical, _ := json.Marshal(payload)
	h := sha256.New()
	_, _ = h.Write([]byte(text))
	_, _ = h.Write([]byte{'|'})
	_, _ = h.Write(canonical)
	sum := h.Sum(nil)
	return "sha256:" + hex.EncodeToString(sum[:8])
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
	if init := strings.TrimSpace(item.Milestone); init != "" {
		parts = append(parts, "Milestone: "+init)
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

// composeGoalText builds the embedding input text for a goal.
func composeGoalText(goal goals.Goal) string {
	var parts []string

	if title := strings.TrimSpace(goal.Title); title != "" {
		parts = append(parts, title)
	}
	if desc := strings.TrimSpace(goal.Description); desc != "" {
		parts = append(parts, "Description: "+desc)
	}
	if status := strings.TrimSpace(goal.Status); status != "" {
		parts = append(parts, "Status: "+status)
	}
	if len(goal.Targets) > 0 {
		parts = append(parts, "Targets: "+strings.Join(goal.Targets, ", "))
	}

	return strings.Join(parts, "\n\n")
}

// buildBacklogPayload returns the Qdrant payload map for a backlog item. Keys
// here must stay in sync with BacklogPayload in models.go.
//
// payloadHash, if non-empty, is added under "payload_hash" so the reconciler
// can later skip re-embedding items whose content is unchanged. Callers
// computing the hash itself must pass "" (the hash cannot include itself);
// callers about to upsert pass the freshly computed hash.
func buildBacklogPayload(item backlog.BacklogItem, payloadHash string) map[string]interface{} {
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	scenarios := pathutil.ScenariosFromGlobs(item.AcceptanceAllow)
	if scenarios == nil {
		scenarios = []string{}
	}
	out := map[string]interface{}{
		"kind":             string(item.Kind),
		"name":             item.Name,
		"title":            item.Title,
		"status":           string(item.Status),
		"priority":         item.Priority,
		"tags":             tags,
		"milestone":        item.Milestone,
		"effort":           item.Effort,
		"archived":         item.ArchivedAt != nil,
		"target_scenarios": scenarios,
	}
	if payloadHash != "" {
		out["payload_hash"] = payloadHash
	}
	return out
}

// buildGoalPayload returns the Qdrant payload map for a goal.
// payloadHash, if non-empty, is added under "payload_hash" — same convention
// as buildBacklogPayload.
func buildGoalPayload(goal goals.Goal, payloadHash string) map[string]interface{} {
	out := map[string]interface{}{
		"name":     goal.Name,
		"title":    goal.Title,
		"status":   goal.Status,
		"priority": goal.Priority,
		"archived": goal.ArchivedAt != nil,
	}
	if payloadHash != "" {
		out["payload_hash"] = payloadHash
	}
	return out
}

// IndexBacklogItem embeds and upserts a backlog item's vector. Safe to call
// from write-through hooks; callers should invoke in a goroutine so CRUD
// latency is not affected by embedding/vector-store latency.
func (s *Service) IndexBacklogItem(ctx context.Context, item backlog.BacklogItem) error {
	if s.embedder == nil || s.backlogStore == nil {
		return fmt.Errorf("aisearch not configured for backlog indexing")
	}

	text := composeBacklogText(item)
	// Compute the hash from the no-hash payload first; assignment evaluates
	// RHS before the map write, so composePayloadHash sees a clean payload.
	payload := buildBacklogPayload(item, "")
	payload["payload_hash"] = composePayloadHash(text, payload)

	vector, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed backlog %s/%s: %w", item.Kind, item.Name, err)
	}

	id := backlogPointID(item.Kind, item.Name)
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

// IndexGoal embeds and upserts a goal's vector.
func (s *Service) IndexGoal(ctx context.Context, goal goals.Goal) error {
	if s.embedder == nil || s.goalStore == nil {
		return fmt.Errorf("aisearch not configured for goal indexing")
	}
	text := composeGoalText(goal)
	payload := buildGoalPayload(goal, "")
	payload["payload_hash"] = composePayloadHash(text, payload)

	vector, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed goal %s: %w", goal.Name, err)
	}

	id := goalPointID(goal.Name)
	if err := s.goalStore.Upsert(ctx, id, vector, payload); err != nil {
		return fmt.Errorf("upsert goal %s: %w", goal.Name, err)
	}

	slog.Debug("[aisearch] indexed goal", "name", goal.Name, "id", id)
	return nil
}

// DeleteGoal removes the vector for a goal.
func (s *Service) DeleteGoal(ctx context.Context, name string) error {
	if s.goalStore == nil {
		return fmt.Errorf("aisearch not configured for goal indexing")
	}
	id := goalPointID(name)
	if err := s.goalStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete goal %s: %w", name, err)
	}
	slog.Debug("[aisearch] deleted goal from index", "name", name, "id", id)
	return nil
}

// IndexCapture embeds the raw capture text for semantic deduplication and
// grounding. It is safe to call asynchronously from the capture write path.
func (s *Service) IndexCapture(ctx context.Context, id, text, note string) error {
	if s.embedder == nil || s.captureStore == nil {
		return fmt.Errorf("aisearch not configured for capture indexing")
	}
	doc := CaptureDocument{ID: id, Text: text, Note: note}
	content := composeCaptureText(doc)
	payload := buildCapturePayload(doc, "")
	payload["payload_hash"] = composePayloadHash(content, payload)
	vector, err := s.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("embed capture %s: %w", id, err)
	}
	if err := s.captureStore.Upsert(ctx, capturePointID(id), vector, payload); err != nil {
		return fmt.Errorf("upsert capture %s: %w", id, err)
	}
	return nil
}

func (s *Service) DeleteCapture(ctx context.Context, id string) error {
	if s.captureStore == nil {
		return fmt.Errorf("aisearch not configured for capture indexing")
	}
	return s.captureStore.Delete(ctx, capturePointID(id))
}
