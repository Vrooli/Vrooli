package catalog_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ai-gateway/internal/catalog"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

// candidateProviderExceptions names candidates that are deliberately not
// resource-backed. The reason is required: a new provider must not be skipped
// silently.
var candidateProviderExceptions = map[string]string{
	"ai-gateway": "AI Gateway's own typed roles; resource_role names another role in this catalog, validated against the role catalog",
	"lpbs":       "the metered door is owned by the subscription backend and has no resources/lpbs/model-policy.json",
}

// dependencyResourceExceptions names .vrooli/service.json dependency resources
// that are intentionally not provider resources.
var dependencyResourceExceptions = map[string]string{
	"lpbs": "the metered door is not a local resource dependency",
}

// TestRoleCandidatesResolveToUsableProviders proves every role candidate names
// a provider the gateway can actually execute. AI Gateway's self candidates
// must name a declared non-decision role; the metered door is explicit; every
// other candidate must resolve through the provider registry.
func TestRoleCandidatesResolveToUsableProviders(t *testing.T) {
	root := repositoryRoot(t)
	roleCatalog, err := rolecatalog.LoadCatalog(filepath.Join(root, "scenarios", "ai-gateway", "config", "inference-role-catalog.json"))
	if err != nil {
		t.Fatalf("load role catalog: %v", err)
	}

	for _, roleName := range sortedRoleNames(roleCatalog) {
		role := roleCatalog.Roles[roleName]
		for _, candidate := range roleCandidates(role) {
			candidate := candidate
			t.Run(fmt.Sprintf("%s/%s/%s", roleName, candidate.Provider, candidate.ResourceRole), func(t *testing.T) {
				provider := strings.ToLower(strings.TrimSpace(candidate.Provider))
				if provider == rolecatalog.ProviderGateway {
					target, ok := roleCatalog.Roles[candidate.ResourceRole]
					if !ok {
						t.Fatalf("role %q candidate %q names undeclared typed role %q", roleName, provider, candidate.ResourceRole)
					}
					if target.IsDecision() {
						t.Fatalf("role %q candidate %q names decision role %q; self candidates must be typed roles", roleName, provider, candidate.ResourceRole)
					}
					return
				}
				if reason, ok := candidateProviderExceptions[provider]; ok {
					t.Logf("explicit candidate-provider exception %s: %s", provider, reason)
					return
				}
				if _, ok := providers.SpecFor(provider); !ok {
					t.Fatalf("role %q candidate provider %q is not a registered provider", roleName, candidate.Provider)
				}
			})
		}
	}
}

// TestRoleCandidatesNameKnownResourceRoles proves every resource-backed
// candidate names a role the resource's model policy actually declares. A typo
// here would otherwise surface at runtime as a skipped candidate.
func TestRoleCandidatesNameKnownResourceRoles(t *testing.T) {
	root := repositoryRoot(t)
	roleCatalog, err := rolecatalog.LoadCatalog(filepath.Join(root, "scenarios", "ai-gateway", "config", "inference-role-catalog.json"))
	if err != nil {
		t.Fatalf("load role catalog: %v", err)
	}
	policyRoles := map[string]map[string]struct{}{}
	rolesFor := func(provider string) map[string]struct{} {
		roles, ok := policyRoles[provider]
		if ok {
			return roles
		}
		roles = loadResourceRoles(t, root, provider)
		policyRoles[provider] = roles
		return roles
	}

	for _, roleName := range sortedRoleNames(roleCatalog) {
		role := roleCatalog.Roles[roleName]
		for _, candidate := range roleCandidates(role) {
			candidate := candidate
			t.Run(fmt.Sprintf("%s/%s/%s", roleName, candidate.Provider, candidate.ResourceRole), func(t *testing.T) {
				provider := strings.ToLower(strings.TrimSpace(candidate.Provider))
				if reason, ok := candidateProviderExceptions[provider]; ok {
					t.Logf("explicit candidate-provider exception %s: %s", provider, reason)
					return
				}
				spec, ok := providers.SpecFor(provider)
				if !ok {
					t.Fatalf("role %q candidate provider %q is not a registered provider", roleName, candidate.Provider)
				}
				roles := rolesFor(spec.ID)
				if _, ok := roles[candidate.ResourceRole]; !ok {
					t.Fatalf("role %q candidate resource_role %q is missing from resources/%s/model-policy.json (available: %s)",
						roleName, candidate.ResourceRole, spec.ID, strings.Join(sortedKeys(roles), ", "))
				}
			})
		}
	}
}

// TestServiceDependencyProvidersAndModelRoles proves .vrooli/service.json only
// depends on registered providers and that every declared model_roles entry is
// a role the resource policy declares.
func TestServiceDependencyProvidersAndModelRoles(t *testing.T) {
	root := repositoryRoot(t)
	servicePath := filepath.Join(root, "scenarios", "ai-gateway", ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read %s: %v", servicePath, err)
	}
	var service struct {
		Dependencies struct {
			Resources map[string]struct {
				Type       string            `json:"type"`
				ModelRoles []json.RawMessage `json:"model_roles"`
			} `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &service); err != nil {
		t.Fatalf("parse %s: %v", servicePath, err)
	}
	if len(service.Dependencies.Resources) == 0 {
		t.Fatal("service.json declares no dependency resources")
	}

	roleCache := map[string]map[string]struct{}{}
	for _, provider := range sortedKeys(service.Dependencies.Resources) {
		dependency := service.Dependencies.Resources[provider]
		provider := provider
		t.Run(provider, func(t *testing.T) {
			spec, ok := providers.SpecFor(provider)
			if !ok {
				if reason, excepted := dependencyResourceExceptions[provider]; excepted {
					t.Logf("explicit dependency-resource exception %s: %s", provider, reason)
					return
				}
				t.Fatalf("service.json dependency resource %q is not a registered provider", provider)
			}
			roles, ok := roleCache[spec.ID]
			if !ok {
				roles = loadResourceRoles(t, root, spec.ID)
				roleCache[spec.ID] = roles
			}
			for _, raw := range dependency.ModelRoles {
				roleName, err := modelRoleName(raw)
				if err != nil {
					t.Fatalf("service.json dependency %q model_roles: %v", provider, err)
				}
				if _, ok := roles[roleName]; ok {
					continue
				}
				t.Fatalf("service.json dependency %q model_roles entry %q is missing from resources/%s/model-policy.json (available: %s)",
					provider, roleName, spec.ID, strings.Join(sortedKeys(roles), ", "))
			}
		})
	}
}

// TestGeneratedProviderCatalogMatchesRegistry proves the checked-in generated
// catalog agrees with the provider registry for the entries the generator
// owns. Providers intentionally excluded by the generator are not required.
func TestGeneratedProviderCatalogMatchesRegistry(t *testing.T) {
	root := repositoryRoot(t)
	generatedPath := filepath.Join(root, "scenarios", "ai-gateway", "api", "internal", "catalog", "providers.generated.json")
	data, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read %s: %v", generatedPath, err)
	}
	var bundle catalog.Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("parse %s: %v", generatedPath, err)
	}

	projectedFamilies := map[string]struct{}{
		providers.FamilyOllama:           {},
		providers.FamilyOpenAICompatible: {},
	}
	specs := providers.DefaultProviderSpecs()
	for _, spec := range specs {
		entry, present := bundle.Providers[spec.ID]
		_, projected := projectedFamilies[spec.Family]
		if projected && !present {
			t.Errorf("generated catalog omits resource-backed provider %q (family %q)", spec.ID, spec.Family)
			continue
		}
		if !present {
			continue
		}
		if entry.ID != spec.ID {
			t.Errorf("generated provider %q has id %q", spec.ID, entry.ID)
		}
		if entry.Family != spec.Family {
			t.Errorf("generated provider %q family = %q, want %q", spec.ID, entry.Family, spec.Family)
		}
		if entry.Locality != spec.Locality {
			t.Errorf("generated provider %q locality = %q, want %q", spec.ID, entry.Locality, spec.Locality)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if pathExists(filepath.Join(dir, ".git")) || pathExists(filepath.Join(dir, "VISION.md")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root from catalog package")
		}
		dir = parent
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadResourceRoles(t *testing.T, root, provider string) map[string]struct{} {
	t.Helper()
	path := filepath.Join(root, "resources", provider, "model-policy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read resource policy for %q: %v", provider, err)
	}
	var policy struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if len(policy.Roles) == 0 {
		t.Fatalf("%s declares no roles", path)
	}
	roles := make(map[string]struct{}, len(policy.Roles))
	for role := range policy.Roles {
		roles[role] = struct{}{}
	}
	return roles
}

func modelRoleName(raw json.RawMessage) (string, error) {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		name = strings.TrimSpace(name)
		if name == "" {
			return "", fmt.Errorf("empty role name")
		}
		return name, nil
	}
	var object struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return "", fmt.Errorf("decode role reference %s: %w", string(raw), err)
	}
	name = strings.TrimSpace(object.Role)
	if name == "" {
		return "", fmt.Errorf("role reference %s has no role", string(raw))
	}
	return name, nil
}

func roleCandidates(role rolecatalog.InferenceRole) []rolecatalog.Candidate {
	candidates := append([]rolecatalog.Candidate{}, role.Candidates...)
	candidates = append(candidates, role.SemanticEscalationCandidates...)
	for _, qualification := range role.QualificationCandidates {
		candidates = append(candidates, qualification.Candidate)
	}
	return candidates
}

func sortedRoleNames(roleCatalog rolecatalog.RoleCatalog) []string {
	names := make([]string, 0, len(roleCatalog.Roles))
	for name := range roleCatalog.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// decide.candidates is the bake-off role: a model pinned for evaluation must be
// reachable through declared data, so every local Ollaya role the resource
// serves is a candidate of it. Production tiers are reordered only from
// measured results, so a role added for evaluation must not leak into them.
func TestBakeoffRoleDeclaresEveryLocalOllayaRoleAndProductionTiersAreSeparate(t *testing.T) {
	root := repositoryRoot(t)
	roleCatalog, err := rolecatalog.LoadCatalog(filepath.Join(root, "scenarios", "ai-gateway", "config", "inference-role-catalog.json"))
	if err != nil {
		t.Fatalf("load role catalog: %v", err)
	}
	bakeoff, ok := roleCatalog.Roles["decide.candidates"]
	if !ok || !bakeoff.IsDecision() {
		t.Fatalf("decide.candidates must be a declared decision role, got %+v", bakeoff)
	}
	declared := map[string]struct{}{}
	for _, candidate := range bakeoff.Candidates {
		declared[candidate.ResourceRole] = struct{}{}
	}

	data, err := os.ReadFile(filepath.Join(root, "resources", "ollaya", "model-policy.json"))
	if err != nil {
		t.Fatalf("read ollaya policy: %v", err)
	}
	var policy struct {
		Upstreams map[string]struct {
			Locality string `json:"locality"`
		} `json:"upstreams"`
		Roles map[string]struct {
			Upstream string `json:"upstream"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatalf("parse ollaya policy: %v", err)
	}
	localRoles := map[string]struct{}{}
	for name, role := range policy.Roles {
		if policy.Upstreams[role.Upstream].Locality != "local" {
			continue
		}
		localRoles[name] = struct{}{}
		if _, ok := declared[name]; !ok {
			t.Errorf("local ollaya role %q is not a candidate of decide.candidates, so it cannot be pinned for a bake-off", name)
		}
	}

	production := map[string]struct{}{}
	for _, tier := range []string{"decide.fast", "decide.accurate", "browser.decide"} {
		for _, candidate := range roleCatalog.Roles[tier].Candidates {
			production[candidate.ResourceRole] = struct{}{}
		}
	}
	for name := range localRoles {
		if _, inProduction := production[name]; inProduction && name != "decide.local" && name != "decide.local.accurate" {
			t.Errorf("bake-off-only role %q is already in a production decision tier; promote it only from measured results", name)
		}
	}
}

// decide.short serves callers whose states fit a 512-token model, and
// decide.short.accurate is its escalation. Escalating to the model that already
// answered would repeat the call, so the two tiers must start on different
// roles, and both must stay on local Ollaya roles the bake-off role can pin.
func TestShortStateTiersEscalateToADifferentModelAndStayLocal(t *testing.T) {
	root := repositoryRoot(t)
	roleCatalog, err := rolecatalog.LoadCatalog(filepath.Join(root, "scenarios", "ai-gateway", "config", "inference-role-catalog.json"))
	if err != nil {
		t.Fatalf("load role catalog: %v", err)
	}
	short, accurate := roleCatalog.Roles["decide.short"], roleCatalog.Roles["decide.short.accurate"]
	if !short.IsDecision() || !accurate.IsDecision() {
		t.Fatalf("decide.short and decide.short.accurate must be decision roles, got %+v and %+v", short, accurate)
	}
	if short.Candidates[0].ResourceRole == accurate.Candidates[0].ResourceRole {
		t.Errorf("both short tiers start on %q, so an escalation would re-ask the same model", short.Candidates[0].ResourceRole)
	}
	bakeoff := map[string]struct{}{}
	for _, candidate := range roleCatalog.Roles["decide.candidates"].Candidates {
		bakeoff[candidate.ResourceRole] = struct{}{}
	}
	for name, role := range map[string]rolecatalog.InferenceRole{"decide.short": short, "decide.short.accurate": accurate} {
		for _, candidate := range role.Candidates {
			if candidate.Provider != "ollaya" {
				t.Errorf("%s candidate %q is not a local Ollaya role", name, candidate.ResourceRole)
			}
			if _, ok := bakeoff[candidate.ResourceRole]; !ok {
				t.Errorf("%s candidate %q is not declared in decide.candidates", name, candidate.ResourceRole)
			}
		}
	}
}
