package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"business-health/internal/extraction"

	pkg "github.com/vrooli/ai-go/search"
	intent "intent-go"
)

// idPrefix namespaces business-health's point IDs (UUIDv5 derivation).
const idPrefix = "business-health:"

// intentKind is the engine-side logical collection for the corpus.
const intentKind = "intent"

// IntentSource loads the fleet-wide intent corpus. The interface exists so
// tests can substitute fixture record sets.
type IntentSource interface {
	LoadRecords(ctx context.Context) ([]IntentRecord, error)
}

// FleetIntentSource extracts intent records from every scenario under
// repoRoot/scenarios via intent-go (the single-parser ratchet — this file
// composes extraction, it never parses artifacts itself).
type FleetIntentSource struct {
	RepoRoot  string
	extractor extraction.FileExtractor
}

// NewFleetIntentSource builds the production source.
func NewFleetIntentSource(repoRoot string) *FleetIntentSource {
	return &FleetIntentSource{RepoRoot: repoRoot, extractor: extraction.NewFileExtractor()}
}

// LoadRecords walks the fleet and emits one record per PRD overview, per
// operational target, and per requirement. Scenarios that fail extraction
// are skipped (a broken contract shows up in validation, not here).
func (f *FleetIntentSource) LoadRecords(ctx context.Context) ([]IntentRecord, error) {
	entries, err := os.ReadDir(filepath.Join(f.RepoRoot, "scenarios"))
	if err != nil {
		return nil, err
	}
	var out []IntentRecord
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		dir := filepath.Join(f.RepoRoot, "scenarios", slug)
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "service.json")); err != nil {
			continue
		}
		contract, err := f.extractor.Load(slug, dir)
		if err != nil || !contract.PRDPresent {
			continue
		}
		out = append(out, recordsForScenario(slug, contract)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// recordsForScenario projects one extracted contract into intent records.
func recordsForScenario(slug string, c extraction.Contract) []IntentRecord {
	purpose := prdPurpose(c.PRDDoc)
	var out []IntentRecord
	if purpose != "" {
		// The purpose doc carries the scenario's capability inventory (its
		// OT titles) so interrogative capability queries ("which scenario
		// does X") retrieve it — purpose prose alone rarely names the
		// concrete capabilities.
		capabilities := make([]string, 0, len(c.PRDDoc.Targets))
		for _, ot := range c.PRDDoc.Targets {
			if strings.TrimSpace(ot.Title) != "" {
				capabilities = append(capabilities, strings.TrimSpace(ot.Title))
			}
		}
		snippet := truncate(purpose, 240)
		body := purpose
		if len(capabilities) > 0 {
			body += "\n\nProvides capabilities: " + strings.Join(capabilities, "; ") + "."
		}
		out = append(out, IntentRecord{
			ID:              slug + "/prd",
			Scenario:        slug,
			Type:            TypePRDOverview,
			Title:           slug + " — purpose",
			Snippet:         snippet,
			Anchor:          "scenarios/" + slug + "/PRD.md",
			ScenarioPurpose: body,
		})
	}
	for _, ot := range c.PRDDoc.Targets {
		out = append(out, IntentRecord{
			ID:              slug + "/" + ot.ID,
			Scenario:        slug,
			Type:            TypeOperationalTarget,
			Title:           ot.Title,
			Snippet:         truncate(ot.Description, 240),
			Anchor:          fmt.Sprintf("scenarios/%s/PRD.md#%s", slug, ot.ID),
			ScenarioPurpose: purpose,
		})
	}
	for _, r := range c.Registry.Requirements() {
		if r.ID == "" {
			continue
		}
		out = append(out, IntentRecord{
			ID:              slug + "/" + r.ID,
			Scenario:        slug,
			Type:            TypeRequirement,
			Title:           r.Title,
			Snippet:         truncate(r.Description, 240),
			Anchor:          "scenarios/" + slug + "/" + r.Module,
			PRDRef:          intent.CanonicalOTID(r.PRDRef),
			ScenarioPurpose: purpose,
		})
	}
	return out
}

// prdPurpose extracts the Overview section's content (purpose + value
// promise prose) from the extracted PRD document.
func prdPurpose(doc intent.PRDDocument) string {
	if !doc.Present {
		return ""
	}
	for _, s := range doc.Sections {
		if s.Level == 2 && intent.NormalizeSectionTitle(s.Title) == "overview" {
			return strings.TrimSpace(s.Content)
		}
	}
	return ""
}

// composeIntentEmbeddingText is the measured-default embed-text strategy:
// identity-first (scenario + type + title), the record's own prose, then
// the owning scenario's purpose context (D5: capability queries must land
// on the right scenario even when the record's own text is terse).
func composeIntentEmbeddingText(r IntentRecord) string {
	parts := []string{
		r.Scenario + " " + strings.ReplaceAll(r.Type, "_", " ") + ": " + r.Title,
	}
	if r.Snippet != "" {
		parts = append(parts, r.Snippet)
	}
	if r.PRDRef != "" {
		parts = append(parts, "Serves operational target "+r.PRDRef+".")
	}
	if r.ScenarioPurpose != "" {
		// Purpose docs keep their full body (it carries the capability
		// inventory appended by recordsForScenario — truncating it away
		// would undo the capability-query bridge); OT/requirement records
		// carry a bounded purpose context only.
		limit := 400
		if r.Type == TypePRDOverview {
			limit = 2000
		}
		parts = append(parts, "Scenario purpose: "+truncate(r.ScenarioPurpose, limit))
	}
	return strings.Join(parts, "\n\n")
}

func intentMeta(r IntentRecord) map[string]any {
	meta := map[string]any{
		"record_id": r.ID,
		"scenario":  r.Scenario,
		"type":      r.Type,
		"title":     r.Title,
		"snippet":   r.Snippet,
		"anchor":    r.Anchor,
	}
	if r.PRDRef != "" {
		meta["prd_ref"] = r.PRDRef
	}
	return meta
}

func intentContentHash(r IntentRecord) string {
	data, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func intentToSourceDoc(r IntentRecord, compose func(IntentRecord) string) pkg.SourceDoc {
	return pkg.SourceDoc{
		ID:          r.ID,
		Kind:        intentKind,
		ContentHash: intentContentHash(r),
		Body:        compose(r),
		Meta:        intentMeta(r),
	}
}

// intentSourceAdapter adapts an IntentSource to the engine's Source.
type intentSourceAdapter struct {
	source  IntentSource
	compose func(IntentRecord) string
}

func newIntentSource(source IntentSource, compose func(IntentRecord) string) *intentSourceAdapter {
	if compose == nil {
		compose = composeIntentEmbeddingText
	}
	return &intentSourceAdapter{source: source, compose: compose}
}

func (s *intentSourceAdapter) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	records, err := s.source.LoadRecords(ctx)
	if err != nil {
		return nil, err
	}
	docs := make([]pkg.SourceDoc, 0, len(records))
	for _, r := range records {
		docs = append(docs, intentToSourceDoc(r, s.compose))
	}
	return docs, nil
}

func pointIDForIntent(recordID string) string {
	return pkg.PointIDFor(idPrefix, recordID, 0, 1)
}

func payloadToHit(id string, score float64, payload map[string]any) SearchHit {
	str := func(key string) string {
		v, _ := payload[key].(string)
		return v
	}
	hit := SearchHit{
		ID:           str("record_id"),
		Scenario:     str("scenario"),
		Type:         str("type"),
		Title:        str("title"),
		Snippet:      str("snippet"),
		Anchor:       str("anchor"),
		PRDRef:       str("prd_ref"),
		Score:        score,
		ScorePercent: int(score*100 + 0.5),
	}
	if hit.ID == "" {
		hit.ID = id
	}
	return hit
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
