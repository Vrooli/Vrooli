package continuity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Evidence is an observation from a Web Console projection or native agent
// source. Reconciliation consumes observations; it never edits the native
// transcript represented by RolloutRef or AgentHomeRef.
type Evidence struct {
	SessionID        string
	Backend          string
	AgentType        string
	AgentSessionID   string
	AgentHomeRef     string
	RolloutRef       string
	CWD              string
	OriginalTitle    string
	CurrentTitle     string
	TopicSummary     string
	CreatedAt        time.Time
	LastActivityAt   time.Time
	HasConversation  bool
	HasNativeHistory bool
}

type Alias struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type CatalogRecord struct {
	SessionID         string
	LifecycleState    State
	AgentType         string
	Backend           string
	AgentSessionID    string
	AgentHomeRef      string
	RolloutRef        string
	OriginalTitle     string
	CurrentTitle      string
	TopicSummary      string
	CWD               string
	CreatedAt         time.Time
	LastActivityAt    time.Time
	SourceFingerprint string
	Aliases           []Alias
}

type ReconcileAction string

const (
	ActionCreate     ReconcileAction = "create"
	ActionUpdate     ReconcileAction = "update"
	ActionQuarantine ReconcileAction = "quarantine"
)

type ReconcileItem struct {
	Action ReconcileAction
	Record CatalogRecord
	Reason string
}

// PublicRecord removes host filesystem locators from transport-facing
// catalog views. Publication uses the private model inside this package; UI
// and API callers receive stable identities without a filesystem reader.
func PublicRecord(record CatalogRecord) CatalogRecord {
	copy := record
	copy.AgentHomeRef = ""
	copy.RolloutRef = ""
	copy.Aliases = make([]Alias, 0, len(record.Aliases))
	for _, alias := range record.Aliases {
		if alias.Kind == "agent_home" || alias.Kind == "rollout" {
			continue
		}
		copy.Aliases = append(copy.Aliases, alias)
	}
	return copy
}

func PublicReconcileItem(item ReconcileItem) ReconcileItem {
	item.Record = PublicRecord(item.Record)
	return item
}

type ReconciliationManifest struct {
	Hash       string
	Generation string
	Items      []ReconcileItem
	Previous   map[string]CatalogRecord
}

// ManifestHash identifies the exact dry-run operation set. It is stable for
// the same generation and ordered evidence, and contains no transcript text.
func ManifestHash(generation string, items []ReconcileItem) string {
	type manifestItem struct {
		Action, SessionID, Reason, LifecycleState, SourceFingerprint string
	}
	values := make([]manifestItem, 0, len(items))
	for _, item := range items {
		values = append(values, manifestItem{string(item.Action), item.Record.SessionID, item.Reason, string(item.Record.LifecycleState), item.Record.SourceFingerprint})
	}
	canonical, _ := json.Marshal(struct {
		Generation string
		Items      []manifestItem
	}{generation, values})
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// BuildCatalogRecord turns one source observation into a stable catalog
// record. Titles are deterministic immediately; model-generated enrichment is
// intentionally outside this function and cannot make a record disappear.
func BuildCatalogRecord(e Evidence) (CatalogRecord, error) {
	if strings.TrimSpace(e.SessionID) == "" {
		return CatalogRecord{}, fmt.Errorf("session id is required")
	}
	state := StateArchived
	if e.HasConversation || e.HasNativeHistory {
		state = StateRecoverable
	}
	title := strings.TrimSpace(e.CurrentTitle)
	if title == "" {
		title = strings.TrimSpace(e.OriginalTitle)
	}
	if title == "" {
		title = "Web Console conversation " + e.SessionID[:min(8, len(e.SessionID))]
	}
	record := CatalogRecord{
		SessionID: e.SessionID, LifecycleState: state, AgentType: strings.TrimSpace(e.AgentType), Backend: strings.TrimSpace(e.Backend),
		AgentSessionID: strings.TrimSpace(e.AgentSessionID), AgentHomeRef: strings.TrimSpace(e.AgentHomeRef),
		RolloutRef: strings.TrimSpace(e.RolloutRef), OriginalTitle: strings.TrimSpace(e.OriginalTitle),
		CurrentTitle: title, TopicSummary: strings.TrimSpace(e.TopicSummary), CWD: strings.TrimSpace(e.CWD),
		CreatedAt: e.CreatedAt.UTC(), LastActivityAt: e.LastActivityAt.UTC(),
	}
	if record.AgentSessionID != "" {
		record.Aliases = append(record.Aliases, Alias{"agent_session", record.AgentSessionID})
	}
	if record.AgentHomeRef != "" {
		record.Aliases = append(record.Aliases, Alias{"agent_home", record.AgentHomeRef})
	}
	if record.RolloutRef != "" {
		record.Aliases = append(record.Aliases, Alias{"rollout", record.RolloutRef})
	}
	sort.Slice(record.Aliases, func(i, j int) bool {
		return record.Aliases[i].Kind+record.Aliases[i].Value < record.Aliases[j].Kind+record.Aliases[j].Value
	})
	canonical, err := json.Marshal(record)
	if err != nil {
		return CatalogRecord{}, fmt.Errorf("fingerprint catalog record: %w", err)
	}
	digest := sha256.Sum256(canonical)
	record.SourceFingerprint = "sha256:" + hex.EncodeToString(digest[:])
	return record, nil
}

// PlanReconciliation returns a deterministic, dry-run plan. Existing records
// are keyed by session id. Alias collisions are quarantined rather than
// silently choosing one source identity.
func PlanReconciliation(observations []Evidence, existing map[string]CatalogRecord) ([]ReconcileItem, error) {
	ordered := append([]Evidence(nil), observations...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].SessionID < ordered[j].SessionID })
	aliases := map[string]string{}
	existingIDs := make([]string, 0, len(existing))
	for sessionID := range existing {
		existingIDs = append(existingIDs, sessionID)
	}
	sort.Strings(existingIDs)
	for _, sessionID := range existingIDs {
		record := existing[sessionID]
		for _, alias := range record.Aliases {
			key := alias.Kind + "\x00" + alias.Value
			// Keep the lexicographically first existing owner when legacy
			// records already collide. This makes quarantine reasons and the
			// resulting manifest stable across process runs.
			if _, occupied := aliases[key]; !occupied {
				aliases[key] = record.SessionID
			}
		}
	}
	items := make([]ReconcileItem, 0, len(ordered))
	for _, observation := range ordered {
		record, err := BuildCatalogRecord(observation)
		if err != nil {
			return nil, err
		}
		collision := ""
		for _, alias := range record.Aliases {
			if owner := aliases[alias.Kind+"\x00"+alias.Value]; owner != "" && owner != record.SessionID {
				collision = owner
				break
			}
		}
		if collision != "" {
			items = append(items, ReconcileItem{Action: ActionQuarantine, Record: record, Reason: "alias collision with " + collision})
			continue
		}
		for _, alias := range record.Aliases {
			aliases[alias.Kind+"\x00"+alias.Value] = record.SessionID
		}
		action := ActionCreate
		if prior, ok := existing[record.SessionID]; ok && prior.SourceFingerprint == record.SourceFingerprint {
			continue
		}
		if _, ok := existing[record.SessionID]; ok {
			action = ActionUpdate
		}
		items = append(items, ReconcileItem{Action: action, Record: record, Reason: "source evidence observed"})
	}
	return items, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
