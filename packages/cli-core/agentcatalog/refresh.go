package agentcatalog

// This file refreshes a resource's model vocabulary from evidence on this
// machine. It never adds a role or changes which tier a role uses, and it fails
// without writing when the installed runner cannot prove what it offers.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/cliutil"
)

// ErrRefreshEvidenceUnavailable means nothing local can prove the catalog.
var ErrRefreshEvidenceUnavailable = errors.New("no verifiable local model evidence")

// CatalogRefreshResult reports one refresh. Changes is empty when the only
// update was the observation date.
type CatalogRefreshResult struct {
	Runner     string   `json:"runner"`
	PolicyPath string   `json:"policy_path"`
	Evidence   string   `json:"evidence"`
	ObservedAt string   `json:"observed_at"`
	Changes    []string `json:"changes"`
	Warnings   []string `json:"warnings"`
}

// ModelContextWindow returns the harness-reported context window of a catalog
// model, resolving an alias through the catalog. Zero means unknown.
func ModelContextWindow(descriptor harness.Descriptor, catalog CodingRoleCatalog, model string) int64 {
	if alias, ok := catalog.ModelAliases[model]; ok {
		canonical := alias.CanonicalModel
		if _, id, qualified := strings.Cut(canonical, "/"); qualified {
			canonical = id
		}
		return descriptor.ContextWindowTokens(canonical, model)
	}
	return descriptor.ContextWindowTokens(model)
}

// CanRefreshCatalog says whether this runner has an evidence reader. A policy
// without one is updated by hand.
func CanRefreshCatalog(runner string) bool {
	return runner == "claude-code" || runner == "codex"
}

// RefreshCodingRoleCatalog rewrites the policy at path from the installed
// runner's own evidence. Only runners with an evidence reader are supported.
func RefreshCodingRoleCatalog(ctx context.Context, runner, path string, now time.Time) (CatalogRefreshResult, error) {
	return RefreshCodingRoleCatalogBy(ctx, runner, path, now, "")
}

// RefreshCodingRoleCatalogBy also names who asked in the policy's provenance.
func RefreshCodingRoleCatalogBy(ctx context.Context, runner, path string, now time.Time, actor string) (CatalogRefreshResult, error) {
	catalog, _, err := loadCodingRoleCatalog(runner, path)
	if err != nil {
		return CatalogRefreshResult{}, err
	}
	result := CatalogRefreshResult{Runner: runner, PolicyPath: path, ObservedAt: now.UTC().Format("2006-01-02"), Changes: []string{}, Warnings: []string{}}
	switch runner {
	case "claude-code":
		evidence, err := readClaudeModelEvidence(ctx)
		if err != nil {
			return CatalogRefreshResult{}, err
		}
		if err := applyClaudeModelEvidence(&catalog, evidence, &result); err != nil {
			return CatalogRefreshResult{}, err
		}
		result.Evidence = "Claude Code " + evidence.Version + " built-in model catalog"
	case "codex":
		live, err := DiscoverModels(ctx, runner)
		if err != nil {
			return CatalogRefreshResult{}, fmt.Errorf("%w: %w", ErrRefreshEvidenceUnavailable, err)
		}
		if !live.IsAuthoritative() {
			return CatalogRefreshResult{}, fmt.Errorf("%w: %s is a cache, not the installed runner", ErrRefreshEvidenceUnavailable, live.Source)
		}
		if err := applyLiveModelEvidence(&catalog, live, &result); err != nil {
			return CatalogRefreshResult{}, err
		}
		result.Evidence = strings.TrimSpace(live.RunnerVersion + " " + live.Source)
	default:
		return CatalogRefreshResult{}, fmt.Errorf("%w: runner %q has no evidence reader; update its policy by hand", ErrRefreshEvidenceUnavailable, runner)
	}
	catalog.Provenance = CatalogProvenance{Source: refreshedProvenance(catalog.Provenance.Source, result.Evidence, actor), ObservedAt: result.ObservedAt}
	if err := validateCodingRoleCatalog(catalog, runner); err != nil {
		return CatalogRefreshResult{}, fmt.Errorf("refreshed policy is invalid: %w", err)
	}
	if err := writeCatalogAtomically(path, catalog); err != nil {
		return CatalogRefreshResult{}, err
	}
	sort.Strings(result.Changes)
	return result, nil
}

const (
	refreshProvenancePrefix  = "policy refresh "
	refreshProvenanceEarlier = "; earlier: "
)

// refreshedProvenance names the new evidence and keeps the policy's standing
// rationale, which a refresh cannot re-derive, without stacking older refreshes.
func refreshedProvenance(previous, evidence, actor string) string {
	if strings.HasPrefix(previous, refreshProvenancePrefix) {
		_, previous, _ = strings.Cut(previous, refreshProvenanceEarlier)
	}
	source := refreshProvenancePrefix
	if actor = strings.TrimSpace(actor); actor != "" {
		source += "by " + actor + " "
	}
	source += "from " + evidence
	if previous = strings.TrimSpace(previous); previous != "" {
		source += refreshProvenanceEarlier + previous
	}
	return source
}

func writeCatalogAtomically(path string, catalog CodingRoleCatalog) error {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(catalog); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(encoded.Bytes()); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(info.Mode().Perm()); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

// claudeModelEvidence is what the installed Claude Code binary says it offers.
type claudeModelEvidence struct {
	Version string
	// Families maps each model ID to its family, such as opus.
	Families map[string]string
	// Aliases maps each alias to the model it selects on the first-party API.
	Aliases map[string]string
}

var (
	claudeCatalogModel = regexp.MustCompile(`\{id:"(claude-[a-z0-9.-]+)",family:"([a-z0-9]+)",display_name:"`)
	claudeCatalogAlias = regexp.MustCompile(`aliases:\{((?:[a-z0-9]+:\{default:"[^"]+"(?:,per_provider:\{[^}]*\})?\},?)+)\}`)
	claudeAliasDefault = regexp.MustCompile(`([a-z0-9]+):\{default:"(claude-[a-z0-9.-]+)"`)
)

func readClaudeModelEvidence(ctx context.Context) (claudeModelEvidence, error) {
	path, err := cliutil.ResolveAgentBinary("claude")
	if err != nil {
		return claudeModelEvidence{}, fmt.Errorf("%w: claude: %v", ErrRefreshEvidenceUnavailable, err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return claudeModelEvidence{}, fmt.Errorf("%w: claude: %v", ErrRefreshEvidenceUnavailable, err)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return claudeModelEvidence{}, fmt.Errorf("%w: read %s: %v", ErrRefreshEvidenceUnavailable, resolved, err)
	}
	evidence, err := parseClaudeModelEvidence(data)
	if err != nil {
		return claudeModelEvidence{}, fmt.Errorf("%w: %s: %v", ErrRefreshEvidenceUnavailable, resolved, err)
	}
	output, err := exec.CommandContext(ctx, resolved, "--version").Output()
	if err != nil {
		return claudeModelEvidence{}, fmt.Errorf("%w: %s --version: %v", ErrRefreshEvidenceUnavailable, resolved, err)
	}
	evidence.Version = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(string(output)), "(Claude Code)"))
	return evidence, nil
}

// parseClaudeModelEvidence reads the model catalog Claude Code compiles in.
// Every copy in the binary must agree, and a release that moved the catalog
// fails here rather than leaving a partial policy.
func parseClaudeModelEvidence(data []byte) (claudeModelEvidence, error) {
	evidence := claudeModelEvidence{Families: map[string]string{}, Aliases: map[string]string{}}
	for _, match := range claudeCatalogModel.FindAllSubmatch(data, -1) {
		evidence.Families[string(match[1])] = string(match[2])
	}
	if len(evidence.Families) == 0 {
		return evidence, errors.New("no built-in model catalog was found")
	}
	blocks := claudeCatalogAlias.FindAllSubmatch(data, -1)
	if len(blocks) == 0 {
		return evidence, errors.New("no built-in model aliases were found")
	}
	for _, block := range blocks {
		for _, match := range claudeAliasDefault.FindAllSubmatch(block[1], -1) {
			alias, model := string(match[1]), string(match[2])
			if existing, seen := evidence.Aliases[alias]; seen && existing != model {
				return evidence, fmt.Errorf("alias %q selects both %s and %s", alias, existing, model)
			}
			if _, known := evidence.Families[model]; !known {
				return evidence, fmt.Errorf("alias %q selects %s, which the catalog does not list", alias, model)
			}
			evidence.Aliases[alias] = model
		}
	}
	if len(evidence.Aliases) == 0 {
		return evidence, errors.New("the built-in model aliases name no default models")
	}
	return evidence, nil
}

func applyClaudeModelEvidence(catalog *CodingRoleCatalog, evidence claudeModelEvidence, result *CatalogRefreshResult) error {
	latest := func(model string) string {
		if family, known := evidence.Families[model]; known {
			if current, ok := evidence.Aliases[family]; ok {
				return current
			}
		}
		return model
	}
	offered := func(model string) bool {
		_, isAlias := evidence.Aliases[model]
		_, isModel := evidence.Families[model]
		return isAlias || isModel
	}
	// Every Claude-served model a role names must still exist, before anything
	// moves. Other sources are served elsewhere and prove nothing here.
	for name, role := range catalog.Roles {
		if !offered(role.Model) {
			return fmt.Errorf("%w: role %q uses %q, which the installed Claude Code does not offer", ErrRefreshEvidenceUnavailable, name, role.Model)
		}
		for _, declared := range role.Models {
			if declared.Source == "claude-subscription" && !offered(declared.Model) {
				return fmt.Errorf("%w: role %q declares %q, which the installed Claude Code does not offer", ErrRefreshEvidenceUnavailable, name, declared.Model)
			}
		}
	}
	if catalog.ModelAliases == nil {
		catalog.ModelAliases = map[string]ModelAlias{}
	}
	setAlias := func(name, canonical string) {
		if catalog.ModelAliases[name].CanonicalModel != canonical {
			result.Changes = append(result.Changes, fmt.Sprintf("alias %s -> %s", name, canonical))
			catalog.ModelAliases[name] = ModelAlias{CanonicalModel: canonical, Provider: "anthropic"}
		}
	}
	for alias, model := range evidence.Aliases {
		setAlias(alias, "anthropic/"+model)
		setAlias(model, "anthropic/"+model)
	}
	for name, role := range catalog.Roles {
		if model, isAlias := evidence.Aliases[role.Model]; isAlias && role.CanonicalModel != "anthropic/"+model {
			result.Changes = append(result.Changes, fmt.Sprintf("role %s canonical model -> anthropic/%s", name, model))
			role.CanonicalModel = "anthropic/" + model
		}
		fallbacks := make([]string, 0, len(role.Fallbacks))
		for _, fallback := range role.Fallbacks {
			if current := latest(fallback); !slices.Contains(fallbacks, current) {
				fallbacks = append(fallbacks, current)
			}
		}
		if !slices.Equal(fallbacks, role.Fallbacks) {
			result.Changes = append(result.Changes, fmt.Sprintf("role %s fallbacks -> %s", name, strings.Join(fallbacks, ", ")))
			role.Fallbacks = fallbacks
		}
		catalog.Roles[name] = role
	}
	if catalog.UsageTiers != nil {
		for from, to := range catalog.UsageTiers.ModelMap {
			if current := latest(to); current != to && current != from {
				result.Changes = append(result.Changes, fmt.Sprintf("usage tier %s -> %s", from, current))
				catalog.UsageTiers.ModelMap[from] = current
			}
		}
		// An alias edge states the tier order, so it also orders what the aliases
		// now select: fable -> opus covers claude-fable-5-1 -> claude-opus-5-5.
		for from, to := range catalog.UsageTiers.ModelMap {
			fromModel, fromAlias := evidence.Aliases[from]
			toModel, toAlias := evidence.Aliases[to]
			if fromAlias && toAlias && catalog.UsageTiers.ModelMap[fromModel] != toModel {
				result.Changes = append(result.Changes, fmt.Sprintf("usage tier %s -> %s", fromModel, toModel))
				catalog.UsageTiers.ModelMap[fromModel] = toModel
			}
		}
	}
	return nil
}

// applyLiveModelEvidence confirms role models against a runner's live listing
// and names what the listing adds. A primary model it cannot confirm fails the
// refresh, because a new observation date would claim otherwise.
func applyLiveModelEvidence(catalog *CodingRoleCatalog, live LiveModelCatalog, result *CatalogRefreshResult) error {
	for _, finding := range liveCatalogFindings(*catalog, live) {
		switch finding.Type {
		case "unnamed_live_model":
		case "missing_fallback_model":
			result.Warnings = append(result.Warnings, fmt.Sprintf("role %s: %s: %s", finding.Role, finding.Model, finding.Message))
		default:
			return fmt.Errorf("%w: role %q model %q: %s", ErrRefreshEvidenceUnavailable, finding.Role, finding.Model, finding.Message)
		}
	}
	if catalog.ModelAliases == nil {
		catalog.ModelAliases = map[string]ModelAlias{}
	}
	for _, model := range live.Models {
		if _, named := catalog.ModelAliases[model]; !named {
			catalog.ModelAliases[model] = ModelAlias{CanonicalModel: catalog.Runner + "/" + model, Provider: catalog.Runner}
			result.Changes = append(result.Changes, "alias "+model+" -> "+catalog.Runner+"/"+model)
		}
	}
	return nil
}
