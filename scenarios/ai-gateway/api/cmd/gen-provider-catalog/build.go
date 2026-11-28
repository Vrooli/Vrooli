package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

type catalogInputs struct {
	roles         json.RawMessage
	features      json.RawMessage
	creditOwners  json.RawMessage
	specs         []providers.ProviderSpec
	resourceRoles map[string]map[string]struct{}
	policies      map[string]json.RawMessage
}

func loadInputs(root string) (catalogInputs, error) {
	read := func(path string) ([]byte, error) { return os.ReadFile(filepath.Join(root, path)) }
	roles, err := read(roleCatalogRelPath)
	if err != nil {
		return catalogInputs{}, err
	}
	catalog, err := rolecatalog.LoadCatalog(filepath.Join(root, roleCatalogRelPath))
	if err != nil {
		return catalogInputs{}, err
	}
	features, err := read("packages/monetization-go/ai-features.generated.json")
	if err != nil {
		return catalogInputs{}, err
	}
	credits, err := read("packages/monetization-go/credit-features.generated.json")
	if err != nil {
		return catalogInputs{}, err
	}
	inputs := catalogInputs{roles: roles, features: features, creditOwners: credits, specs: providers.DefaultProviderSpecs(), resourceRoles: map[string]map[string]struct{}{}, policies: map[string]json.RawMessage{}}
	for _, spec := range inputs.specs {
		policy, err := read("resources/" + spec.ID + "/model-policy.json")
		if err != nil {
			return catalogInputs{}, fmt.Errorf("read model policy for provider %q: %w", spec.ID, err)
		}
		resourceRoleSet, err := parseResourceRoles(policy)
		if err != nil {
			return catalogInputs{}, fmt.Errorf("resources/%s/model-policy.json: %w", spec.ID, err)
		}
		inputs.resourceRoles[spec.ID], inputs.policies[spec.ID] = resourceRoleSet, policy
	}
	if err := validateRoleCandidates(catalog, inputs.resourceRoles); err != nil {
		return catalogInputs{}, err
	}
	if err := validateServiceModelRoles(root, inputs.resourceRoles); err != nil {
		return catalogInputs{}, err
	}
	return inputs, nil
}

func providerEntries(root string, inputs catalogInputs) (map[string]any, error) {
	entries := map[string]any{}
	for _, spec := range inputs.specs {
		if spec.Family != providers.FamilyOllama && spec.Family != providers.FamilyOpenAICompatible {
			continue
		}
		resourceData, err := os.ReadFile(filepath.Join(root, "resources", spec.ID, "resource.json"))
		if err != nil {
			return nil, err
		}
		entry, err := resourceEntry(spec, inputs.policies[spec.ID], resourceData)
		if err != nil {
			return nil, err
		}
		entries[spec.ID] = entry
	}
	return entries, nil
}

func resourceEntry(spec providers.ProviderSpec, policy, resourceData []byte) (map[string]any, error) {
	if spec.ID == providers.ProviderOpenRouter {
		if err := validateExactModelIdentityProvenance("resources/openrouter/model-policy.json", policy); err != nil {
			return nil, err
		}
		if err := validateOpenRouterModelTermsScope("resources/openrouter/model-policy.json", policy); err != nil {
			return nil, err
		}
	}
	var resource struct {
		Endpoint string `json:"endpoint"`
		Ports    []struct {
			Host int `json:"host"`
		} `json:"ports"`
		Credentials struct {
			Descriptors []struct {
				Identity string `json:"logical_id"`
				Field    string `json:"field"`
			} `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(resourceData, &resource); err != nil {
		return nil, err
	}
	endpoint := strings.TrimSuffix(resource.Endpoint, "/models")
	if spec.Family == providers.FamilyOllama && len(resource.Ports) > 0 {
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", resource.Ports[0].Host)
	}
	entry := map[string]any{"id": spec.ID, "family": spec.Family, "locality": spec.Locality, "endpoint": endpoint, "policy": json.RawMessage(policy)}
	if len(resource.Credentials.Descriptors) > 0 {
		entry["credential_identity"] = resource.Credentials.Descriptors[0].Identity
		entry["credential_field"] = resource.Credentials.Descriptors[0].Field
	}
	return entry, nil
}

type sourceTermsEvidence struct {
	URL          string `json:"url"`
	DocumentDate string `json:"document_date"`
	ReadAt       string `json:"read_at"`
}

type sourceTermsPlanRule struct {
	Plan          string              `json:"plan"`
	UseClasses    map[string]string   `json:"use_classes"`
	Conditions    []string            `json:"conditions"`
	Clause        string              `json:"clause"`
	Reason        string              `json:"reason"`
	SourceAnchors []sourceScopeAnchor `json:"source_anchors"`
}

type sourceScopeAnchor struct {
	SupplierID       string `json:"supplier_id"`
	ModelID          string `json:"model_id"`
	URL              string `json:"url"`
	Anchor           string `json:"anchor"`
	DocumentDate     string `json:"document_date"`
	ObservedAt       string `json:"observed_at"`
	RetrievalStatus  string `json:"retrieval_status"`
	Supports         string `json:"supports"`
	DoesNotEstablish string `json:"does_not_establish"`
	ClaimRef         string `json:"claim_ref"`
}

func validateTermsEvidence(path string, evidence sourceTermsEvidence) error {
	if !hasDirectHTTPSCatalogURL(evidence.URL) || !hasDocumentDateOrExplicitUnknown(evidence.DocumentDate) {
		return fmt.Errorf("invalid supplier declaration %s: terms evidence needs a direct HTTPS source and a document date or explicit undated statement", path)
	}
	if _, err := time.Parse("2006-01-02", evidence.ReadAt); err != nil {
		return fmt.Errorf("invalid supplier declaration %s: terms evidence read_at must use YYYY-MM-DD", path)
	}
	return nil
}

func validateTermsPlanRules(path string, rules []sourceTermsPlanRule) error {
	if len(rules) == 0 {
		return fmt.Errorf("invalid supplier declaration %s: terms must define at least one scoped plan rule", path)
	}
	for index, rule := range rules {
		if strings.TrimSpace(rule.Plan) == "" || strings.TrimSpace(rule.Clause) == "" || strings.TrimSpace(rule.Reason) == "" || len(rule.Conditions) == 0 || len(rule.UseClasses) == 0 {
			return fmt.Errorf("invalid supplier declaration %s: terms plan rule %d needs plan ID, clause, reason, use classes, and explicit conditions", path, index)
		}
		for useClass, permission := range rule.UseClasses {
			if strings.TrimSpace(useClass) == "" || (permission != "allowed" && permission != "denied" && permission != "forbidden" && permission != "unverified") {
				return fmt.Errorf("invalid supplier declaration %s: terms plan rule %d has invalid use-class decision %q=%q", path, index, useClass, permission)
			}
		}
	}
	return nil
}

func validateOpenRouterModelTermsScope(path string, policyJSON []byte) error {
	var policy struct {
		Provider struct {
			Terms struct {
				PlanRules []sourceTermsPlanRule `json:"plan_rules"`
				Evidence  sourceTermsEvidence   `json:"evidence"`
			} `json:"terms"`
			HostedRoutes []json.RawMessage `json:"hosted_routes"`
		} `json:"provider"`
		Models map[string]struct {
			Provenance struct {
				Terms struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"terms"`
			} `json:"provenance"`
			Caveats []string `json:"caveats"`
		} `json:"models"`
	}
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		return fmt.Errorf("invalid OpenRouter model policy %s: %w", path, err)
	}
	if err := validateTermsEvidence(path, policy.Provider.Terms.Evidence); err != nil {
		return err
	}
	if err := validateTermsPlanRules(path, policy.Provider.Terms.PlanRules); err != nil {
		return err
	}
	if err := validateScopedTermAnchors(path, policyJSON); err != nil {
		return err
	}
	if len(policy.Provider.HostedRoutes) != 0 {
		return fmt.Errorf("invalid OpenRouter model policy %s: terms plan scope must exist and hosted routes must remain empty", path)
	}
	if len(policy.Models) == 0 {
		return fmt.Errorf("invalid OpenRouter model policy %s: model-scope records are required", path)
	}
	for modelID, model := range policy.Models {
		if !hasDirectHTTPSCatalogURL(model.Provenance.Terms.Source) {
			return fmt.Errorf("invalid OpenRouter model policy %s: model %q needs a direct HTTPS model-terms scope source", path, modelID)
		}
		if _, err := time.Parse("2006-01-02", model.Provenance.Terms.ObservedAt); err != nil {
			return fmt.Errorf("invalid OpenRouter model policy %s: model %q terms observed_at must use YYYY-MM-DD", path, modelID)
		}
		if !hasExplicitScopeUnknown(strings.Join(model.Caveats, " ")) {
			return fmt.Errorf("invalid OpenRouter model policy %s: model %q must state an unresolved model-specific terms/right unknown", path, modelID)
		}
	}
	return nil
}

func addSupplierEntries(root string, entries map[string]any) error {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "ai-gateway", "config", "suppliers", "*.json"))
	if err != nil {
		return err
	}
	registered := make(map[string]struct{})
	for _, spec := range providers.DefaultProviderSpecs() {
		registered[spec.ID] = struct{}{}
	}
	supplierIDs := make(map[string]string, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := addSupplierEntry(root, path, data, entries, registered, supplierIDs); err != nil {
			return err
		}
	}
	return nil
}

type supplierDeclaration struct {
	ID       string          `json:"id"`
	Family   string          `json:"family"`
	Locality string          `json:"locality"`
	Endpoint string          `json:"endpoint"`
	Policy   json.RawMessage `json:"policy"`
}

func addSupplierEntry(root, path string, data []byte, entries map[string]any, registered map[string]struct{}, supplierIDs map[string]string) error {
	var entry supplierDeclaration
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("invalid supplier declaration %s: %w", path, err)
	}
	if err := validateSupplierDeclaration(path, entry); err != nil {
		return err
	}
	if _, exists := registered[entry.ID]; exists {
		return fmt.Errorf("supplier %s duplicates registered provider declaration", entry.ID)
	}
	if previous, exists := supplierIDs[entry.ID]; exists {
		return fmt.Errorf("supplier %s is declared by both %s and %s", entry.ID, previous, filepath.Base(path))
	}
	if _, exists := entries[entry.ID]; exists {
		return fmt.Errorf("supplier %s has two declaration owners", entry.ID)
	}
	if err := validateNoResourceDeclaration(root, entry.ID); err != nil {
		return err
	}
	supplierIDs[entry.ID] = filepath.Base(path)
	entries[entry.ID] = json.RawMessage(data)
	return nil
}

func validateSupplierDeclaration(path string, entry supplierDeclaration) error {
	if err := validateSupplierIdentity(path, entry); err != nil {
		return err
	}
	if err := validateSupplierPolicy(path, entry); err != nil {
		return err
	}
	return nil
}

func validateSupplierIdentity(path string, entry supplierDeclaration) error {
	if !isValidSupplierID(entry.ID) || !isValidSupplierMetadata(entry) {
		return fmt.Errorf("invalid supplier declaration %s", path)
	}
	return nil
}

func isValidSupplierID(id string) bool {
	return id != "" && strings.TrimSpace(id) == id && filepath.Base(id) == id && id != "." && id != ".." && !strings.ContainsAny(id, `/\\`)
}

func isValidSupplierMetadata(entry supplierDeclaration) bool {
	validFamily := entry.Family == providers.FamilyOpenAICompatible
	validLocality := entry.Locality == "local" || entry.Locality == "remote"
	validEndpoint := entry.Endpoint == "" || (strings.TrimSpace(entry.Endpoint) != "" && strings.TrimSpace(entry.Endpoint) == entry.Endpoint)
	return validFamily && validLocality && validEndpoint
}

func validateSupplierPolicy(path string, entry supplierDeclaration) error {
	if err := validateSupplierSourceAndModelScope(path, entry.Policy); err != nil {
		return err
	}
	var policyObject struct {
		Provider struct {
			ID    string `json:"id"`
			Terms struct {
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			HostedRoutes []json.RawMessage `json:"hosted_routes"`
			Agreements   []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Models map[string]struct {
			DefaultEligible *bool `json:"default_eligible"`
			Provenance      struct {
				Catalog struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"catalog"`
				Identity struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"identity"`
			} `json:"provenance"`
		} `json:"models"`
	}
	var raw map[string]json.RawMessage
	if len(entry.Policy) == 0 || !json.Valid(entry.Policy) || json.Unmarshal(entry.Policy, &raw) != nil || raw == nil || json.Unmarshal(entry.Policy, &policyObject) != nil {
		return fmt.Errorf("invalid supplier declaration %s", path)
	}
	reviewStatus := policyObject.Provider.Terms.Review.Status
	proposalOnly := reviewStatus == "proposed" || reviewStatus == "pending_review"
	for modelID, model := range policyObject.Models {
		if model.DefaultEligible == nil {
			return fmt.Errorf("invalid supplier declaration %s: model %q must set default_eligible explicitly", path, modelID)
		}
		if reviewStatus != "approved" && *model.DefaultEligible {
			return fmt.Errorf("invalid supplier declaration %s: proposal model %q must set default_eligible false", path, modelID)
		}
		if !hasDirectHTTPSCatalogURL(model.Provenance.Catalog.Source) {
			return fmt.Errorf("invalid supplier declaration %s: model %q needs a direct HTTPS catalog source", path, modelID)
		}
		if _, err := time.Parse("2006-01-02", model.Provenance.Catalog.ObservedAt); err != nil {
			return fmt.Errorf("invalid supplier declaration %s: model %q needs catalog provenance observed_at in YYYY-MM-DD form", path, modelID)
		}
		if err := validateExactModelIdentitySource(path, modelID, model.Provenance.Identity.Source, model.Provenance.Identity.ObservedAt); err != nil {
			return err
		}
	}
	hasExecutableAuthority := len(policyObject.Provider.HostedRoutes) != 0 || len(policyObject.Provider.Agreements) != 0
	if hasExecutableAuthority && reviewStatus != "approved" {
		return fmt.Errorf("invalid supplier declaration %s: hosted routes and agreements require approved review status", path)
	}
	if entry.Endpoint == "" {
		if policyObject.Provider.ID == "" || policyObject.Provider.ID != entry.ID {
			return fmt.Errorf("invalid supplier declaration %s: endpoint-less supplier policy identity must match the declaration", path)
		}
		if !proposalOnly {
			return fmt.Errorf("invalid supplier declaration %s: endpoint-less suppliers must remain proposal-only without hosted routes or agreements", path)
		}
	}
	return nil
}

func validateSupplierSourceAndModelScope(path string, policyJSON []byte) error {
	var source struct {
		Provider struct {
			Terms struct {
				PlanRules []sourceTermsPlanRule `json:"plan_rules"`
				Evidence  sourceTermsEvidence   `json:"evidence"`
			} `json:"terms"`
		} `json:"provider"`
		Models map[string]struct {
			Provenance struct {
				Terms struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"terms"`
			} `json:"provenance"`
			Caveats []string `json:"caveats"`
		} `json:"models"`
		Research struct {
			Activation  string   `json:"activation"`
			ModelReason string   `json:"model_reason"`
			SourceScope string   `json:"source_scope"`
			Evidence    []string `json:"evidence"`
		} `json:"research"`
	}
	if err := json.Unmarshal(policyJSON, &source); err != nil {
		return fmt.Errorf("invalid supplier declaration %s: decode source/scope records: %w", path, err)
	}
	if err := validateTermsEvidence(path, source.Provider.Terms.Evidence); err != nil {
		return err
	}
	if err := validateTermsPlanRules(path, source.Provider.Terms.PlanRules); err != nil {
		return err
	}
	if err := validateScopedTermAnchors(path, policyJSON); err != nil {
		return err
	}
	if strings.TrimSpace(source.Research.Activation) == "" || strings.TrimSpace(source.Research.ModelReason) == "" || !hasExplicitScopeUnknown(source.Research.ModelReason) || strings.TrimSpace(source.Research.SourceScope) == "" || len(source.Research.Evidence) == 0 {
		return fmt.Errorf("invalid supplier declaration %s: research needs activation boundary, model-scope reason, source scope, and evidence links", path)
	}
	for _, sourceURL := range source.Research.Evidence {
		if !hasDirectHTTPSCatalogURL(sourceURL) {
			return fmt.Errorf("invalid supplier declaration %s: research evidence must be direct HTTPS URLs", path)
		}
	}
	for modelID, model := range source.Models {
		if !hasDirectHTTPSCatalogURL(model.Provenance.Terms.Source) {
			return fmt.Errorf("invalid supplier declaration %s: model %q needs a direct HTTPS model-scope terms source", path, modelID)
		}
		if _, err := time.Parse("2006-01-02", model.Provenance.Terms.ObservedAt); err != nil {
			return fmt.Errorf("invalid supplier declaration %s: model %q terms provenance observed_at must use YYYY-MM-DD", path, modelID)
		}
		if len(model.Caveats) == 0 || !hasExplicitScopeUnknown(strings.Join(model.Caveats, " ")) {
			return fmt.Errorf("invalid supplier declaration %s: model %q needs an explicit scope or unresolved-rights caveat", path, modelID)
		}
	}
	return nil
}

func validateScopedTermAnchors(path string, policyJSON []byte) error {
	var policy struct {
		Provider struct {
			ID    string `json:"id"`
			Terms struct {
				Evidence  sourceTermsEvidence   `json:"evidence"`
				PlanRules []sourceTermsPlanRule `json:"plan_rules"`
			} `json:"terms"`
		} `json:"provider"`
		Models map[string]struct {
			Provenance struct {
				Terms struct {
					Source        string              `json:"source"`
					ObservedAt    string              `json:"observed_at"`
					SourceAnchors []sourceScopeAnchor `json:"source_anchors"`
				} `json:"terms"`
			} `json:"provenance"`
		} `json:"models"`
	}
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		return fmt.Errorf("invalid scoped terms anchors in %s: %w", path, err)
	}
	if policy.Provider.ID == "" {
		policy.Provider.ID = "openrouter"
	}
	for _, rule := range policy.Provider.Terms.PlanRules {
		if len(rule.SourceAnchors) == 0 {
			return fmt.Errorf("invalid terms scope in %s: plan %q has no source anchors", path, rule.Plan)
		}
		hasPrimaryAnchor := false
		for _, anchor := range rule.SourceAnchors {
			if anchor.URL == policy.Provider.Terms.Evidence.URL && anchor.Anchor == rule.Clause && anchor.DocumentDate == policy.Provider.Terms.Evidence.DocumentDate {
				hasPrimaryAnchor = true
			}
			if anchor.ClaimRef != rule.Plan || strings.TrimSpace(anchor.Anchor) == "" || strings.TrimSpace(anchor.Supports) == "" || !hasExplicitScopeUnknown(anchor.DoesNotEstablish) {
				return fmt.Errorf("invalid terms scope in %s: plan %q has a mismatched or overconfident source anchor", path, rule.Plan)
			}
			if !hasDirectHTTPSCatalogURL(anchor.URL) || !hasDocumentDateOrExplicitUnknown(anchor.DocumentDate) {
				return fmt.Errorf("invalid terms scope in %s: plan %q anchor needs direct URL and dated/explicitly undated source", path, rule.Plan)
			}
			if _, err := time.Parse("2006-01-02", anchor.ObservedAt); err != nil {
				return fmt.Errorf("invalid terms scope in %s: plan %q anchor has invalid observed_at", path, rule.Plan)
			}
			if anchor.URL == policy.Provider.Terms.Evidence.URL && anchor.DocumentDate != policy.Provider.Terms.Evidence.DocumentDate {
				return fmt.Errorf("invalid terms scope in %s: plan %q anchor date conflicts with the cited evidence page", path, rule.Plan)
			}
			if anchor.RetrievalStatus != "read_public_page" && anchor.RetrievalStatus != "content_unreadable" && anchor.RetrievalStatus != "current_recheck_unavailable" {
				return fmt.Errorf("invalid terms scope in %s: plan %q has unknown retrieval status", path, rule.Plan)
			}
			if anchor.RetrievalStatus != "read_public_page" && !hasExplicitScopeUnknown(anchor.DoesNotEstablish+" "+anchor.Supports) {
				return fmt.Errorf("invalid terms scope in %s: plan %q unavailable source must remain unresolved", path, rule.Plan)
			}
			if anchor.SupplierID != policy.Provider.ID {
				return fmt.Errorf("invalid terms scope in %s: plan %q anchor supplier mismatch", path, rule.Plan)
			}
		}
		if !hasPrimaryAnchor {
			return fmt.Errorf("invalid terms scope in %s: plan %q must retain its exact primary source/page anchor", path, rule.Plan)
		}
	}
	for modelID, model := range policy.Models {
		terms := model.Provenance.Terms
		if len(terms.SourceAnchors) == 0 {
			return fmt.Errorf("invalid terms scope in %s: model %q has no source anchors", path, modelID)
		}
		for _, anchor := range terms.SourceAnchors {
			if anchor.SupplierID != policy.Provider.ID || anchor.ModelID != modelID {
				return fmt.Errorf("invalid terms scope in %s: model %q anchor identity mismatch", path, modelID)
			}
			if !hasDirectHTTPSCatalogURL(anchor.URL) || !strings.Contains(terms.Source, anchor.URL) {
				return fmt.Errorf("invalid terms scope in %s: model %q anchor URL does not match the terms source", path, modelID)
			}
			if strings.TrimSpace(anchor.Anchor) == "" || strings.TrimSpace(anchor.Supports) == "" || !hasExplicitScopeUnknown(anchor.DoesNotEstablish) {
				return fmt.Errorf("invalid terms scope in %s: model %q anchor must state supported scope and unresolved rights", path, modelID)
			}
			if _, err := time.Parse("2006-01-02", anchor.ObservedAt); err != nil || anchor.ObservedAt != terms.ObservedAt {
				return fmt.Errorf("invalid terms scope in %s: model %q anchor observation does not match terms provenance", path, modelID)
			}
			if !hasDocumentDateOrExplicitUnknown(anchor.DocumentDate) {
				return fmt.Errorf("invalid terms scope in %s: model %q anchor needs a dated or explicitly undated source", path, modelID)
			}
			if anchor.RetrievalStatus != "read_public_page" && anchor.RetrievalStatus != "content_unreadable" && anchor.RetrievalStatus != "current_recheck_unavailable" {
				return fmt.Errorf("invalid terms scope in %s: model %q has unknown retrieval status", path, modelID)
			}
		}
	}
	return nil
}

func hasDocumentDateOrExplicitUnknown(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return true
	}
	for index := 0; index+len("2006-01-02") <= len(value); index++ {
		if _, err := time.Parse("2006-01-02", value[index:index+len("2006-01-02")]); err == nil {
			return true
		}
	}
	value = strings.ToLower(value)
	for _, marker := range []string{"no visible date", "no effective date", "no date", "undated", "not stated", "not readable", "date unknown", "unknown date"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func hasExplicitScopeUnknown(text string) bool {
	text = strings.ToLower(text)
	for _, marker := range []string{
		"unknown", "unverified", "not established", "not proven", "not verified", "not disclosed", "not identified", "not confirmed", "not readable", "not evaluated", "not been evaluated", "not account-scoped", "not account scoped", "not the actual", "not actual", "does not establish", "do not establish", "does not identify", "do not identify", "does not prove", "do not prove", "does not grant", "do not grant", "does not authorize", "do not authorize", "unresolved", "remain separate", "may change", "remains conditional",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func validateExactModelIdentityProvenance(path string, policy []byte) error {
	var identityPolicy struct {
		Models map[string]struct {
			Provenance struct {
				Identity struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"identity"`
			} `json:"provenance"`
		} `json:"models"`
	}
	if err := json.Unmarshal(policy, &identityPolicy); err != nil {
		return fmt.Errorf("invalid model policy %s: %w", path, err)
	}
	if len(identityPolicy.Models) == 0 {
		return fmt.Errorf("invalid model policy %s: exact model identity source gate requires at least one model", path)
	}
	for modelID, model := range identityPolicy.Models {
		if err := validateExactModelIdentitySource(path, modelID, model.Provenance.Identity.Source, model.Provenance.Identity.ObservedAt); err != nil {
			return err
		}
	}
	return nil
}

func validateExactModelIdentitySource(path, modelID, source, observedAt string) error {
	if !hasDirectHTTPSCatalogURL(source) || !strings.Contains(source, modelID) {
		return fmt.Errorf("invalid model policy %s: model %q needs direct HTTPS identity provenance naming the exact ID", path, modelID)
	}
	if _, err := time.Parse("2006-01-02", observedAt); err != nil {
		return fmt.Errorf("invalid model policy %s: model %q needs identity provenance observed_at in YYYY-MM-DD form", path, modelID)
	}
	return nil
}

func hasDirectHTTPSCatalogURL(source string) bool {
	fields := strings.Fields(strings.TrimSpace(source))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "https://") {
		return false
	}
	parsed, err := url.Parse(fields[0])
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != ""
}

func validateNoResourceDeclaration(root, supplierID string) error {
	resourcePath := filepath.Join(root, "resources", supplierID, "resource.json")
	if _, err := os.Stat(resourcePath); err == nil {
		return fmt.Errorf("supplier %s duplicates resource declaration %s", supplierID, resourcePath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check supplier %s resource declaration: %w", supplierID, err)
	}
	return nil
}

func loadHarnesses(root string) (map[string]json.RawMessage, error) {
	paths, err := filepath.Glob(filepath.Join(root, "resources", "*", "model-policy.json"))
	if err != nil {
		return nil, err
	}
	harnesses := map[string]json.RawMessage{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var policy struct {
			Billing json.RawMessage `json:"billing"`
		}
		if err := json.Unmarshal(data, &policy); err != nil {
			return nil, err
		}
		if len(policy.Billing) > 0 {
			harnesses[filepath.Base(filepath.Dir(path))] = data
		}
	}
	return harnesses, nil
}

func renderCatalog(inputs catalogInputs, entries map[string]any, harnesses map[string]json.RawMessage) ([]byte, error) {
	data, err := json.MarshalIndent(map[string]any{"roles": inputs.roles, "features": inputs.features, "credit_feature_owners": inputs.creditOwners, "providers": entries, "harnesses": harnesses}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func writeCatalog(output string, data []byte, check bool) error {
	old, _ := os.ReadFile(output)
	if bytes.Equal(old, data) {
		return nil
	}
	if check {
		return fmt.Errorf("provider catalog stale; run make provider-catalog from scenarios/ai-gateway")
	}
	return os.WriteFile(output, data, 0o644)
}
