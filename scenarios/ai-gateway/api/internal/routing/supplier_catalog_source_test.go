package routing

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ai-gateway/internal/catalog"
	"github.com/vrooli/ai-provider-go/policy"
)

const proposedSupplierFixture = `{"provider":{"id":"nanogpt","terms":{"plan_rules":[{"plan":"pay_as_you_go","access_tier":"1a","use_classes":{"harness_access":"allowed"}}],"evidence":{"url":"https://nano-gpt.com/legal/terms-of-service","read_at":"2026-11-22"},"review":{"status":"proposed"},"effective_from":"2026-11-22","on_expiry":"deny"},"hosted_routes":[],"agreements":[]},"models":{"z-ai/glm-5.3":{"family":"glm-5","capabilities":["chat","text_input","text_output"],"modalities":{"input":["text"],"output":["text"]},"endpoints":["chat"],"default_eligible":false,"provenance":{"terms":{"source_kind":"manual_policy","confidence":"low","source":"AC2 public terms research; actual serving provider terms unresolved","observed_at":"2026-11-22"}}}}}`

func TestResourcePolicySourceReadsProposedSupplierFromRepositoryAndHostedBundle(t *testing.T) {
	t.Run("repository supplier file remains denied", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "scenarios", "ai-gateway", "config", "suppliers", "nanogpt.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(proposedSupplierFixture), 0o600); err != nil {
			t.Fatal(err)
		}
		snapshot, err := (ResourcePolicySource{Root: root}).Snapshot(context.Background(), "fixture")
		if err != nil {
			t.Fatal(err)
		}
		assertProposedSupplierDenied(t, snapshot)
	})

	t.Run("hosted bundle supplier file remains denied", func(t *testing.T) {
		var raw json.RawMessage = []byte(proposedSupplierFixture)
		bundle := catalog.Bundle{Providers: map[string]catalog.Provider{
			"nanogpt": {ID: "nanogpt", Family: "openai-compatible", Locality: "remote", Endpoint: "https://api.nano-gpt.com/api/v1", Policy: raw},
		}}
		snapshot, err := (ResourcePolicySource{Catalog: &bundle}).Snapshot(context.Background(), "fixture")
		if err != nil {
			t.Fatal(err)
		}
		assertProposedSupplierDenied(t, snapshot)
	})
}

// This exercises the same embedded provider bundle used when Gateway runs
// without a repository checkout. It guards the J13 package boundary: supplier
// declarations must be present in the compiled catalog, and merely packaging
// a draft must not make it executable.
func TestAC2EmbeddedCatalogProjectsEverySupplierDeclarationAsDraft(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "..", "config", "suppliers"))
	if err != nil {
		t.Fatalf("read supplier declarations: %v", err)
	}
	source := NewPackagedResourcePolicySource(nil)
	if source.Root != "" || source.Catalog == nil || source.CatalogError != nil {
		t.Fatalf("packaged policy source is not rooted in the embedded catalog: %+v", source)
	}
	bundle := *source.Catalog
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))]
		declaration, err := os.ReadFile(filepath.Join("..", "..", "..", "config", "suppliers", entry.Name()))
		if err != nil {
			t.Fatalf("read supplier declaration %q: %v", id, err)
		}
		var supplier struct {
			ID     string          `json:"id"`
			Policy json.RawMessage `json:"policy"`
		}
		if err := json.Unmarshal(declaration, &supplier); err != nil {
			t.Fatalf("decode supplier declaration %q: %v", id, err)
		}
		if supplier.ID != id {
			t.Fatalf("supplier declaration filename %q contains id %q", id, supplier.ID)
		}
		projected, ok := bundle.Providers[id]
		if !ok {
			t.Errorf("embedded provider catalog omits supplier declaration %q", id)
			continue
		}
		var declaredPolicy, packagedPolicy any
		if err := json.Unmarshal(supplier.Policy, &declaredPolicy); err != nil {
			t.Fatalf("decode supplier declaration policy %q: %v", id, err)
		}
		if err := json.Unmarshal(projected.Policy, &packagedPolicy); err != nil {
			t.Fatalf("decode packaged supplier policy %q: %v", id, err)
		}
		declared, _ := json.Marshal(declaredPolicy)
		packaged, _ := json.Marshal(packagedPolicy)
		if string(declared) != string(packaged) {
			t.Errorf("embedded supplier %q differs from its declaration", id)
		}
		endpoint, err := source.Endpoint(context.Background(), id)
		if projected.Endpoint == "" {
			if err == nil || endpoint.URL != "" {
				t.Errorf("endpoint-less proposal %q resolved to %+v, error %v; want fail closed", id, endpoint, err)
			}
		} else if err != nil {
			t.Errorf("resolve packaged supplier endpoint %q: %v", id, err)
		} else if endpoint.URL != projected.Endpoint || endpoint.APIKey != "" {
			t.Errorf("packaged supplier endpoint %q = %+v, want URL %q with no credential value", id, endpoint, projected.Endpoint)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		t.Fatal("no supplier declarations were discovered")
	}
	sort.Strings(ids)
	openRouter, ok := bundle.Providers["openrouter"]
	if !ok {
		t.Fatal("embedded provider catalog omits OpenRouter's existing registry entry")
	}
	var openRouterTerms struct {
		Provider struct {
			Terms struct {
				PlanRules []struct {
					Plan       string            `json:"plan"`
					UseClasses map[string]string `json:"use_classes"`
					Clause     string            `json:"clause"`
					Reason     string            `json:"reason"`
				} `json:"plan_rules"`
				Evidence struct {
					URL          string `json:"url"`
					DocumentDate string `json:"document_date"`
					ReadAt       string `json:"read_at"`
				} `json:"evidence"`
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(openRouter.Policy, &openRouterTerms); err != nil {
		t.Fatalf("decode embedded OpenRouter terms: %v", err)
	}
	if openRouterTerms.Provider.Terms.Review.Status != string(policy.ReviewProposed) || openRouterTerms.Provider.Terms.Evidence.URL != "https://openrouter.ai/terms" || openRouterTerms.Provider.Terms.Evidence.DocumentDate != "2026-08-31" || openRouterTerms.Provider.Terms.Evidence.ReadAt != "2026-11-23" {
		t.Fatalf("embedded OpenRouter terms source/review = %+v, want current sourced proposal", openRouterTerms.Provider.Terms)
	}
	foundStandardPlan := false
	for index := range openRouterTerms.Provider.Terms.PlanRules {
		standardPlan := openRouterTerms.Provider.Terms.PlanRules[index]
		if standardPlan.Plan == "pay_as_you_go" {
			foundStandardPlan = true
			if standardPlan.UseClasses["in_app"] != "forbidden" || standardPlan.UseClasses["harness_access"] != "forbidden" || standardPlan.Clause != "Terms of Service §§5.1–5.2, 7" || len(standardPlan.Reason) == 0 {
				t.Fatalf("embedded OpenRouter standard plan fails to deny resale explicitly: %+v", standardPlan)
			}
			break
		}
	}
	if !foundStandardPlan {
		t.Fatal("embedded OpenRouter terms omit the standard pay-as-you-go plan")
	}
	if len(openRouterTerms.Provider.Routes) != 0 || len(openRouterTerms.Provider.Agreements) != 0 {
		t.Fatalf("proposed OpenRouter terms added executable routes or agreements: routes=%d agreements=%d", len(openRouterTerms.Provider.Routes), len(openRouterTerms.Provider.Agreements))
	}
	openRouterEndpoint, err := source.Endpoint(context.Background(), "openrouter")
	if err != nil || openRouterEndpoint.URL != openRouter.Endpoint {
		t.Fatalf("existing OpenRouter candidate endpoint = %+v, err=%v; want catalog URL %q", openRouterEndpoint, err, openRouter.Endpoint)
	}
	orcaRouter, ok := bundle.Providers["orcarouter"]
	if !ok {
		t.Fatal("embedded provider catalog omits OrcaRouter's existing resource entry")
	}
	var orcaTerms struct {
		Provider struct {
			Terms struct {
				PlanRules []struct {
					Plan       string            `json:"plan"`
					UseClasses map[string]string `json:"use_classes"`
					Conditions []string          `json:"conditions"`
					Reason     string            `json:"reason"`
				} `json:"plan_rules"`
				Evidence struct {
					URL          string `json:"url"`
					DocumentDate string `json:"document_date"`
					ReadAt       string `json:"read_at"`
				} `json:"evidence"`
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(orcaRouter.Policy, &orcaTerms); err != nil {
		t.Fatalf("decode embedded OrcaRouter terms: %v", err)
	}
	if orcaTerms.Provider.Terms.Review.Status != string(policy.ReviewProposed) || orcaTerms.Provider.Terms.Evidence.URL != "https://www.orcarouter.ai/terms.html" || orcaTerms.Provider.Terms.Evidence.DocumentDate != "2026-04-18" || orcaTerms.Provider.Terms.Evidence.ReadAt != "2026-11-23" {
		t.Fatalf("embedded OrcaRouter terms source/review = %+v, want current sourced proposal", orcaTerms.Provider.Terms)
	}
	foundOrcaStandardPlan := false
	for _, plan := range orcaTerms.Provider.Terms.PlanRules {
		if plan.Plan != "pay_as_you_go" {
			continue
		}
		foundOrcaStandardPlan = true
		if plan.UseClasses["in_app"] != "unverified" || plan.UseClasses["harness_access"] != "unverified" || len(plan.Conditions) < 3 || len(plan.Reason) == 0 {
			t.Fatalf("embedded OrcaRouter standard plan no longer fails closed on unresolved downstream rights: %+v", plan)
		}
	}
	if !foundOrcaStandardPlan || len(orcaTerms.Provider.Routes) != 0 || len(orcaTerms.Provider.Agreements) != 0 {
		t.Fatalf("embedded OrcaRouter proposal has no unresolved standard plan or adds executable authority: found_plan=%t routes=%d agreements=%d", foundOrcaStandardPlan, len(orcaTerms.Provider.Routes), len(orcaTerms.Provider.Agreements))
	}
	candidateIDs := append(append([]string(nil), ids...), "openrouter")
	snapshot, err := source.Snapshot(context.Background(), "ac2-embedded-catalog")
	if err != nil {
		t.Fatalf("load embedded bundle into packaged policy source: %v", err)
	}
	seen := map[string]bool{}
	for _, record := range snapshot.Records {
		seen[record.Supplier] = true
		if containsString(candidateIDs, record.Supplier) && record.Terms.Review.Status != policy.ReviewProposed {
			t.Errorf("supplier %q review status = %q, want proposed", record.Supplier, record.Terms.Review.Status)
		}
	}
	for _, id := range candidateIDs {
		if !seen[id] {
			t.Errorf("packaged policy source omitted current candidate record %q", id)
		}
	}
	for _, route := range snapshot.Routes {
		if containsString(candidateIDs, route.Supplier) {
			t.Errorf("proposal-only supplier %q produced executable route %q", route.Supplier, route.ID)
		}
	}
	for _, agreement := range snapshot.Agreements {
		if containsString(candidateIDs, agreement.Supplier) {
			t.Errorf("proposal-only supplier %q produced executable agreement", agreement.Supplier)
		}
	}
}

func TestPackagedPolicySourceFailsClosedWhenEmbeddedCatalogCannotLoad(t *testing.T) {
	source := ResourcePolicySource{CatalogError: errors.New("fixture parse failure")}
	if _, err := source.Snapshot(context.Background(), "fixture"); err == nil {
		t.Fatal("snapshot succeeded without the embedded provider catalog")
	}
	if _, err := source.Endpoint(context.Background(), "nanogpt"); err == nil {
		t.Fatal("endpoint resolution succeeded without the embedded provider catalog")
	}
}

func TestAC2RouterOneSourcedModelDraftsRemainIneligible(t *testing.T) {
	source := NewPackagedResourcePolicySource(nil)
	if source.Catalog == nil || source.CatalogError != nil {
		t.Fatalf("embedded catalog unavailable: %v", source.CatalogError)
	}
	entry, ok := source.Catalog.Providers["router-one"]
	if !ok {
		t.Fatal("embedded catalog omits Router One")
	}
	var draft struct {
		Provider struct {
			Terms struct {
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Roles  map[string]json.RawMessage `json:"roles"`
		Models map[string]struct {
			Eligible   *bool `json:"default_eligible"`
			Provenance struct {
				Catalog struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"catalog"`
				Terms struct {
					Source string `json:"source"`
				} `json:"terms"`
			} `json:"provenance"`
			Caveats []string `json:"caveats"`
		} `json:"models"`
	}
	if err := json.Unmarshal(entry.Policy, &draft); err != nil {
		t.Fatalf("decode embedded Router One draft: %v", err)
	}
	want := []string{"openai/gpt-6.2-sol", "anthropic/claude-sonnet-5", "google/gemini-3.8-flash"}
	if draft.Provider.Terms.Review.Status != string(policy.ReviewProposed) || len(draft.Provider.Routes) != 0 || len(draft.Provider.Agreements) != 0 || len(draft.Roles) != 0 {
		t.Fatalf("Router One proposal acquired executable authority: review=%q routes=%d agreements=%d roles=%d", draft.Provider.Terms.Review.Status, len(draft.Provider.Routes), len(draft.Provider.Agreements), len(draft.Roles))
	}
	if len(draft.Models) != len(want) {
		t.Fatalf("Router One catalog model drafts = %d; want exactly %d reviewed samples", len(draft.Models), len(want))
	}
	for _, id := range want {
		rule, ok := draft.Models[id]
		if !ok || rule.Eligible == nil || *rule.Eligible {
			t.Errorf("Router One model %q draft missing or eligible: %+v", id, rule)
			continue
		}
		if rule.Provenance.Catalog.ObservedAt != "2026-11-23" || !strings.HasPrefix(rule.Provenance.Catalog.Source, "https://router.one/models/") {
			t.Errorf("Router One model %q lacks dated public catalog evidence: %+v", id, rule.Provenance.Catalog)
		}
		if len(rule.Caveats) < 2 || !strings.Contains(strings.Join(rule.Caveats, " "), "undisclosed") {
			t.Errorf("Router One model %q fails to record undisclosed upstream rights: %v", id, rule.Caveats)
		}
		if !strings.Contains(rule.Provenance.Terms.Source, "https://router.one/faq") {
			t.Errorf("Router One model %q lacks the upstream-route limitation source: %q", id, rule.Provenance.Terms.Source)
		}
	}
}

func TestAC2GroqCloudModelIdentityDraftsStayDisabledInEmbeddedCatalog(t *testing.T) {
	source := NewPackagedResourcePolicySource(nil)
	if source.Catalog == nil || source.CatalogError != nil {
		t.Fatalf("embedded catalog unavailable: %v", source.CatalogError)
	}
	entry, ok := source.Catalog.Providers["groqcloud"]
	if !ok {
		t.Fatal("embedded catalog omits GroqCloud")
	}
	var draft struct {
		Provider struct {
			Terms struct {
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Models map[string]struct {
			Eligible   *bool `json:"default_eligible"`
			Context    int   `json:"context_window_tokens"`
			Provenance struct {
				Catalog struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"catalog"`
				Terms struct {
					Source string `json:"source"`
				} `json:"terms"`
			} `json:"provenance"`
			Caveats []string `json:"caveats"`
		} `json:"models"`
	}
	if err := json.Unmarshal(entry.Policy, &draft); err != nil {
		t.Fatalf("decode embedded GroqCloud draft: %v", err)
	}
	want := map[string]int{
		"openai/gpt-oss-120b": 131072,
		"openai/gpt-oss-20b":  131072,
	}
	if draft.Provider.Terms.Review.Status != string(policy.ReviewProposed) || len(draft.Provider.Routes) != 0 || len(draft.Provider.Agreements) != 0 {
		t.Fatalf("GroqCloud proposal acquired executable authority: review=%q routes=%d agreements=%d", draft.Provider.Terms.Review.Status, len(draft.Provider.Routes), len(draft.Provider.Agreements))
	}
	if len(draft.Models) != len(want) {
		t.Fatalf("GroqCloud catalog model drafts = %d; want %d exact sourced identities", len(draft.Models), len(want))
	}
	for id, contextWindow := range want {
		rule, ok := draft.Models[id]
		if !ok || rule.Eligible == nil || *rule.Eligible || rule.Context != contextWindow {
			t.Errorf("GroqCloud model %q draft missing, enabled, or has wrong context window: %+v", id, rule)
			continue
		}
		if rule.Provenance.Catalog.ObservedAt != "2026-11-23" || !strings.Contains(rule.Provenance.Catalog.Source, "https://console.groq.com/docs/models") {
			t.Errorf("GroqCloud model %q lacks dated public catalog evidence: %+v", id, rule.Provenance.Catalog)
		}
		if !strings.Contains(rule.Provenance.Terms.Source, "https://console.groq.com/docs/legal/services-agreement") || len(rule.Caveats) < 3 {
			t.Errorf("GroqCloud model %q lacks terms source or rights caveats: provenance=%+v caveats=%v", id, rule.Provenance.Terms, rule.Caveats)
		}
	}
}

func TestAC2NanoGPTCurrentModelEvidenceDoesNotInferHostedRouteRights(t *testing.T) {
	source := NewPackagedResourcePolicySource(nil)
	if source.Catalog == nil || source.CatalogError != nil {
		t.Fatalf("embedded catalog unavailable: %v", source.CatalogError)
	}
	entry, ok := source.Catalog.Providers["nanogpt"]
	if !ok {
		t.Fatal("embedded catalog omits NanoGPT")
	}
	var draft struct {
		Provider struct {
			Terms struct {
				PlanRules []struct {
					Plan       string            `json:"plan"`
					UseClasses map[string]string `json:"use_classes"`
				} `json:"plan_rules"`
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Models map[string]struct {
			Eligible   *bool    `json:"default_eligible"`
			Caveats    []string `json:"caveats"`
			Provenance map[string]struct {
				Source     string `json:"source"`
				ObservedAt string `json:"observed_at"`
			} `json:"provenance"`
		} `json:"models"`
	}
	if err := json.Unmarshal(entry.Policy, &draft); err != nil {
		t.Fatalf("decode embedded NanoGPT draft: %v", err)
	}
	if draft.Provider.Terms.Review.Status != string(policy.ReviewProposed) || len(draft.Provider.Routes) != 0 || len(draft.Provider.Agreements) != 0 {
		t.Fatalf("NanoGPT proposal acquired executable authority: review=%q routes=%d agreements=%d", draft.Provider.Terms.Review.Status, len(draft.Provider.Routes), len(draft.Provider.Agreements))
	}
	foundPayg := false
	for _, plan := range draft.Provider.Terms.PlanRules {
		if plan.Plan == "pay_as_you_go" {
			foundPayg = true
			if plan.UseClasses["in_app"] != "unverified" || plan.UseClasses["harness_access"] != "allowed" {
				t.Errorf("human-only NanoGPT supplier terms item changed state: %+v", plan.UseClasses)
			}
		}
	}
	if !foundPayg {
		t.Fatal("NanoGPT proposal omits PAYG scope")
	}
	model, ok := draft.Models["z-ai/glm-5.3"]
	if !ok || model.Eligible == nil || *model.Eligible {
		t.Fatalf("NanoGPT model identity is absent or eligible: %+v", model)
	}
	for _, category := range []string{"catalog", "terms", "license"} {
		provenance, ok := model.Provenance[category]
		if !ok || provenance.ObservedAt != "2026-11-23" || provenance.Source == "" {
			t.Errorf("NanoGPT GLM-5.3 model draft missing dated %q provenance: %+v", category, provenance)
		}
	}
	if !strings.Contains(model.Provenance["catalog"].Source, "nano-gpt.com/models") || !strings.Contains(model.Provenance["license"].Source, "huggingface.co/zai-org/GLM-5.3") {
		t.Errorf("NanoGPT model identity lacks exact catalog/license sources: %+v", model.Provenance)
	}
	joined := strings.Join(model.Caveats, " ")
	if !strings.Contains(joined, "actual serving provider") || !strings.Contains(joined, "mere relaying") {
		t.Errorf("NanoGPT model draft fails to distinguish publisher license from hosted route: %v", model.Caveats)
	}
}

func TestAC2APInexConflictedCatalogRemainsEndpointlessAndModelless(t *testing.T) {
	source := NewPackagedResourcePolicySource(nil)
	if source.Catalog == nil || source.CatalogError != nil {
		t.Fatalf("embedded catalog unavailable: %v", source.CatalogError)
	}
	entry, ok := source.Catalog.Providers["apinex"]
	if !ok {
		t.Fatal("embedded catalog omits APInex")
	}
	if entry.Endpoint != "" {
		t.Fatalf("APInex unresolved candidate has endpoint %q; want endpointless proposal", entry.Endpoint)
	}
	if _, err := source.Endpoint(context.Background(), "apinex"); err == nil {
		t.Fatal("APInex endpoint resolved despite unreadable service terms and catalog mismatch")
	}
	var draft struct {
		Provider struct {
			Terms struct {
				Evidence struct {
					URL    string `json:"url"`
					ReadAt string `json:"read_at"`
				} `json:"evidence"`
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
				PlanRules []struct {
					UseClasses map[string]string `json:"use_classes"`
				} `json:"plan_rules"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Models   map[string]json.RawMessage `json:"models"`
		Research struct {
			Activation  string `json:"activation"`
			ModelReason string `json:"model_reason"`
		} `json:"research"`
	}
	if err := json.Unmarshal(entry.Policy, &draft); err != nil {
		t.Fatalf("decode embedded APInex proposal: %v", err)
	}
	if draft.Provider.Terms.Review.Status != string(policy.ReviewProposed) || draft.Provider.Terms.Evidence.ReadAt != "2026-11-23" || draft.Provider.Terms.Evidence.URL != "https://apinex.bond/developers/resell" {
		t.Fatalf("APInex terms evidence/review = %+v; want current official reseller guide and proposal", draft.Provider.Terms)
	}
	if len(draft.Models) != 0 || len(draft.Provider.Routes) != 0 || len(draft.Provider.Agreements) != 0 {
		t.Fatalf("APInex source gap created model or executable authority: models=%d routes=%d agreements=%d", len(draft.Models), len(draft.Provider.Routes), len(draft.Provider.Agreements))
	}
	if len(draft.Provider.Terms.PlanRules) != 1 || draft.Provider.Terms.PlanRules[0].UseClasses["in_app"] != "unverified" || draft.Provider.Terms.PlanRules[0].UseClasses["harness_access"] != "unverified" {
		t.Fatalf("APInex conditional path became verified: %+v", draft.Provider.Terms.PlanRules)
	}
	if !strings.Contains(draft.Research.ModelReason, "53 versus 36+") || !strings.Contains(draft.Research.ModelReason, "no substantive API Terms") || !strings.Contains(draft.Research.Activation, "denied") {
		t.Fatalf("APInex unresolved evidence is not preserved fail-closed: %+v", draft.Research)
	}
}

func TestAC2APmixResaleCatalogIdentitiesRemainDisabledWithoutRouteAuthority(t *testing.T) {
	source := NewPackagedResourcePolicySource(nil)
	if source.Catalog == nil || source.CatalogError != nil {
		t.Fatalf("embedded catalog unavailable: %v", source.CatalogError)
	}
	entry, ok := source.Catalog.Providers["apmix"]
	if !ok {
		t.Fatal("embedded catalog omits APmix")
	}
	if entry.Endpoint != "https://api.apmix.ai/v1" {
		t.Fatalf("APmix public API endpoint = %q; want sourced documented base URL", entry.Endpoint)
	}
	var draft struct {
		Provider struct {
			Terms struct {
				Evidence struct {
					ReadAt string `json:"read_at"`
				} `json:"evidence"`
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Models map[string]struct {
			Eligible   *bool `json:"default_eligible"`
			Provenance map[string]struct {
				Confidence string `json:"confidence"`
				ObservedAt string `json:"observed_at"`
				Source     string `json:"source"`
			} `json:"provenance"`
		} `json:"models"`
		Research struct {
			ModelReason string `json:"model_reason"`
		} `json:"research"`
	}
	if err := json.Unmarshal(entry.Policy, &draft); err != nil {
		t.Fatalf("decode embedded APmix proposal: %v", err)
	}
	if draft.Provider.Terms.Review.Status != string(policy.ReviewProposed) || draft.Provider.Terms.Evidence.ReadAt != "2026-11-23" {
		t.Fatalf("APmix terms evidence/review = %+v; want current sourced proposal", draft.Provider.Terms)
	}
	wantIDs := []string{
		"grok-4.5", "qwen3.8-flash", "glm-5.3-flash", "glm-5.3", "glm-5.2", "qwen3.8-max",
		"claude-sonnet-4-6", "grok-4.6", "kimi-k3", "grok-4.7", "gpt-6-sol", "gpt-6-luna",
		"gpt-6.1-sol", "kimi-k2.8-preview", "minimax-m3.1-flash", "mimo-v2.6-flash", "mimo-v2.6-pro",
		"kimi-k3.1-pro", "deepseek-v4-pro-0813", "deepseek-v4-flash-0731", "hy4", "deepseek-v4.1-flash-1017",
		"hy4flash", "deepseek-v4.1-flash-1017-abliterated",
	}
	if len(draft.Models) != len(wantIDs) || len(draft.Provider.Routes) != 0 || len(draft.Provider.Agreements) != 0 {
		t.Fatalf("APmix catalog/authority = models:%d routes:%d agreements:%d; want 24 catalog identities, no executable routes or agreements", len(draft.Models), len(draft.Provider.Routes), len(draft.Provider.Agreements))
	}
	for _, id := range wantIDs {
		if _, ok := draft.Models[id]; !ok {
			t.Errorf("APmix dated Business catalog identity %q missing from embedded policy", id)
		}
	}
	for id, model := range draft.Models {
		if model.Eligible == nil || *model.Eligible {
			t.Errorf("APmix model %q is missing an explicit ineligible draft flag: %+v", id, model)
		}
		for _, source := range []string{"catalog", "terms"} {
			provenance := model.Provenance[source]
			if provenance.ObservedAt != "2026-11-23" || provenance.Source == "" {
				t.Errorf("APmix model %q lacks dated %q provenance: %+v", id, source, provenance)
			}
		}
		identity := model.Provenance["identity"]
		if identity.ObservedAt != "2026-11-23" || identity.Confidence != "high" || !strings.Contains(identity.Source, "https://business.apmix.ai/") || !strings.Contains(identity.Source, id) {
			t.Errorf("APmix model %q lacks exact dated Business-table identity evidence: %+v", id, identity)
		}
	}
	if !strings.Contains(draft.Research.ModelReason, "upstream provider") || !strings.Contains(draft.Research.ModelReason, "account-scoped availability") {
		t.Fatalf("APmix model rights gap is not retained: %+v", draft.Research)
	}
	if !strings.Contains(draft.Research.ModelReason, "20 resale models") || !strings.Contains(draft.Research.ModelReason, "24") || !strings.Contains(draft.Research.ModelReason, "claude-sonnet-5") || !strings.Contains(draft.Research.ModelReason, "claude-sonnet-4-6") {
		t.Fatalf("APmix page-count/docs model-ID mismatches are not preserved: %+v", draft.Research)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func assertProposedSupplierDenied(t *testing.T, snapshot PrivatePolicySnapshot) {
	t.Helper()
	if len(snapshot.Records) != 1 || snapshot.Records[0].Supplier != "nanogpt" {
		t.Fatalf("records = %#v, want one NanoGPT draft", snapshot.Records)
	}
	if snapshot.Records[0].Terms.Review.Status != policy.ReviewProposed {
		t.Fatalf("review status = %q, want proposed", snapshot.Records[0].Terms.Review.Status)
	}
	if len(snapshot.Routes) != 0 || len(snapshot.Agreements) != 0 {
		t.Fatalf("proposed supplier created executable routes or trust agreements: routes=%#v agreements=%#v", snapshot.Routes, snapshot.Agreements)
	}
}
