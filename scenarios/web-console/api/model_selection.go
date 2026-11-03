package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/cliutil"
	"web-console/handlers/modelselection"
)

const codingRoleName = "code.default"

// contextLimitCandidates are the compaction limits a launcher offers, before the
// model's window and the harness's minimum narrow them.
var contextLimitCandidates = []int64{1000000, 300000, 150000}

var safeNativeModelArg = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/+-]{0,255}$`)

// GetRoleModelChoices returns only resource-declared choices whose source can
// be expressed through the runner's native launch model argument. It does not
// inspect credentials or claim that a configured provider is reachable.
func (s *Server) GetRoleModelChoices(_ context.Context, agentID, targetID, launchCommand string) modelselection.Result {
	result := modelselection.Result{PolicyStatus: "unknown", Availability: "unknown", Role: codingRoleName}
	agentID, targetID = strings.TrimSpace(agentID), strings.TrimSpace(targetID)
	if targetID == "" {
		targetID = "local"
	}
	if targetID != "local" {
		result.PolicyStatus = "unsupported"
		result.Detail = "model policy for this remote target has not been observed"
		return result
	}
	descriptor, ok := harness.Active().ByID(agentID)
	if !ok {
		result.PolicyStatus = "unsupported"
		result.Detail = "this shortcut is not linked to a supported coding resource"
		return result
	}
	result.CommandLiteral = launchCommand != "" && !composableLaunchCommand(descriptor, launchCommand)
	resource := descriptor.Resource
	path, ok := modelPolicyPath(resource)
	if !ok {
		result.Detail = "resource policy could not be found"
		return result
	}
	catalog, raw, err := agentcatalog.LoadCodingRoleCatalog(resource, path)
	if err != nil {
		result.Detail = "resource policy could not be read or validated"
		return result
	}
	result.Runner = catalog.Runner
	result.SourceConfigurationSupported = cliutil.SupportsModelSource(catalog.Runner)
	result.ModelFlag = ""
	result.Provenance = catalog.Provenance.Source
	result.ObservedAt = catalog.Provenance.ObservedAt
	digest := sha256.Sum256(raw)
	result.PolicyDigest = hex.EncodeToString(digest[:])
	choices, err := agentcatalog.RunnerModelChoices(catalog, codingRoleName)
	if err != nil {
		result.Detail = "resource role choices could not be resolved"
		return result
	}
	selection := descriptor.Features.ModelSelection
	if selection == nil {
		result.PolicyStatus = "unsupported"
		result.Detail = "this runner has no verified interactive model argument mapping"
		return result
	}
	for _, choice := range choices {
		if !slices.Contains(selection.Sources, choice.Source) {
			continue
		}
		launchModel := choice.Model
		if selection.PrefixSource && choice.Source != "" && !strings.HasPrefix(launchModel, choice.Source+"/") {
			launchModel = choice.Source + "/" + launchModel
		}
		if !safeNativeModelArg.MatchString(launchModel) {
			continue
		}
		window := agentcatalog.ModelContextWindow(descriptor, catalog, choice.Model)
		item := modelselection.Choice{
			Model: choice.Model, DisplayName: modelDisplayNameForSource(catalog, choice.Source, choice.Model), Source: choice.Source, Provider: choice.Provider, LaunchModel: launchModel,
			SourceLabel: modelSourceLabel(catalog.Billing, choice.Source), IsDefault: choice.Model == catalog.Roles[codingRoleName].Model,
			ContextWindowTokens: window, ContextLimitTokens: descriptor.ContextLimitOptions(window, contextLimitCandidates),
			Metered: modelSourceMetered(catalog.Billing, choice.Source),
		}
		if choice.Effort != nil {
			item.Effort = *choice.Effort
		}
		result.Choices = append(result.Choices, item)
	}
	sortModelChoices(catalog, result.Choices)
	if len(result.Choices) == 0 {
		result.PolicyStatus = "unsupported"
		result.Detail = "the resource declares no choices with a verified interactive launch mapping"
		return result
	}
	result.PolicyStatus = "ready"
	result.ModelFlag = selection.Flag
	// A stale list warns (LD-2): the status stays informative for every consumer,
	// and the choices stay offered. Only metered choices wait for a refresh, since
	// a price or route the list no longer describes costs money.
	if len(agentcatalog.CatalogStalenessFindings(catalog, time.Now(), false)) > 0 {
		result.PolicyStatus = "stale"
		result.CatalogStale = true
		result.Detail = "this model list may be out of date; metered choices need a refresh"
		if !agentcatalog.CanRefreshCatalog(catalog.Runner) {
			result.OperatorCommand = "$EDITOR resources/" + resource + "/model-policy.json"
		}
	}
	return result
}

// sortModelChoices keeps each source together, the default's first. Within a source
// the default leads, then higher tiers, then newer versions.
func sortModelChoices(catalog agentcatalog.CodingRoleCatalog, choices []modelselection.Choice) {
	rank := map[string]int{}
	for _, choice := range choices {
		if _, seen := rank[choice.Source]; !seen || choice.IsDefault {
			rank[choice.Source] = len(rank) + 1
			if choice.IsDefault {
				rank[choice.Source] = 0
			}
		}
	}
	tiers := modelFamilyTiers(catalog)
	slices.SortStableFunc(choices, func(a, b modelselection.Choice) int {
		if a.Source != b.Source {
			return rank[a.Source] - rank[b.Source]
		}
		if a.IsDefault != b.IsDefault {
			if a.IsDefault {
				return -1
			}
			return 1
		}
		familyA, versionA := splitModelName(a.DisplayName)
		familyB, versionB := splitModelName(b.DisplayName)
		if familyA != familyB {
			if delta := tiers[familyB] - tiers[familyA]; delta != 0 {
				return delta
			}
			return strings.Compare(familyA, familyB)
		}
		if delta := compareVersions(versionB, versionA); delta != 0 {
			return delta
		}
		return strings.Compare(a.DisplayName, b.DisplayName)
	})
}

// modelFamilyTiers ranks families by how far usage tiers lower them: fable lowers to
// opus lowers to sonnet, so fable ranks highest. A family outside the map, such as a
// cheap model nothing lowers to, ranks below them. The catalog is the only source.
func modelFamilyTiers(catalog agentcatalog.CodingRoleCatalog) map[string]int {
	tiers := map[string]int{}
	if catalog.UsageTiers == nil {
		return tiers
	}
	steps := func(model string) int {
		count, seen := 1, map[string]bool{model: true}
		for next, ok := catalog.UsageTiers.ModelMap[model]; ok && !seen[next]; next, ok = catalog.UsageTiers.ModelMap[next] {
			seen[next] = true
			count++
		}
		return count
	}
	note := func(model string, rank int) {
		family, _ := splitModelName(modelDisplayName(catalog, model))
		if rank > tiers[family] {
			tiers[family] = rank
		}
	}
	for model := range catalog.UsageTiers.ModelMap {
		note(model, steps(model))
	}
	if session := catalog.UsageTiers.SessionModel; session != "" {
		note(session, steps(session))
	}
	return tiers
}

// splitModelName separates "Opus 5.5" into its family and version. A name with no
// version is all family.
func splitModelName(name string) (family string, version []int) {
	fields := strings.Fields(name)
	for index, field := range fields {
		digits := strings.TrimPrefix(strings.ToLower(field), "v")
		if digits == "" || digits[0] < '0' || digits[0] > '9' {
			continue
		}
		for _, part := range strings.Split(digits, ".") {
			number, err := strconv.Atoi(part)
			if err != nil {
				return strings.Join(fields, " "), nil
			}
			version = append(version, number)
		}
		return strings.Join(fields[:index], " "), version
	}
	return strings.Join(fields, " "), nil
}

func compareVersions(a, b []int) int {
	return slices.Compare(a, b)
}

func canonicalModelName(catalog agentcatalog.CodingRoleCatalog, model string) string {
	if alias, ok := catalog.ModelAliases[model]; ok && alias.CanonicalModel != "" {
		return alias.CanonicalModel
	}
	return model
}

var (
	modelDateSuffix = regexp.MustCompile(`-\d{8}$`)
	modelNameWords  = map[string]string{"gpt": "GPT", "glm": "GLM", "deepseek": "DeepSeek", "mimo": "MiMo"}
)

// modelDisplayName turns a catalog ID into a readable name. Policies declare no
// display names, and "opus" alone hides the version, so it reads the canonical
// model: anthropic/claude-opus-5-5 becomes "Opus 5.5".
func modelDisplayNameForSource(catalog agentcatalog.CodingRoleCatalog, source, model string) string {
	if source != "" && !strings.HasPrefix(model, source+"/") {
		if alias, ok := catalog.ModelAliases[source+"/"+model]; ok && strings.TrimSpace(alias.DisplayName) != "" {
			return strings.TrimSpace(alias.DisplayName)
		}
	}
	return modelDisplayName(catalog, model)
}

func modelDisplayName(catalog agentcatalog.CodingRoleCatalog, model string) string {
	if alias, ok := catalog.ModelAliases[model]; ok && strings.TrimSpace(alias.DisplayName) != "" {
		return strings.TrimSpace(alias.DisplayName)
	}
	id := canonicalModelName(catalog, model)
	id = id[strings.LastIndex(id, "/")+1:]
	id = modelDateSuffix.ReplaceAllString(strings.TrimPrefix(id, "claude-"), "")
	var words []string
	for _, part := range strings.FieldsFunc(id, func(r rune) bool { return r == '-' || r == ':' }) {
		number := part[0] >= '0' && part[0] <= '9'
		last := len(words) - 1
		switch {
		case number && last >= 0 && words[last][0] >= '0' && words[last][0] <= '9':
			words[last] += "." + part
		case number:
			words = append(words, part)
		case modelNameWords[part] != "":
			words = append(words, modelNameWords[part])
		default:
			words = append(words, strings.ToUpper(part[:1])+part[1:])
		}
	}
	if len(words) == 0 {
		return model
	}
	return strings.Join(words, " ")
}

func modelSourceLabel(billing *agentcatalog.Billing, source string) string {
	if billing == nil {
		return source
	}
	for _, candidate := range billing.Sources {
		if candidate.ID == source && strings.TrimSpace(candidate.Label) != "" {
			return candidate.Label
		}
	}
	return source
}

func modelSourceMetered(billing *agentcatalog.Billing, source string) bool {
	if billing == nil {
		return false
	}
	for _, candidate := range billing.Sources {
		if candidate.ID == source {
			return candidate.BillingMode == "metered"
		}
	}
	return billing.Mode == "metered"
}

func modelPolicyPath(resource string) (string, bool) {
	var roots []string
	for _, key := range []string{"VROOLI_RESOURCE_ROOT", "VROOLI_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(key)); root != "" {
			roots = append(roots, root)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir, i := cwd, 0; i < 8; dir, i = filepath.Dir(dir), i+1 {
			roots = append(roots, dir)
		}
	}
	for _, root := range roots {
		path := filepath.Join(root, "resources", resource, "model-policy.json")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}
