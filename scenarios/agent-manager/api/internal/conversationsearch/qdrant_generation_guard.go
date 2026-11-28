package conversationsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

// The Qdrant generation catalog must name exactly one serving generation per
// (namespace, alias). The 2026-11-11 defect left 6,086 rows 'active' because
// every promotion retired only the alias target it could still find, so both
// the catalog and the semantic path have to tolerate and repair legacy
// duplicates before the invariant can be trusted.
const (
	// qdrantOneActiveGenerationIndex is the durable one-active invariant. It is
	// installed by the owner (not by schema.sql) because a live catalog can
	// contain legacy duplicate active rows that a boot-time unique index would
	// reject, which would abort schema application instead of repairing it.
	qdrantOneActiveGenerationIndex = "idx_conversation_search_qdrant_one_active_generation"
	// generationReconcileTimeout bounds the owner repair so a status request
	// cannot hang behind a locked database.
	generationReconcileTimeout = 30 * time.Second
	// generationHealthRefreshInterval avoids reconciling (and potentially
	// writing) on every status poll.
	generationHealthRefreshInterval = time.Minute
	// aliasLookupTimeout bounds the read-only Qdrant alias probe used as
	// serving truth during reconciliation.
	aliasLookupTimeout = 3 * time.Second
)

// ErrAmbiguousActiveGenerations refuses to guess when several complete active
// generations carry different immutable embedding recipes and no evidence
// (Qdrant alias, explicit operator reconciliation) says which one serves.
var ErrAmbiguousActiveGenerations = errors.New("multiple complete active conversation generations record different embedding recipes")

// Generation reconciliation health states. They are deliberately explicit:
// "unknown" and "unavailable" never mean healthy.
const (
	GenerationHealthHealthy       = "healthy"
	GenerationHealthRecovered     = "recovered"
	GenerationHealthMissingRecipe = "missing-recipe"
	GenerationHealthAmbiguous     = "ambiguous"
	GenerationHealthUnavailable   = "unavailable"
	GenerationHealthUnknown       = "unknown"
)

// GenerationReconcileOptions carries the evidence a caller may supply while
// reconciling. AuthoritativeCollection is the Qdrant alias target; when it
// matches an active row it settles divergent recipes without guessing.
// AllowRecipeDivergence is an explicit operator decision and must never be set
// by an automatic path.
type GenerationReconcileOptions struct {
	AuthoritativeCollection string
	AllowRecipeDivergence   bool
	Reason                  string
}

// RetiredActiveGeneration is one duplicate active row retired by
// reconciliation. The physical collection and the catalog row are retained.
type RetiredActiveGeneration struct {
	GenerationID   string `json:"generationId"`
	CollectionName string `json:"collectionName"`
	Reason         string `json:"reason"`
}

// ActiveGenerationReconciliation is the auditable result of one pass. It is
// also the health record surfaced through index status.
type ActiveGenerationReconciliation struct {
	Namespace          string                    `json:"namespace"`
	Alias              string                    `json:"alias"`
	ObservedAt         time.Time                 `json:"observedAt"`
	Reason             string                    `json:"reason"`
	ActiveBefore       int                       `json:"activeBefore"`
	SurvivorID         string                    `json:"survivorId,omitempty"`
	SurvivorCollection string                    `json:"survivorCollection,omitempty"`
	RecipeIdentity     string                    `json:"recipeIdentity,omitempty"`
	RecipeValid        bool                      `json:"recipeValid"`
	Retired            []RetiredActiveGeneration `json:"retired,omitempty"`
	Findings           []string                  `json:"findings,omitempty"`
	Ambiguous          bool                      `json:"ambiguous"`
	InvariantPresent   bool                      `json:"invariantPresent"`
}

// ActiveGenerationReconciler is implemented by catalogs that can repair their
// own one-active invariant. The runtime and the CLI degrade explicitly (never
// silently) when a catalog does not offer it.
type ActiveGenerationReconciler interface {
	ReconcileActiveGenerations(context.Context, string, string, GenerationReconcileOptions) (ActiveGenerationReconciliation, error)
}

// SemanticGenerationHealth is the semantic leg's generation health. State is
// one of the GenerationHealth* constants; Degradation() is empty only when the
// leg is genuinely healthy.
type SemanticGenerationHealth struct {
	State              string `json:"state"`
	Detail             string `json:"detail,omitempty"`
	SurvivorID         string `json:"survivorId,omitempty"`
	SurvivorCollection string `json:"survivorCollection,omitempty"`
	RecipeValid        bool   `json:"recipeValid"`
	ActiveGenerations  int    `json:"activeGenerations"`
	RetiredDuplicates  int    `json:"retiredDuplicates"`
	EmbeddedDocuments  uint64 `json:"embeddedDocuments,omitempty"`
	// EmbeddedObserved is true once a vector count has been read; before that
	// EmbeddedDocuments is absent, not zero.
	EmbeddedObserved  bool      `json:"embeddedObserved,omitempty"`
	EligibleDocuments uint64    `json:"eligibleDocuments,omitempty"`
	ObservedAt        time.Time `json:"observedAt,omitempty"`
	Findings          []string  `json:"findings,omitempty"`
}

// Degradation returns the status line for this health, or "" when healthy.
func (h SemanticGenerationHealth) Degradation() string {
	coverage := ""
	if h.EligibleDocuments > 0 && !h.EmbeddedObserved {
		coverage = fmt.Sprintf("; vector coverage not yet observed (%d semantic-eligible chunks)", h.EligibleDocuments)
	} else if h.EligibleDocuments > 0 {
		coverage = fmt.Sprintf("; vectors %d/%d semantic-eligible chunks (%.1f%%; the catalog denominator also counts non-embeddable tool payloads)",
			h.EmbeddedDocuments, h.EligibleDocuments, 100*float64(h.EmbeddedDocuments)/float64(h.EligibleDocuments))
	}
	switch h.State {
	case "", GenerationHealthHealthy:
		return ""
	case GenerationHealthRecovered:
		return fmt.Sprintf("semantic-generation: %s%s", h.Detail, coverage)
	default:
		return fmt.Sprintf("semantic-generation: %s: %s%s", h.State, h.Detail, coverage)
	}
}

// SemanticGenerationHealthReporter exposes generation health to index status.
type SemanticGenerationHealthReporter interface {
	GenerationHealthSnapshot() SemanticGenerationHealth
}

// activeGenerationChoice is the pure selection result shared by reconciliation
// (which persists it) and loadActiveRecipeRecord (which must tolerate a
// catalog that cannot persist).
type activeGenerationChoice struct {
	Survivor       *aisearch.GenerationRecord
	Retire         []aisearch.GenerationRecord
	RecipeIdentity string
	RecipeValid    bool
	Findings       []string
	Ambiguous      bool
}

// generationRecipeIdentity is the complete serving identity of a generation.
// Two active rows with the same identity serve identical vectors, so keeping
// the newest is safe; a different identity means they cannot both be serving.
func generationRecipeIdentity(metadata aisearch.GenerationMetadata) string {
	return strings.Join([]string{
		strings.TrimSpace(metadata.RecipeID), strings.TrimSpace(metadata.Model),
		strings.TrimSpace(metadata.EmbeddingRole), fmt.Sprintf("%d", metadata.EmbeddingDimensions),
		strings.TrimSpace(metadata.PolicySchemaVersion), strings.TrimSpace(metadata.ChunkPolicy),
		fmt.Sprintf("%t", metadata.EmbeddingTaskPrefix),
	}, "\x00")
}

// generationServingRank ranks an active row by the identity it carries.
//
//	0 no usable identity (drifting/no recipe columns at all)
//	1 usable serving identity (model, role, dimensions, policy, chunk policy)
//	  but no immutable recipe identity, which is the legacy pre-recipe shape
//	2 verified immutable recipe identity (id + canonical JSON agree with it)
//
// Rank 1 still serves real vectors; it simply cannot be provenance-verified.
// Treating it as unusable would disable search on every pre-recipe generation.
func generationServingRank(metadata aisearch.GenerationMetadata) int {
	if metadata.EmbeddingDimensions <= 0 || strings.TrimSpace(metadata.Model) == "" ||
		strings.TrimSpace(metadata.EmbeddingRole) == "" || strings.TrimSpace(metadata.PolicySchemaVersion) == "" ||
		strings.TrimSpace(metadata.ChunkPolicy) == "" {
		return 0
	}
	if verifyGenerationRecipe(metadata) == nil {
		return 2
	}
	return 1
}

// generationServingIdentity reports whether the row describes usable vectors.
func generationServingIdentity(metadata aisearch.GenerationMetadata) bool {
	return generationServingRank(metadata) > 0
}

// chooseActiveGeneration decides which active row serves and which are stale.
//
// Precedence:
//  1. a verified recipe beats a legacy serving identity, which beats no identity;
//  2. among rows at the same rank, the newest wins;
//  3. divergent verified recipes are ambiguous unless the Qdrant alias names
//     exactly one of them (evidence) or the caller explicitly allows it;
//  4. when no active row carries a usable identity, the alias target is retained
//     so the runtime can adopt the configured policy, otherwise every row is
//     retired so a fresh generation is built.
func chooseActiveGeneration(actives []aisearch.GenerationRecord, authoritativeCollection string, allowDivergence bool) (activeGenerationChoice, error) {
	choice := activeGenerationChoice{}
	if len(actives) == 0 {
		choice.Findings = append(choice.Findings, "no active conversation generation is recorded")
		return choice, nil
	}
	ordered := append([]aisearch.GenerationRecord(nil), actives...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if !ordered[i].Metadata.CreatedAt.Equal(ordered[j].Metadata.CreatedAt) {
			return ordered[i].Metadata.CreatedAt.After(ordered[j].Metadata.CreatedAt)
		}
		return ordered[i].Metadata.ID > ordered[j].Metadata.ID
	})
	verified := make([]aisearch.GenerationRecord, 0, len(ordered))
	identified := make([]aisearch.GenerationRecord, 0, len(ordered))
	for _, record := range ordered {
		switch generationServingRank(record.Metadata) {
		case 2:
			verified = append(verified, record)
			identified = append(identified, record)
		case 1:
			identified = append(identified, record)
		}
	}
	authoritative := func(candidates []aisearch.GenerationRecord) *aisearch.GenerationRecord {
		if authoritativeCollection == "" {
			return nil
		}
		for i := range candidates {
			if candidates[i].CollectionName == authoritativeCollection {
				return &candidates[i]
			}
		}
		return nil
	}
	switch {
	case len(identified) == 0:
		if kept := authoritative(ordered); kept != nil {
			choice.Survivor = kept
			choice.Findings = append(choice.Findings, fmt.Sprintf(
				"active generation %q has no usable embedding identity; retained because the Qdrant alias serves it",
				kept.Metadata.ID))
		} else {
			choice.Findings = append(choice.Findings, fmt.Sprintf(
				"%d active generation(s) have no usable embedding identity; retiring them so a fresh generation can be built",
				len(ordered)))
		}
	case len(verified) == 0:
		// Legacy pre-recipe generations: the Qdrant sentinel proves they serve,
		// so the alias target (else the newest) keeps serving while the runtime
		// adopts its identity.
		choice.Survivor = authoritative(identified)
		if choice.Survivor == nil {
			choice.Survivor = &identified[0]
		}
		choice.Findings = append(choice.Findings, fmt.Sprintf(
			"serving generation %q records no immutable recipe identity (legacy); its embedding identity is adopted, not invented",
			choice.Survivor.Metadata.ID))
	case len(verified) == 1:
		choice.Survivor = &verified[0]
		choice.RecipeValid = true
		if len(ordered) > 1 {
			choice.Findings = append(choice.Findings, fmt.Sprintf(
				"retained the only active generation with a verified recipe (%q) among %d active rows",
				verified[0].Metadata.ID, len(ordered)))
		}
	default:
		identities := make([]string, 0, len(verified))
		byIdentity := make(map[string]struct{}, len(verified))
		for _, record := range verified {
			identity := generationRecipeIdentity(record.Metadata)
			if _, seen := byIdentity[identity]; seen {
				continue
			}
			byIdentity[identity] = struct{}{}
			label := strings.TrimSpace(record.Metadata.RecipeID)
			if label == "" {
				label = record.Metadata.ID
			}
			identities = append(identities, label)
		}
		sort.Strings(identities)
		switch {
		case len(identities) > 1:
			switch target := authoritative(verified); {
			case target != nil:
				choice.Survivor = target
				choice.Findings = append(choice.Findings, fmt.Sprintf(
					"%d complete active generations disagree on the embedding recipe; the Qdrant alias serves %q so that recipe wins",
					len(verified), target.CollectionName))
			case allowDivergence:
				choice.Survivor = &verified[0]
				choice.Findings = append(choice.Findings, fmt.Sprintf(
					"%d complete active generations disagree on the embedding recipe; explicit reconciliation keeps the newest (%q)",
					len(verified), verified[0].Metadata.ID))
			default:
				choice.Ambiguous = true
				choice.Findings = append(choice.Findings, fmt.Sprintf(
					"%d complete active generations disagree on the embedding recipe (%s); refusing to guess without an authoritative alias target or explicit reconciliation",
					len(verified), strings.Join(identities, ", ")))
				return choice, fmt.Errorf("%w: %s", ErrAmbiguousActiveGenerations, strings.Join(identities, ", "))
			}
		default:
			choice.Survivor = &verified[0]
			choice.Findings = append(choice.Findings, fmt.Sprintf(
				"%d complete active generations share one embedding recipe; keeping the newest (%q)",
				len(verified), verified[0].Metadata.ID))
		}
		choice.RecipeValid = true
	}
	if choice.Survivor != nil {
		choice.RecipeIdentity = generationRecipeIdentity(choice.Survivor.Metadata)
	}
	for _, record := range ordered {
		if choice.Survivor != nil && record.Metadata.ID == choice.Survivor.Metadata.ID {
			continue
		}
		choice.Retire = append(choice.Retire, record)
	}
	return choice, nil
}

// retirementReason records why one duplicate active row stopped serving.
func retirementReason(choice activeGenerationChoice, record aisearch.GenerationRecord) string {
	survivorID := ""
	if choice.Survivor != nil {
		survivorID = choice.Survivor.Metadata.ID
	}
	switch {
	case choice.Survivor == nil:
		return "duplicate-active: no usable identity; retired so a fresh generation can be built"
	case !generationServingIdentity(record.Metadata):
		return fmt.Sprintf("duplicate-active: incomplete identity; superseded by %s", survivorID)
	case generationRecipeIdentity(record.Metadata) != generationRecipeIdentity(choice.Survivor.Metadata):
		return fmt.Sprintf("duplicate-active: divergent recipe; superseded by %s", survivorID)
	default:
		return fmt.Sprintf("duplicate-active: same recipe; superseded by %s", survivorID)
	}
}

// ensureOneActiveGenerationInvariant installs the partial unique index. It is
// idempotent and must run after duplicates have been retired, inside the same
// transaction, so the invariant and the repair commit together.
// oneActiveInvariantPresent is a read-only check: the fast path for a healthy
// catalog that has no duplicate rows to retire.
func (c *SQLiteQdrantCatalog) oneActiveInvariantPresent(ctx context.Context) (bool, error) {
	if c == nil || c.db == nil {
		return false, errors.New("conversation generation catalog is unavailable")
	}
	var count int
	if err := c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, qdrantOneActiveGenerationIndex).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func ensureOneActiveGenerationInvariant(ctx context.Context, execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
},
) error {
	_, err := execer.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS `+qdrantOneActiveGenerationIndex+`
        ON conversation_search_qdrant_generations(namespace, alias_name) WHERE state = 'active'`)
	if err != nil {
		return fmt.Errorf("install the one-active conversation generation invariant (reconcile duplicate active rows first): %w", err)
	}
	return nil
}

// ReconcileActiveGenerations makes the one-active invariant true and records
// why every retired duplicate stopped serving. Rows and physical collections
// are never deleted here.
func (c *SQLiteQdrantCatalog) ReconcileActiveGenerations(ctx context.Context, namespace, alias string, options GenerationReconcileOptions) (ActiveGenerationReconciliation, error) {
	now := time.Now().UTC()
	report := ActiveGenerationReconciliation{
		Namespace: strings.TrimSpace(namespace), Alias: strings.TrimSpace(alias),
		ObservedAt: now, Reason: strings.TrimSpace(options.Reason),
	}
	if c == nil || c.db == nil {
		return report, errors.New("conversation generation catalog is unavailable")
	}
	if report.Namespace == "" || report.Alias == "" {
		return report, errors.New("conversation generation reconciliation requires namespace and alias")
	}
	actives, err := c.listActiveGenerationRecords(ctx, report.Namespace, report.Alias)
	if err != nil {
		return report, err
	}
	report.ActiveBefore = len(actives)
	choice, chooseErr := chooseActiveGeneration(actives, strings.TrimSpace(options.AuthoritativeCollection), options.AllowRecipeDivergence)
	report.Findings = append(report.Findings, choice.Findings...)
	report.Ambiguous = choice.Ambiguous
	report.RecipeValid = choice.RecipeValid
	if choice.Survivor != nil {
		report.SurvivorID = choice.Survivor.Metadata.ID
		report.SurvivorCollection = choice.Survivor.CollectionName
		report.RecipeIdentity = choice.RecipeIdentity
	}
	if chooseErr != nil {
		return report, chooseErr
	}
	// Steady state is one active row and no retirement work. A status read must
	// not open a write transaction (and contend with the indexer's single
	// writer) just to confirm that. Read the invariant's presence instead; only
	// a real duplicate or a missing invariant needs the transaction.
	if len(choice.Retire) == 0 {
		if present, invariantErr := c.oneActiveInvariantPresent(ctx); invariantErr == nil && present {
			report.InvariantPresent = true
			return report, nil
		}
	}

	conn, err := c.db.Conn(ctx)
	if err != nil {
		return report, fmt.Errorf("open conversation generation catalog transaction: %w", err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return report, fmt.Errorf("begin conversation generation reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, record := range choice.Retire {
		reason := retirementReason(choice, record)
		result, err := tx.ExecContext(ctx, `UPDATE conversation_search_qdrant_generations
            SET state = 'retired', state_reason = ?, superseded_by = ?, state_changed_at = ?, updated_at = ?
            WHERE generation_id = ? AND namespace = ? AND alias_name = ? AND state = 'active'`,
			reason, report.SurvivorID, formatTime(now), formatTime(now), record.Metadata.ID, report.Namespace, report.Alias)
		if err != nil {
			return report, fmt.Errorf("retire duplicate active generation %q: %w", record.Metadata.ID, err)
		}
		if changed, rowsErr := result.RowsAffected(); rowsErr == nil && changed == 0 {
			// A concurrent owner already retired or replaced it; nothing to claim.
			continue
		}
		report.Retired = append(report.Retired, RetiredActiveGeneration{
			GenerationID: record.Metadata.ID, CollectionName: record.CollectionName, Reason: reason,
		})
	}
	if err := ensureOneActiveGenerationInvariant(ctx, tx); err != nil {
		return report, err
	}
	if err := tx.Commit(); err != nil {
		return report, fmt.Errorf("commit conversation generation reconciliation: %w", err)
	}
	report.InvariantPresent = true
	if len(report.Retired) > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf(
			"retired %d duplicate active generation(s) for %q (rows retained as evidence)",
			len(report.Retired), report.Alias))
	}
	return report, nil
}

func (c *SQLiteQdrantCatalog) listActiveGenerationRecords(ctx context.Context, namespace, alias string) ([]aisearch.GenerationRecord, error) {
	var rows []qdrantGenerationRow
	if err := c.db.SelectContext(ctx, &rows, `SELECT * FROM conversation_search_qdrant_generations
        WHERE namespace = ? AND alias_name = ? AND state = 'active'
        ORDER BY created_at DESC, generation_id DESC`, namespace, alias); err != nil {
		return nil, fmt.Errorf("list active conversation generations: %w", err)
	}
	records := make([]aisearch.GenerationRecord, 0, len(rows))
	for _, row := range rows {
		record, err := row.record()
		if err != nil {
			return nil, fmt.Errorf("decode active conversation generation %q: %w", row.GenerationID, err)
		}
		records = append(records, record)
	}
	return records, nil
}

// reconcileGenerationCatalog reconciles when the catalog supports it and
// reports whether it does. A catalog without the seam is reported as unknown,
// never as healthy.
func reconcileGenerationCatalog(ctx context.Context, catalog aisearch.GenerationCatalog, alias string, options GenerationReconcileOptions) (ActiveGenerationReconciliation, bool, error) {
	reconciler, ok := catalog.(ActiveGenerationReconciler)
	if !ok {
		return ActiveGenerationReconciliation{}, false, nil
	}
	report, err := reconciler.ReconcileActiveGenerations(ctx, conversationCollectionDomain, alias, options)
	return report, true, err
}

// resolveServingAliasCollection reads the Qdrant alias target. It is the only
// external evidence of which physical collection actually serves, so it is
// used to settle otherwise ambiguous catalogs. Failure is not fatal: an
// unreachable Qdrant leaves the recipe choice to the catalog alone.
func resolveServingAliasCollection(parent context.Context, baseURL, apiKey, alias string) string {
	baseURL = strings.TrimSpace(baseURL)
	alias = strings.TrimSpace(alias)
	if baseURL == "" || alias == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(parent, aliasLookupTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/aliases", nil)
	if err != nil {
		return ""
	}
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		request.Header.Set("api-key", apiKey)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ""
	}
	var payload struct {
		Result struct {
			Aliases []struct {
				AliasName      string `json:"alias_name"`
				CollectionName string `json:"collection_name"`
			} `json:"aliases"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ""
	}
	for _, item := range payload.Result.Aliases {
		if item.AliasName == alias {
			return item.CollectionName
		}
	}
	return ""
}

var _ ActiveGenerationReconciler = (*SQLiteQdrantCatalog)(nil)
