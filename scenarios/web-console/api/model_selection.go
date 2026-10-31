package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"web-console/handlers/modelselection"
)

const codingRoleName = "code.default"

var safeNativeModelArg = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/+-]{0,255}$`)

// GetRoleModelChoices returns only resource-declared choices whose source can
// be expressed through the runner's native launch model argument. It does not
// inspect credentials or claim that a configured provider is reachable.
func (s *Server) GetRoleModelChoices(_ context.Context, agentID, targetID string) modelselection.Result {
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
	resource, ok := agentResource(agentID)
	if !ok {
		result.PolicyStatus = "unsupported"
		result.Detail = "this shortcut is not linked to a supported coding resource"
		return result
	}
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
	result.ModelFlag = ""
	result.Provenance = catalog.Provenance.Source
	result.ObservedAt = catalog.Provenance.ObservedAt
	digest := sha256.Sum256(raw)
	result.PolicyDigest = hex.EncodeToString(digest[:])
	choices, err := agentcatalog.RoleModelChoices(catalog, codingRoleName)
	if err != nil {
		result.Detail = "resource role choices could not be resolved"
		return result
	}
	allowedSources, flag, prefixSource, supported := nativeModelTranslation(catalog.Runner)
	if !supported {
		result.PolicyStatus = "unsupported"
		result.Detail = "this runner has no verified interactive model argument mapping"
		return result
	}
	for _, choice := range choices {
		if _, ok := allowedSources[choice.Source]; !ok {
			continue
		}
		launchModel := choice.Model
		if prefixSource && choice.Source != "" && !strings.HasPrefix(launchModel, choice.Source+"/") {
			launchModel = choice.Source + "/" + launchModel
		}
		if !safeNativeModelArg.MatchString(launchModel) {
			continue
		}
		item := modelselection.Choice{Model: choice.Model, Source: choice.Source, Provider: choice.Provider, LaunchModel: launchModel, SourceLabel: modelSourceLabel(catalog.Billing, choice.Source), IsDefault: choice.Model == catalog.Roles[codingRoleName].Model}
		if choice.Effort != nil {
			item.Effort = *choice.Effort
		}
		result.Choices = append(result.Choices, item)
	}
	if len(result.Choices) == 0 {
		result.PolicyStatus = "unsupported"
		result.Detail = "the resource declares no choices with a verified interactive launch mapping"
		return result
	}
	// Vrooli-funded coding is an application-owned route. Keep it out of the
	// resource policy and advertise it only after the paid execution bridge has
	// been explicitly qualified by the server owner.
	if catalog.Runner == "opencode" && targetID == "local" && s.paidCodingProtocolQualified {
		result.Choices = append(result.Choices, modelselection.Choice{
			Model: "coding", Source: "vrooli-credits", Provider: "vrooli",
			LaunchModel: "vrooli/coding", SourceLabel: "Vrooli credits",
		})
	}
	result.PolicyStatus = "ready"
	result.ModelFlag = flag
	if len(agentcatalog.CatalogStalenessFindings(catalog, time.Now(), false)) > 0 {
		result.PolicyStatus = "stale"
		result.Detail = "resource policy is stale; refresh it before choosing an override"
	}
	if flag == "" { // Defensive invariant: every ready choice has an argv mapping.
		result.PolicyStatus = "unsupported"
		result.Choices = nil
		result.Detail = "this runner has no verified interactive model argument mapping"
	}
	return result
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

func nativeModelTranslation(runner string) (map[string]struct{}, string, bool, bool) {
	switch runner {
	case "codex":
		return map[string]struct{}{"codex-subscription": {}}, "-m", false, true
	case "claude-code":
		return map[string]struct{}{"claude-subscription": {}}, "--model", false, true
	case "opencode":
		return map[string]struct{}{"opencode-go": {}, "openrouter": {}, "ollama": {}, "orcarouter": {}, "abliteration": {}}, "-m", true, true
	default:
		return nil, "", false, false
	}
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
