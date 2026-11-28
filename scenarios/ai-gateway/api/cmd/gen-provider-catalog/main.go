// gen-provider-catalog is the owner of the deployed, credential-free catalog.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

const (
	roleCatalogRelPath = "scenarios/ai-gateway/config/inference-role-catalog.json"
	serviceRelPath     = "scenarios/ai-gateway/.vrooli/service.json"
)

// candidateProviderExceptions names candidates that are deliberately not
// resource-backed. The reason is required so a new provider is never skipped
// silently.
var candidateProviderExceptions = map[string]string{
	providers.ProviderMetered: "the metered door is owned by the subscription backend and has no resource model policy",
}

// dependencyResourceExceptions names service.json dependency resources that
// are intentionally not provider resources.
var dependencyResourceExceptions = map[string]string{
	providers.ProviderMetered: "the metered door is not a local resource dependency",
}

func main() {
	root := flag.String("root", "../../..", "repository root from api directory")
	output := flag.String("output", "internal/catalog/providers.generated.json", "output path")
	check := flag.Bool("check", false, "verify output without writing")
	flag.Parse()
	if err := run(*root, *output, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, output string, check bool) error {
	inputs, err := loadInputs(root)
	if err != nil {
		return err
	}
	entries, err := providerEntries(root, inputs)
	if err != nil {
		return err
	}
	if err := addSupplierEntries(root, entries); err != nil {
		return err
	}
	harnesses, err := loadHarnesses(root)
	if err != nil {
		return err
	}
	data, err := renderCatalog(inputs, entries, harnesses)
	if err != nil {
		return err
	}
	return writeCatalog(output, data, check)
}

type candidateRef struct {
	Candidate rolecatalog.Candidate
	Stage     string
}

func roleCandidates(role rolecatalog.InferenceRole) []candidateRef {
	candidates := make([]candidateRef, 0, len(role.Candidates)+len(role.SemanticEscalationCandidates)+len(role.QualificationCandidates))
	for _, candidate := range role.Candidates {
		candidates = append(candidates, candidateRef{Candidate: candidate, Stage: "candidate"})
	}
	for _, candidate := range role.SemanticEscalationCandidates {
		candidates = append(candidates, candidateRef{Candidate: candidate, Stage: "semantic escalation"})
	}
	for _, qualification := range role.QualificationCandidates {
		candidates = append(candidates, candidateRef{Candidate: qualification.Candidate, Stage: "qualification"})
	}
	return candidates
}

func parseResourceRoles(data []byte) (map[string]struct{}, error) {
	var policy struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("parse model policy: %w", err)
	}
	if len(policy.Roles) == 0 {
		return nil, fmt.Errorf("model policy declares no roles")
	}
	roles := make(map[string]struct{}, len(policy.Roles))
	for role := range policy.Roles {
		roles[role] = struct{}{}
	}
	return roles, nil
}

// validateRoleCandidates checks every candidate a role catalog can dispatch,
// including non-serving qualification and escalation candidates, against the
// provider registry and the resource policy that owns the named role.
func validateRoleCandidates(catalog rolecatalog.RoleCatalog, resourceRoles map[string]map[string]struct{}) error {
	var problems []string
	for _, roleName := range sortedCatalogRoles(catalog) {
		role := catalog.Roles[roleName]
		for _, ref := range roleCandidates(role) {
			problems = append(problems, validateRoleCandidate(roleName, ref, catalog, resourceRoles)...)
		}
	}
	return validationProblems("provider catalog validation failed", problems)
}

func validateRoleCandidate(roleName string, ref candidateRef, catalog rolecatalog.RoleCatalog, resourceRoles map[string]map[string]struct{}) []string {
	provider := strings.ToLower(strings.TrimSpace(ref.Candidate.Provider))
	if provider == rolecatalog.ProviderGateway {
		return validateGatewayRoleCandidate(roleName, ref, catalog)
	}
	return validateProviderRoleCandidate(roleName, ref, provider, resourceRoles)
}

func validateGatewayRoleCandidate(roleName string, ref candidateRef, catalog rolecatalog.RoleCatalog) []string {
	target, ok := catalog.Roles[ref.Candidate.ResourceRole]
	if !ok {
		return []string{fmt.Sprintf("role %q %s candidate names undeclared gateway role %q", roleName, ref.Stage, ref.Candidate.ResourceRole)}
	}
	if target.IsDecision() {
		return []string{fmt.Sprintf("role %q %s candidate names decision gateway role %q; self candidates must be typed roles", roleName, ref.Stage, ref.Candidate.ResourceRole)}
	}
	return nil
}

func validateProviderRoleCandidate(roleName string, ref candidateRef, provider string, resourceRoles map[string]map[string]struct{}) []string {
	if _, ok := candidateProviderExceptions[provider]; ok {
		return nil
	}
	spec, ok := providers.SpecFor(provider)
	if !ok {
		return []string{fmt.Sprintf("role %q %s candidate provider %q is not a registered provider", roleName, ref.Stage, ref.Candidate.Provider)}
	}
	roles, ok := resourceRoles[spec.ID]
	if !ok {
		return []string{fmt.Sprintf("role %q %s candidate provider %q has no loaded model policy", roleName, ref.Stage, spec.ID)}
	}
	if _, ok := roles[ref.Candidate.ResourceRole]; ok {
		return nil
	}
	return []string{fmt.Sprintf("role %q %s candidate resource_role %q is missing from resources/%s/model-policy.json (available: %s)",
		roleName, ref.Stage, ref.Candidate.ResourceRole, spec.ID, strings.Join(sortedKeys(roles), ", "))}
}

func validationProblems(label string, problems []string) error {
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("%s:\n  - %s", label, strings.Join(problems, "\n  - "))
	}
	return nil
}

// validateServiceModelRoles checks that .vrooli/service.json only depends on
// registered providers and that each model_roles entry is declared by the
// resource's own model policy.
func validateServiceModelRoles(root string, resourceRoles map[string]map[string]struct{}) error {
	path := filepath.Join(root, serviceRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", serviceRelPath, err)
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
		return fmt.Errorf("parse %s: %w", serviceRelPath, err)
	}
	if len(service.Dependencies.Resources) == 0 {
		return fmt.Errorf("%s declares no dependency resources", serviceRelPath)
	}

	var problems []string
	for _, provider := range sortedKeys(service.Dependencies.Resources) {
		problems = append(problems, validateServiceDependencyRoles(provider, service.Dependencies.Resources[provider], resourceRoles)...)
	}
	return validationProblems("service dependency validation failed", problems)
}

func validateServiceDependencyRoles(provider string, dependency struct {
	Type       string            `json:"type"`
	ModelRoles []json.RawMessage `json:"model_roles"`
}, resourceRoles map[string]map[string]struct{},
) []string {
	spec, ok := providers.SpecFor(provider)
	if !ok {
		if _, excepted := dependencyResourceExceptions[provider]; excepted {
			return nil
		}
		return []string{fmt.Sprintf("dependency resource %q is not a registered provider", provider)}
	}
	roles, ok := resourceRoles[spec.ID]
	if !ok {
		return []string{fmt.Sprintf("dependency resource %q has no loaded model policy", provider)}
	}
	var problems []string
	for _, raw := range dependency.ModelRoles {
		problems = append(problems, validateServiceModelRole(provider, spec.ID, raw, roles)...)
	}
	return problems
}

func validateServiceModelRole(provider, providerID string, raw json.RawMessage, roles map[string]struct{}) []string {
	roleName, err := modelRoleName(raw)
	if err != nil {
		return []string{fmt.Sprintf("dependency %q model_roles: %v", provider, err)}
	}
	if _, ok := roles[roleName]; ok {
		return nil
	}
	return []string{fmt.Sprintf("dependency %q model_roles entry %q is missing from resources/%s/model-policy.json (available: %s)",
		provider, roleName, providerID, strings.Join(sortedKeys(roles), ", "))}
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

func sortedCatalogRoles(catalog rolecatalog.RoleCatalog) []string {
	roles := make([]string, 0, len(catalog.Roles))
	for role := range catalog.Roles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
