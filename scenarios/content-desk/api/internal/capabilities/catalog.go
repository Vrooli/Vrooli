package capabilities

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// catalogJSON is the versioned canonical marketing-capability catalog. It is
// derived from MARKETING.md (19 categories) and the Phase 1 owner inventory
// findings; it is the initial seed for this domain, not a second editorial
// ledger. Owner records (campaigns, drafts, claims, releases, metrics) remain
// their own sources of truth.
//
//go:embed catalog.json
var catalogJSON []byte

// catalogNamespace is the fixed UUID namespace used to derive stable capability
// ids from catalog slugs. Using a fixed namespace keeps a capability's id
// identical across processes and re-seeds while still emitting UUID-shaped ids.
var catalogNamespace = uuid.NameSpaceOID

type catalogFile struct {
	Version      int            `json:"version"`
	Source       string         `json:"source"`
	Capabilities []catalogEntry `json:"capabilities"`
}

type catalogEntry struct {
	Slug                     string   `json:"slug"`
	Name                     string   `json:"name"`
	Medium                   string   `json:"medium"`
	Aliases                  []string `json:"aliases"`
	Channels                 []string `json:"channels"`
	AudienceApplicability    string   `json:"audience_applicability"`
	DeliveryApplicability    string   `json:"delivery_applicability"`
	ProducingOperation       string   `json:"producing_operation"`
	Prerequisites            []string `json:"prerequisites"`
	Priority                 int32    `json:"priority"`
	PriorityReason           string   `json:"priority_reason"`
	PriorityScope            string   `json:"priority_scope"`
	DefinitionStatus         string   `json:"definition_status"`
	ImplementationStatus     string   `json:"implementation_status"`
	OperationalReadiness     string   `json:"operational_readiness"`
	OutputQuality            string   `json:"output_quality"`
	DistributionConnectivity string   `json:"distribution_connectivity"`
	Owner                    string   `json:"owner"`
	SourceRefs               []string `json:"source_refs"`
	NextAction               string   `json:"next_action"`
}

// CanonicalCatalog returns the embedded capability catalog with deterministic
// ids derived from each entry's slug. The result is validated so a malformed
// seed fails closed rather than persisting an invalid readiness dimension.
func CanonicalCatalog() ([]Capability, error) {
	var file catalogFile
	if err := json.Unmarshal(catalogJSON, &file); err != nil {
		return nil, fmt.Errorf("decode capability catalog: %w", err)
	}
	if file.Version != 1 {
		return nil, fmt.Errorf("unsupported capability catalog version %d", file.Version)
	}
	if len(file.Capabilities) == 0 {
		return nil, fmt.Errorf("capability catalog is empty")
	}
	seen := make(map[string]struct{}, len(file.Capabilities))
	out := make([]Capability, 0, len(file.Capabilities))
	for _, entry := range file.Capabilities {
		slug := strings.TrimSpace(entry.Slug)
		if slug == "" {
			return nil, fmt.Errorf("capability catalog entry missing slug")
		}
		if _, ok := seen[slug]; ok {
			return nil, fmt.Errorf("duplicate capability catalog slug %q", slug)
		}
		seen[slug] = struct{}{}
		aliases := make([]string, 0, len(entry.Aliases)+1)
		aliases = append(aliases, slug)
		aliases = append(aliases, entry.Aliases...)
		aliases = dedupeStrings(aliases)
		capability := Capability{
			ID:                       uuid.NewSHA1(catalogNamespace, []byte("content-desk:capability:"+slug)).String(),
			Name:                     entry.Name,
			Medium:                   entry.Medium,
			Aliases:                  aliases,
			Channels:                 entry.Channels,
			AudienceApplicability:    entry.AudienceApplicability,
			DeliveryApplicability:    entry.DeliveryApplicability,
			ProducingOperation:       entry.ProducingOperation,
			Prerequisites:            entry.Prerequisites,
			Priority:                 entry.Priority,
			PriorityReason:           entry.PriorityReason,
			PriorityScope:            entry.PriorityScope,
			DefinitionStatus:         entry.DefinitionStatus,
			ImplementationStatus:     entry.ImplementationStatus,
			OperationalReadiness:     entry.OperationalReadiness,
			OutputQuality:            entry.OutputQuality,
			DistributionConnectivity: entry.DistributionConnectivity,
			Owner:                    entry.Owner,
			SourceRefs:               entry.SourceRefs,
			NextAction:               entry.NextAction,
		}
		if err := validateCapability(capability); err != nil {
			return nil, fmt.Errorf("capability catalog entry %q: %w", slug, err)
		}
		out = append(out, capability)
	}
	return out, nil
}

// SeedResult reports what a catalog seed inserted versus left untouched.
type SeedResult struct {
	Inserted int
	Existing int
}

// SeedCatalog inserts the canonical catalog through the owning repository. It
// never overwrites an existing capability, so operator edits and observed
// qualifications survive later restarts.
func SeedCatalog(ctx context.Context, repo Repository) (SeedResult, error) {
	catalog, err := CanonicalCatalog()
	if err != nil {
		return SeedResult{}, err
	}
	return repo.Seed(ctx, catalog)
}
