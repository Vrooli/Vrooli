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
	paths, err := filepath.Glob(filepath.Join(s.Root, "resources", "*", "model-policy.json"))
	if s.Root == "" || s.Catalog != nil {
		bundle, bundleErr := s.bundle()
		if bundleErr != nil {
			return result, bundleErr
		}
		paths = nil
		for id := range bundle.Providers {
			paths = append(paths, filepath.Join("resources", id, "model-policy.json"))
		}
		sort.Strings(paths)
	}
	if err != nil {
		return result, err
	}
	for _, path := range paths {
		data, err := s.readPolicy(filepath.Base(filepath.Dir(path)))
		if err != nil {
			return result, err
		}
		var catalog supplierCatalog
		if json.Unmarshal(data, &catalog) != nil {
			return result, errors.New("invalid supplier policy catalog")
		}
		if catalog.Provider == nil || catalog.Provider.Terms == nil {
			continue
		}
		provider := catalog.Provider
		models := map[string]policy.ModelRule{}
		for name, model := range catalog.Models {
			models[name] = policy.ModelRule{TermsOverride: model.TermsOverride}
			if model.Licence != nil {
				rule := models[name]
				rule.LicenceCommercialUse = model.Licence.CommercialUse
				models[name] = rule
			}
		}
		digest, err := policy.ProviderRuleDigest(provider.ID, *provider.Terms, models)
		if err != nil {
			return result, err
		}
		verify := func(ref string) bool { return s.verify(ctx, ref, digest, policy.ProviderDecisionScopeRule) }
		observation, err := observePolicyReview(*provider.Terms, s.now(), verify)
		if err != nil {
			continue // Unknown review states close this supplier only.
		}
		status, err := transitionPolicyReview(provider.Terms.Review.Status, observation)
		if err != nil {
			continue
		}
		terms := *provider.Terms
		terms.Review.Status = status
		// Verification happened for this exact digest above. Do not perform a
		// second Hub lookup with a potentially different answer or deadline.
		record, err := policy.NewVerifiedRuleRecord(provider.ID, terms, models, func(string) bool { return observation == policyApprovalVerified })
		if err != nil {
			continue
		}
		result.Records = append(result.Records, record)
		if observation == policyApprovalUnverified {
			continue // Preserve supplier-local closure before any route/agreement work.
		}
		for _, route := range provider.Routes {
			if route.Supplier != provider.ID {
				return result, errors.New("route supplier differs from its resource owner")
			}
			if route.Harness != "" {
				route.HarnessSource = s.harnessSource(ctx, route.Harness)
			}
			result.Routes = append(result.Routes, route)
		}
		for _, agreement := range provider.Agreements {
			digest, err := policy.ProviderAgreementDigest(agreement)
			if err != nil {
				return result, err
			}
			verified, err := policy.NewVerifiedAgreement(agreement, func(ref string) bool { return s.verify(ctx, ref, digest, policy.ProviderDecisionScopeAgreement) })
			if err == nil {
				result.Agreements = append(result.Agreements, verified)
			}
		}
	}
	return result, nil
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
	var data []byte
	var err error
	if s.Root == "" || s.Catalog != nil {
		bundle, bundleErr := s.bundle()
		if bundleErr != nil {
			return SupplierEndpoint{}, bundleErr
		}
		p, ok := bundle.Providers[supplier]
		if !ok {
			return SupplierEndpoint{}, errors.New("supplier absent from packaged catalog")
		}
		data, err = json.Marshal(map[string]any{"endpoint": p.Endpoint, "credentials": map[string]any{"descriptors": []map[string]string{{"logical_id": p.CredentialIdentity, "field": p.CredentialField}}}})
	} else {
		data, err = os.ReadFile(filepath.Join(s.Root, "resources", supplier, "resource.json"))
	}
	if err != nil {
		return SupplierEndpoint{}, err
	}
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
		return SupplierEndpoint{}, errors.New("invalid resource endpoint")
	}
	result := SupplierEndpoint{URL: strings.TrimSuffix(resource.Endpoint, "/models")}
	if len(resource.Credentials.Descriptors) > 0 && resource.Credentials.Descriptors[0].Identity != "" {
		descriptor := resource.Credentials.Descriptors[0]
		authority, err := credentialauthority.Default()
		if err != nil {
			return result, err
		}
		identity, err := credentialauthority.ParseIdentity(descriptor.Identity)
		if err != nil {
			return result, err
		}
		result.APIKey, err = authority.Require(identity, descriptor.Field)
		if err != nil {
			return result, err
		}
	}
	return result, nil
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

func NewResourcePrivatePolicyService(db *sql.DB, features []PrivateFeature, root string, hub *ProviderDecisionHubClient) *PrivatePolicyService {
	source := ResourcePolicySource{Root: root, Hub: hub}
	features = append(features, PrivateFeature{ID: "agent.session", Role: "chat", Profile: "balanced", MaxInputBytes: 1 << 20, MaxOutputBytes: 1 << 20, MaxUserTurns: 1})
	service := NewPrivatePolicyService(features, source.Snapshot, NewSQLPrivatePlanRepository(db), NewSQLPrivateReceiptRepository(db))
	service.SetStreamExecutor(&HTTPStreamExecutor{DB: db, ResolveSupplier: source.Endpoint})
	return service
}
