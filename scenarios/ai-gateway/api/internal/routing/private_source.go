package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ai-gateway/internal/catalog"

	"github.com/vrooli/ai-provider-go/policy"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// ResourcePolicySource reads resource-owned supplier data on each admission;
// neither a cached approval nor a caller's feature label grants permission.
type ResourcePolicySource struct {
	Root           string
	Hub            *ProviderDecisionHubClient
	VerifyDecision func(context.Context, string, string, string) bool
	Catalog        *catalog.Bundle
	CatalogError   error
	clock          func() time.Time
}
type supplierCatalog struct {
	Provider *struct {
		ID         string                `json:"id"`
		Terms      *policy.ProviderTerms `json:"terms"`
		Routes     []policy.Route        `json:"hosted_routes"`
		Agreements []policy.Agreement    `json:"agreements"`
	} `json:"provider"`
	Models map[string]policy.Model `json:"models"`
}

func (s ResourcePolicySource) Snapshot(ctx context.Context, _ string) (PrivatePolicySnapshot, error) {
	result := PrivatePolicySnapshot{Account: policy.Account{BalanceStanding: true}}
	paths, err := s.snapshotPaths()
	if err != nil {
		return result, err
	}
	for _, path := range paths {
		if err := s.appendSupplierSnapshot(ctx, path, &result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s ResourcePolicySource) snapshotPaths() ([]string, error) {
	if s.CatalogError != nil {
		return nil, errors.New("embedded provider catalog is unavailable")
	}
	paths, err := filepath.Glob(filepath.Join(s.Root, "resources", "*", "model-policy.json"))
	if err != nil {
		return nil, err
	}
	if s.Root != "" && s.Catalog == nil {
		supplierPaths, supplierErr := filepath.Glob(filepath.Join(s.Root, "scenarios", "ai-gateway", "config", "suppliers", "*.json"))
		if supplierErr != nil {
			return nil, supplierErr
		}
		paths = append(paths, supplierPaths...)
		sort.Strings(paths)
		return paths, nil
	}
	if s.Root == "" || s.Catalog != nil {
		bundle, bundleErr := s.bundle()
		if bundleErr != nil {
			return nil, bundleErr
		}
		paths = nil
		for id := range bundle.Providers {
			paths = append(paths, filepath.Join("resources", id, "model-policy.json"))
		}
		sort.Strings(paths)
	}
	return paths, nil
}

func (s ResourcePolicySource) appendSupplierSnapshot(ctx context.Context, path string, snapshot *PrivatePolicySnapshot) error {
	var data []byte
	var err error
	if filepath.Base(filepath.Dir(path)) == "suppliers" && filepath.Base(filepath.Dir(filepath.Dir(path))) == "config" {
		data, err = os.ReadFile(path)
	} else {
		data, err = s.readPolicy(filepath.Base(filepath.Dir(path)))
	}
	if err != nil {
		return err
	}
	var catalog supplierCatalog
	if json.Unmarshal(data, &catalog) != nil {
		return errors.New("invalid supplier policy catalog")
	}
	if catalog.Provider == nil || catalog.Provider.Terms == nil {
		return nil
	}
	provider := catalog.Provider
	models := supplierModelRules(catalog.Models)
	record, observation, keep, err := s.verifiedSupplierPolicy(ctx, provider, models)
	if err != nil {
		return err
	}
	if !keep {
		return nil
	}
	snapshot.Records = append(snapshot.Records, record)
	if observation == policyApprovalUnverified {
		return nil
	}
	if err := s.appendSupplierRoutes(ctx, provider, snapshot); err != nil {
		return err
	}
	return s.appendSupplierAgreements(ctx, provider, snapshot)
}

func (s ResourcePolicySource) verifiedSupplierPolicy(ctx context.Context, provider *struct {
	ID         string                `json:"id"`
	Terms      *policy.ProviderTerms `json:"terms"`
	Routes     []policy.Route        `json:"hosted_routes"`
	Agreements []policy.Agreement    `json:"agreements"`
}, models map[string]policy.ModelRule,
) (policy.RuleRecord, policyReviewEvent, bool, error) {
	digest, err := policy.ProviderRuleDigest(provider.ID, *provider.Terms, models)
	if err != nil {
		return policy.RuleRecord{}, "", false, err
	}
	verify := func(ref string) bool { return s.verify(ctx, ref, digest, policy.ProviderDecisionScopeRule) }
	observation, err := observePolicyReview(*provider.Terms, s.now(), verify)
	if err != nil {
		return policy.RuleRecord{}, "", false, nil
	}
	status, err := transitionPolicyReview(provider.Terms.Review.Status, observation)
	if err != nil {
		return policy.RuleRecord{}, "", false, nil
	}
	terms := *provider.Terms
	terms.Review.Status = status
	record, err := policy.NewVerifiedRuleRecord(provider.ID, terms, models, func(string) bool { return observation == policyApprovalVerified })
	if err != nil {
		return policy.RuleRecord{}, "", false, nil
	}
	return record, observation, true, nil
}

func supplierModelRules(models map[string]policy.Model) map[string]policy.ModelRule {
	rules := make(map[string]policy.ModelRule, len(models))
	for name, model := range models {
		rule := policy.ModelRule{TermsOverride: model.TermsOverride}
		if model.Licence != nil {
			rule.LicenceCommercialUse = model.Licence.CommercialUse
		}
		rules[name] = rule
	}
	return rules
}

func (s ResourcePolicySource) appendSupplierRoutes(ctx context.Context, provider *struct {
	ID         string                `json:"id"`
	Terms      *policy.ProviderTerms `json:"terms"`
	Routes     []policy.Route        `json:"hosted_routes"`
	Agreements []policy.Agreement    `json:"agreements"`
}, snapshot *PrivatePolicySnapshot,
) error {
	for _, route := range provider.Routes {
		if route.Supplier != provider.ID {
			return errors.New("route supplier differs from its resource owner")
		}
		if route.Harness != "" {
			route.HarnessSource = s.harnessSource(ctx, route.Harness)
		}
		snapshot.Routes = append(snapshot.Routes, route)
	}
	return nil
}

func (s ResourcePolicySource) appendSupplierAgreements(ctx context.Context, provider *struct {
	ID         string                `json:"id"`
	Terms      *policy.ProviderTerms `json:"terms"`
	Routes     []policy.Route        `json:"hosted_routes"`
	Agreements []policy.Agreement    `json:"agreements"`
}, snapshot *PrivatePolicySnapshot,
) error {
	for _, agreement := range provider.Agreements {
		digest, err := policy.ProviderAgreementDigest(agreement)
		if err != nil {
			return err
		}
		verified, err := policy.NewVerifiedAgreement(agreement, func(ref string) bool { return s.verify(ctx, ref, digest, policy.ProviderDecisionScopeAgreement) })
		if err == nil {
			snapshot.Agreements = append(snapshot.Agreements, verified)
		}
	}
	return nil
}

func (s ResourcePolicySource) harnessSource(ctx context.Context, harness string) *policy.HarnessBillingSource {
	if harness == "" || filepath.Base(harness) != harness {
		return nil
	}
	data, err := s.readPolicy(harness)
	if err != nil {
		return nil
	}
	var record struct {
		Billing struct {
			Sources []policy.HarnessBillingSource `json:"sources"`
		} `json:"billing"`
	}
	if json.Unmarshal(data, &record) != nil {
		return nil
	}
	var found *policy.HarnessBillingSource
	for _, source := range record.Billing.Sources {
		if source.ID != "vrooli-credits" {
			continue
		}
		if found != nil {
			return nil
		}
		digest, err := policy.HarnessBillingSourceDigest(harness, source)
		if err != nil {
			return nil
		}
		verified, err := policy.NewVerifiedHarnessBillingSource(source, func(ref string) bool { return s.verify(ctx, ref, digest, policy.ProviderDecisionScopeRule) })
		if err != nil {
			return nil
		}
		found = &verified
	}
	return found
}

func (s ResourcePolicySource) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}

func (s ResourcePolicySource) verify(ctx context.Context, ref, digest, scope string) bool {
	if s.VerifyDecision != nil {
		return s.VerifyDecision(ctx, ref, digest, scope)
	}
	if s.Hub == nil {
		return false
	}
	actual, err := s.Hub.Lookup(ctx, ref, digest, scope)
	if err != nil {
		return false
	}
	data, _ := json.Marshal(actual)
	var receipt policy.ProviderDecisionReceipt
	if json.Unmarshal(data, &receipt) != nil {
		return false
	}
	expected := policy.ProviderDecisionReceipt{Source: "ai-gateway", EventType: policy.ProviderDecisionEventType, Requester: ref, CanonicalTermsSHA256: digest, Scope: scope}
	return policy.VerifyProviderDecisionReceipt(expected, receipt, s.now()) == nil
}

func (s ResourcePolicySource) Endpoint(ctx context.Context, supplier string) (SupplierEndpoint, error) {
	if supplier == "" || filepath.Base(supplier) != supplier {
		return SupplierEndpoint{}, errors.New("invalid supplier")
	}
	data, err := s.resourceEndpointData(supplier)
	if err != nil {
		return SupplierEndpoint{}, err
	}
	resource, err := decodeSupplierEndpoint(data)
	if err != nil {
		return SupplierEndpoint{}, err
	}
	endpoint := strings.TrimSpace(resource.Endpoint)
	if endpoint == "" {
		return SupplierEndpoint{}, errors.New("supplier endpoint is not configured")
	}
	result := SupplierEndpoint{URL: strings.TrimSuffix(endpoint, "/models")}
	result.APIKey, err = s.supplierCredential(resource.Credentials.Descriptors)
	return result, err
}

func (s ResourcePolicySource) resourceEndpointData(supplier string) ([]byte, error) {
	if s.CatalogError != nil {
		return nil, errors.New("embedded provider catalog is unavailable")
	}
	var data []byte
	var err error
	if s.Root == "" || s.Catalog != nil {
		bundle, bundleErr := s.bundle()
		if bundleErr != nil {
			return nil, bundleErr
		}
		p, ok := bundle.Providers[supplier]
		if !ok {
			return nil, errors.New("supplier absent from packaged catalog")
		}
		data, err = json.Marshal(map[string]any{"endpoint": p.Endpoint, "credentials": map[string]any{"descriptors": []map[string]string{{"logical_id": p.CredentialIdentity, "field": p.CredentialField}}}})
	} else {
		data, err = os.ReadFile(filepath.Join(s.Root, "resources", supplier, "resource.json"))
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func decodeSupplierEndpoint(data []byte) (struct {
	Endpoint    string `json:"endpoint"`
	Credentials struct {
		Descriptors []struct {
			Identity string `json:"logical_id"`
			Field    string `json:"field"`
		} `json:"descriptors"`
	} `json:"credentials"`
}, error,
) {
	var resource struct {
		Endpoint    string `json:"endpoint"`
		Credentials struct {
			Descriptors []struct {
				Identity string `json:"logical_id"`
				Field    string `json:"field"`
			} `json:"descriptors"`
		} `json:"credentials"`
	}
	if json.Unmarshal(data, &resource) != nil {
		return resource, errors.New("invalid resource endpoint")
	}
	return resource, nil
}

func (s ResourcePolicySource) supplierCredential(descriptors []struct {
	Identity string `json:"logical_id"`
	Field    string `json:"field"`
},
) (string, error) {
	if len(descriptors) == 0 || descriptors[0].Identity == "" {
		return "", nil
	}
	descriptor := descriptors[0]
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(descriptor.Identity)
	if err != nil {
		return "", err
	}
	return authority.Require(identity, descriptor.Field)
}

func (s ResourcePolicySource) bundle() (catalog.Bundle, error) {
	if s.Catalog != nil {
		return *s.Catalog, nil
	}
	return catalog.Embedded()
}

func (s ResourcePolicySource) readPolicy(id string) ([]byte, error) {
	if s.Root != "" && s.Catalog == nil {
		return os.ReadFile(filepath.Join(s.Root, "resources", id, "model-policy.json"))
	}
	bundle, err := s.bundle()
	if err != nil {
		return nil, err
	}
	if p, ok := bundle.Providers[id]; ok {
		return p.Policy, nil
	}
	if p, ok := bundle.Harnesses[id]; ok {
		return p, nil
	}
	return nil, errors.New("policy absent from packaged catalog")
}

// NewPackagedResourcePolicySource always uses the catalog compiled into the
// Gateway artifact. Hosted deployments cannot rely on provider resource
// trees being present, and a repository checkout must not shadow the packaged
// declarations at runtime.
func NewPackagedResourcePolicySource(hub *ProviderDecisionHubClient) ResourcePolicySource {
	bundle, err := catalog.Embedded()
	if err != nil {
		return ResourcePolicySource{Hub: hub, CatalogError: err}
	}
	return ResourcePolicySource{Hub: hub, Catalog: &bundle}
}

func NewResourcePrivatePolicyService(db *sql.DB, features []PrivateFeature, hub *ProviderDecisionHubClient) *PrivatePolicyService {
	source := NewPackagedResourcePolicySource(hub)
	service := NewPrivatePolicyService(features, source.Snapshot, NewSQLPrivatePlanRepository(db), NewSQLPrivateReceiptRepository(db))
	service.SetStreamExecutor(&HTTPStreamExecutor{DB: db, ResolveSupplier: source.Endpoint})
	return service
}
