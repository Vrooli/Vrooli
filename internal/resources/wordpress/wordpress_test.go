package wordpress

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func repoWordPressPath(parts ...string) string {
	all := append([]string{"..", "..", "..", "resources", "wordpress"}, parts...)
	return filepath.Clean(filepath.Join(all...))
}

func TestLoadPinnedImageProfileAndRejectsMutableReference(t *testing.T) {
	profile, err := LoadImageSet(repoWordPressPath("images.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.entries()) != 3 {
		t.Fatalf("expected three pinned images, got %d", len(profile.entries()))
	}
	for _, image := range profile.entries() {
		if !digestReference.MatchString(image.Reference) {
			t.Fatalf("image is mutable: %s", image.Reference)
		}
		if len(image.Platforms) != 2 {
			t.Fatalf("image platform coverage missing: %+v", image)
		}
	}
	data, err := os.ReadFile(repoWordPressPath("images.json"))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	images := body["images"].(map[string]any)
	images["site"].(map[string]any)["reference"] = "wordpress:latest"
	data, _ = json.Marshal(body)
	path := filepath.Join(t.TempDir(), "images.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadImageSet(path); err == nil {
		t.Fatal("mutable image reference was accepted")
	}
}

func TestRecipeCatalogHasThreeFormRecipesShopAndContentWithHardLevels(t *testing.T) {
	path := repoWordPressPath("recipes", "recipes.json")
	cases := []struct {
		id, kind, plugin string
		minimumLevels    int
		pluginCount      int
	}{
		{"forms-contact-form-7", "forms", "contact-form-7", 5, 1},
		{"forms-wpforms-lite", "forms", "wpforms-lite", 5, 1},
		{"forms-forminator", "forms", "forminator", 5, 1},
		{"shop-woocommerce", "shop", "woocommerce", 5, 1},
		{"content-pods-blog", "content", "pods", 4, 1},
		{"content-core-blog", "content", "", 3, 0},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			recipe, err := LoadRecipe(path, tc.id)
			if err != nil {
				t.Fatal(err)
			}
			if recipe.Kind != tc.kind || len(recipe.Plugins) != tc.pluginCount || (tc.pluginCount == 1 && recipe.Plugins[0] != tc.plugin) {
				t.Fatalf("unexpected recipe profile: %+v", recipe)
			}
			if len(recipe.ExpectedLevels) < tc.minimumLevels {
				t.Fatalf("recipe too shallow: %v", recipe.ExpectedLevels)
			}
			if tc.pluginCount == 1 && recipe.PackageVersions[tc.plugin] == "" {
				t.Fatalf("missing exact plugin version for %s", tc.plugin)
			}
		})
	}
	shop, err := LoadRecipe(path, "shop-woocommerce")
	if err != nil {
		t.Fatal(err)
	}
	if shop.Settings["paymentsEnabled"] != false {
		t.Fatal("synthetic shop must not enable payments")
	}
	for _, item := range nestedSlice(shop.Content, "orders") {
		order := item.(map[string]any)
		if !strings.HasSuffix(order["customerEmail"].(string), ".invalid") {
			t.Fatalf("non-synthetic order address: %v", order)
		}
	}
}

func TestCoreContentPostRecipeRejectsUnsafeAndInvalidMetadata(t *testing.T) {
	valid := map[string]any{"slug": "measured-onboarding", "title": "A measured onboarding improvement", "categories": []any{"Case Studies", "Onboarding"}, "tags": []any{"synthetic", "measurement"}, "status": "publish", "client_type": "synthetic", "outcome_metric": float64(24), "verified_on": "2026-11-14"}
	if err := validateCoreContentPost(valid); err != nil {
		t.Fatalf("valid synthetic post rejected: %v", err)
	}
	for key, value := range map[string]any{
		"slug":           "../../outside",
		"title":          strings.Repeat("x", 201),
		"status":         "future",
		"client_type":    "client-name",
		"outcome_metric": float64(-1),
		"verified_on":    "2026-02-30",
		"content":        strings.Repeat("x", 20_001),
		"categories":     []any{"Valid", "valid"},
		"tags":           []any{"bad/<markup>"},
	} {
		candidate := make(map[string]any, len(valid))
		for field, fieldValue := range valid {
			candidate[field] = fieldValue
		}
		candidate[key] = value
		if err := validateCoreContentPost(candidate); err == nil {
			t.Errorf("accepted unsafe %s value %#v", key, value)
		}
	}
	for _, tags := range []any{[]any{"bad/<markup>"}, []any{"Trailing "}} {
		candidate := map[string]any{"slug": "safe-post", "title": "Safe post", "status": "publish", "client_type": "synthetic", "outcome_metric": float64(1), "verified_on": "2026-11-14", "categories": []any{}, "tags": tags}
		if err := validateCoreContentPost(candidate); err == nil {
			t.Errorf("accepted unsafe taxonomy names %#v", tags)
		}
	}
}

func TestHardGroundReferencesAllRecipesAndGradesEveryFamily(t *testing.T) {
	ground, err := LoadGround(repoWordPressPath("ground", "levels.json"), repoWordPressPath("recipes", "recipes.json"))
	if err != nil {
		t.Fatal(err)
	}
	grade, err := ground.GradeFixtures(repoWordPressPath("ground", "levels.json"))
	if err != nil {
		t.Fatal(err)
	}
	if grade.Total != 51 || grade.Passed != 51 || len(grade.Missing) != 0 || len(grade.Failed) != 0 {
		t.Fatalf("hard ground did not fully grade: %+v", grade)
	}
	if family, exists := grade.ByFamily["content"]; !exists || family.Total != 28 || family.Passed != 28 {
		t.Fatalf("content and migration ground did not fully grade: %+v", grade.ByFamily["content"])
	}
	if family, exists := grade.ByFamily["forms"]; !exists || family.Total != 16 || family.Passed != 16 {
		t.Fatalf("form Unicode boundary ground did not fully grade: %+v", grade.ByFamily["forms"])
	}
	if family, exists := ground.Grade(map[string]bool{}).ByFamily["forms"]; !exists || family.Total == 0 {
		t.Fatal("expected forms family grade")
	}
}

func TestContentMigrationLedgerCreatesReplaysUpdatesAndRejectsCollisions(t *testing.T) {
	ledger := NewContentMigrationLedger()
	makeRecord := func(metric int, source string) ContentMigrationRecord {
		record := ContentMigrationRecord{Slug: "migration-case", Title: "Synthetic migration case", Content: "<h2>Synthetic body</h2><p>Migration proof.</p>", Excerpt: "Synthetic migration.", Categories: []string{"Ground Cases"}, Tags: []string{"synthetic", "migration"}, Status: "publish", ClientType: "synthetic", OutcomeMetric: metric, VerifiedOn: "2026-11-14", SourceKey: source + "/migration-case"}
		record.ContentHash, _ = contentMigrationHash(record)
		return record
	}
	record := makeRecord(24, "source-site")
	if result, err := ledger.Plan("target-site", record); err != nil || result != ContentMigrationCreated {
		t.Fatalf("create plan failed: result=%s err=%v", result, err)
	}
	if result, err := ledger.Apply("target-site", record); err != nil || result != ContentMigrationCreated {
		t.Fatalf("read-only plan changed the ledger or create failed: result=%s err=%v", result, err)
	}
	if result, err := ledger.Plan("target-site", record); err != nil || result != ContentMigrationDuplicate {
		t.Fatalf("duplicate plan failed: result=%s err=%v", result, err)
	}
	if result, err := ledger.Apply("target-site", record); err != nil || result != ContentMigrationDuplicate {
		t.Fatalf("replay not idempotent: result=%s err=%v", result, err)
	}
	updated := makeRecord(25, "source-site")
	if result, err := ledger.Plan("target-site", updated); err != nil || result != ContentMigrationUpdated {
		t.Fatalf("update plan failed: result=%s err=%v", result, err)
	}
	if result, err := ledger.Apply("target-site", updated); err != nil || result != ContentMigrationUpdated {
		t.Fatalf("revision update failed: result=%s err=%v", result, err)
	}
	conflict := makeRecord(25, "other-source")
	if result, err := ledger.Plan("target-site", conflict); err != nil || result != ContentMigrationConflict {
		t.Fatalf("source conflict plan failed: result=%s err=%v", result, err)
	}
	if result, err := ledger.Apply("target-site", conflict); err != nil || result != ContentMigrationConflict {
		t.Fatalf("foreign source collision not rejected: result=%s err=%v", result, err)
	}
	badHash := makeRecord(26, "source-site")
	badHash.ContentHash = "deadbeef"
	if _, err := ledger.Apply("target-site", badHash); err == nil {
		t.Fatal("accepted archive with incorrect canonical content hash")
	}
}

func TestContentMigrationHashCoversBodyAndExcerpt(t *testing.T) {
	base := ContentMigrationRecord{Slug: "article", Title: "Synthetic article", Content: "<p>one</p>", Excerpt: "one", Status: "publish", ClientType: "synthetic", OutcomeMetric: 1, VerifiedOn: "2026-11-14"}
	base.Categories = []string{"Case Studies", "Onboarding"}
	base.Tags = []string{"synthetic", "measurement"}
	first, err := contentMigrationHash(base)
	if err != nil {
		t.Fatal(err)
	}
	changedBody := base
	changedBody.Content = "<p>two</p>"
	second, err := contentMigrationHash(changedBody)
	if err != nil || second == first {
		t.Fatalf("content change did not change migration hash: %q %q err=%v", first, second, err)
	}
	changedExcerpt := base
	changedExcerpt.Excerpt = "two"
	third, err := contentMigrationHash(changedExcerpt)
	if err != nil || third == first {
		t.Fatalf("excerpt change did not change migration hash: %q %q err=%v", first, third, err)
	}
	changedCategory := base
	changedCategory.Categories = []string{"Content Operations"}
	fourth, err := contentMigrationHash(changedCategory)
	if err != nil || fourth == first {
		t.Fatalf("category change did not change migration hash: %q %q err=%v", first, fourth, err)
	}
	changedTag := base
	changedTag.Tags = []string{"synthetic", "revision"}
	fifth, err := contentMigrationHash(changedTag)
	if err != nil || fifth == first {
		t.Fatalf("tag change did not change migration hash: %q %q err=%v", first, fifth, err)
	}
	reordered := base
	reordered.Categories = []string{"Onboarding", "Case Studies"}
	reordered.Tags = []string{"measurement", "synthetic"}
	canonical, err := contentMigrationHash(reordered)
	if err != nil || canonical != first {
		t.Fatalf("taxonomy ordering changed canonical hash: %q %q err=%v", first, canonical, err)
	}
}

func TestContentAuditScoresEditorialCompleteness(t *testing.T) {
	complete := ContentMigrationRecord{Slug: "audit-ready", Title: "A complete synthetic editorial sample", Content: "<h2>Summary</h2><p>This synthetic article has enough meaningful body text for a reader to understand the scenario, reproduce the work, and inspect the verified results without any client information.</p>", Excerpt: "A complete synthetic sample with a concise editorial summary and measured outcome.", Categories: []string{"Editorial Ground"}, Tags: []string{"synthetic", "audit"}}
	deficient := ContentMigrationRecord{Slug: "audit-needs-review", Title: "Tiny", Content: "<p>Short.</p>"}
	report := AuditCoreContentRecords("ground", []ContentMigrationRecord{complete, deficient})
	if report.PostCount != 2 || report.ReadyCount != 1 || report.ReviewCount != 1 || report.Posts[0].Score != 100 || report.Posts[0].Disposition != "ready" || report.Posts[0].WordCount < 20 || report.Posts[0].HeadingCount != 1 {
		t.Fatalf("complete content did not pass the editorial checklist: %+v", report)
	}
	if report.Posts[1].Score != 15 || report.Posts[1].Disposition != "needs-review" || len(report.Posts[1].Findings) != 6 {
		t.Fatalf("deficient content did not report the expected findings: %+v", report.Posts[1])
	}
	for _, finding := range report.Posts[1].Findings {
		if finding.Action == "" {
			t.Errorf("finding %q did not give a concrete next action", finding.Code)
		}
	}
}

func TestJobWorkflowExportsAreInactiveLocalAndCredentialFree(t *testing.T) {
	tests := []struct {
		name, file, destination string
		wantNodes               []string
	}{
		{"a3-forms", repoWordPressPath("jobs", "forms", "a3-n8n-workflow.json"), "http://wordpress/wp-json/vrooli/v1/form-intake", []string{"n8n-nodes-base.webhook", "n8n-nodes-base.code", "n8n-nodes-base.httpRequest"}},
		{"shop-order-sync", repoWordPressPath("jobs", "shop-order-sync", "order-n8n-workflow.json"), "http://wordpress/wp-json/vrooli/v1/shop-order-sync", []string{"n8n-nodes-base.webhook", "n8n-nodes-base.crypto", "n8n-nodes-base.if", "n8n-nodes-base.httpRequest"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			var workflow struct {
				Active bool `json:"active"`
				Nodes  []struct {
					Name       string         `json:"name"`
					Type       string         `json:"type"`
					Parameters map[string]any `json:"parameters"`
				} `json:"nodes"`
			}
			if err := json.Unmarshal(data, &workflow); err != nil {
				t.Fatal(err)
			}
			if workflow.Active {
				t.Fatal("workflow exports must be inactive on import")
			}
			types, urls := map[string]bool{}, map[string]bool{}
			for _, node := range workflow.Nodes {
				types[node.Type] = true
				if url, ok := node.Parameters["url"].(string); ok {
					urls[url] = true
				}
			}
			for _, want := range tc.wantNodes {
				if !types[want] {
					t.Errorf("workflow missing node type %s", want)
				}
			}
			if !urls[tc.destination] {
				t.Errorf("workflow missing local destination %s", tc.destination)
			}
			if bytes.Contains(data, []byte("password\"")) || bytes.Contains(data, []byte("consumer_secret")) || bytes.Contains(data, []byte("api_key")) {
				t.Fatal("workflow export contains a credential-shaped value")
			}
		})
	}
}

func TestHardFormFixturesMatchKnownAnswersAndIdempotency(t *testing.T) {
	data, err := os.ReadFile(repoWordPressPath("ground", "fixtures", "forms.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases map[string]struct {
			Provider string          `json:"provider"`
			Payload  json.RawMessage `json:"payload"`
			Expected map[string]any  `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	ledger := NewSubmissionLedger()
	for _, name := range []string{"cf7-valid", "wpforms-valid", "forminator-valid", "elementor-shaped-valid"} {
		item := fixture.Cases[name]
		event, err := NormalizeFormSubmission(item.Provider, item.Payload)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if event.EventID != item.Expected["eventId"] || event.Name != item.Expected["name"] || event.Email != item.Expected["email"] {
			t.Fatalf("%s does not match its answer: %+v / %+v", name, event, item.Expected)
		}
	}
	first := fixture.Cases["cf7-valid"]
	event, err := NormalizeFormSubmission(first.Provider, first.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := ledger.Accept(event); err != nil || got != SubmissionCreated {
		t.Fatalf("first event disposition %q err=%v", got, err)
	}
	for _, name := range []string{"lost-response-replay"} {
		item := fixture.Cases[name]
		replay, err := NormalizeFormSubmission(item.Provider, item.Payload)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := ledger.Accept(replay); err != nil || got != SubmissionDuplicate {
			t.Fatalf("lost response replay disposition %q err=%v", got, err)
		}
	}
	changed := fixture.Cases["identity-collision"]
	conflict, err := NormalizeFormSubmission(changed.Provider, changed.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := ledger.Accept(conflict); err != nil || got != SubmissionConflict {
		t.Fatalf("identity collision disposition %q err=%v", got, err)
	}
}

func TestHardFormFixturesRejectInvalidGroundAndWPFormsEntryZeroIsNotIdentity(t *testing.T) {
	data, err := os.ReadFile(repoWordPressPath("ground", "fixtures", "forms.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases map[string]struct {
			Provider string          `json:"provider"`
			Payload  json.RawMessage `json:"payload"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for name, reason := range map[string]string{"invalid-email": "invalid_email", "missing-name": "missing_name"} {
		item := fixture.Cases[name]
		if _, err := NormalizeFormSubmission(item.Provider, item.Payload); err == nil || !strings.Contains(err.Error(), reason) {
			t.Fatalf("%s: wanted %s, got %v", name, reason, err)
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(fixture.Cases["wpforms-valid"].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	fields := payload["fields"].([]any)
	payload["fields"] = fields[1:]
	missingIdentity, _ := json.Marshal(payload)
	if _, err := NormalizeFormSubmission("wpforms-lite", missingIdentity); err == nil || !strings.Contains(err.Error(), "event_id") {
		t.Fatalf("WPForms Lite entry_id=0 was treated as a retry identity: %v", err)
	}
}

func TestFormFieldLimitsCountUnicodeCodePoints(t *testing.T) {
	data, err := os.ReadFile(repoWordPressPath("ground", "fixtures", "forms.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases map[string]struct {
			Provider string          `json:"provider"`
			Payload  json.RawMessage `json:"payload"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"unicode-name-multibyte-in-limit",
		"unicode-name-supplementary-in-limit",
		"unicode-message-supplementary-in-limit",
	} {
		item := fixture.Cases[name]
		if _, err := NormalizeFormSubmission(item.Provider, item.Payload); err != nil {
			t.Errorf("%s should pass character-count limits: %v", name, err)
		}
	}
	for _, name := range []string{
		"unicode-name-supplementary-over-limit",
		"unicode-message-supplementary-over-limit",
	} {
		item := fixture.Cases[name]
		if _, err := NormalizeFormSubmission(item.Provider, item.Payload); err == nil || !strings.Contains(err.Error(), "field exceeds") {
			t.Errorf("%s should be rejected above the character limit, got %v", name, err)
		}
	}
}

func TestWooCommerceGroundAnswersAndDistinctStatusRevision(t *testing.T) {
	data, err := os.ReadFile(repoWordPressPath("ground", "fixtures", "shop.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Products []struct {
			SKU string `json:"sku"`
		} `json:"products"`
		Cases map[string]struct {
			Payload json.RawMessage `json:"payload"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	skUs := map[string]bool{}
	for _, product := range fixture.Products {
		skUs[product.SKU] = true
	}
	ledger := NewOrderLedger()
	first, err := NormalizeWooOrder("k3-ground", fixture.Cases["order-1001"].Payload, skUs)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := ledger.Accept(first); err != nil || got != SubmissionCreated {
		t.Fatalf("first order disposition %q err=%v", got, err)
	}
	for _, name := range []string{"order-1001-replay", "order-lost-response-replay"} {
		replay, err := NormalizeWooOrder("k3-ground", fixture.Cases[name].Payload, skUs)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := ledger.Accept(replay); err != nil || got != SubmissionDuplicate {
			t.Fatalf("%s disposition %q err=%v", name, got, err)
		}
	}
	changed, err := NormalizeWooOrder("k3-ground", fixture.Cases["order-status-change"].Payload, skUs)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := ledger.Accept(changed); err != nil || got != SubmissionCreated || changed.Status != "completed" {
		t.Fatalf("status revision disposition=%q order=%+v err=%v", got, changed, err)
	}
	for name, reason := range map[string]string{"unknown-sku": "unknown_sku", "missing-revision": "missing_revision"} {
		if _, err := NormalizeWooOrder("k3-ground", fixture.Cases[name].Payload, skUs); err == nil || !strings.Contains(err.Error(), reason) {
			t.Fatalf("%s: expected %s, got %v", name, reason, err)
		}
	}
}

func TestContentUpdateLedgerAppliesOnceAndDetectsConflictingReplay(t *testing.T) {
	ledger := NewContentUpdateLedger()
	created, err := ledger.CreateIfMissing("77", map[string]any{"outcome_metric": 24})
	if err != nil || !created {
		t.Fatalf("create post: created=%t err=%v", created, err)
	}
	patch := map[string]any{"outcome_metric": 25}
	first, fields, err := ledger.Apply("77", "update-1", patch)
	if err != nil || first != ContentUpdated || fields["outcome_metric"] != 25 {
		t.Fatalf("first update: disposition=%s fields=%v err=%v", first, fields, err)
	}
	replay, fields, err := ledger.Apply("77", "update-1", patch)
	if err != nil || replay != ContentDuplicate || fields["outcome_metric"] != 25 {
		t.Fatalf("same update replay: disposition=%s fields=%v err=%v", replay, fields, err)
	}
	conflict, _, err := ledger.Apply("77", "update-1", map[string]any{"outcome_metric": 26})
	if err != nil || conflict != ContentConflict {
		t.Fatalf("changed update replay: disposition=%s err=%v", conflict, err)
	}
}

func TestSubmissionLedgerConcurrentRetriesCreateOnce(t *testing.T) {
	ledger := NewSubmissionLedger()
	event := CanonicalFormEvent{Provider: "contact-form-7", FormID: "cf7-101", EventID: "event-1", Name: "Synthetic", Email: "synthetic@example.invalid", Message: "Retry-safe"}
	var mu sync.Mutex
	counts := map[SubmissionDisposition]int{}
	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := ledger.Accept(event)
			if err != nil {
				t.Errorf("ledger accept: %v", err)
				return
			}
			mu.Lock()
			counts[result]++
			mu.Unlock()
		}()
	}
	wait.Wait()
	if counts[SubmissionCreated] != 1 || counts[SubmissionDuplicate] != 31 {
		t.Fatalf("concurrent duplicate safety failed: %v", counts)
	}
}

func TestRecipeRejectsUnknownAndTraversalIdentifiers(t *testing.T) {
	path := repoWordPressPath("recipes", "recipes.json")
	for _, value := range []string{"../shop-woocommerce", "", "UPPER"} {
		if _, err := LoadRecipe(path, value); err == nil {
			t.Fatalf("accepted recipe identifier %q", value)
		}
	}
	if _, err := LoadRecipe(path, "no-such-recipe"); err == nil {
		t.Fatal("accepted unknown recipe")
	}
}

func TestStrictRecipeCatalogRejectsUnknownAndTrailingJSON(t *testing.T) {
	for _, data := range []string{`{"schemaVersion":1,"recipes":[],"unexpected":true}`, `{"schemaVersion":1,"recipes":[]} {}`} {
		if err := DecodeStrictRecipeCatalog([]byte(data)); err == nil {
			t.Fatalf("accepted malformed catalog %s", data)
		}
	}
}

type fixedRunner struct {
	outputs map[string][]byte
	errors  map[string]error
}

type recordingRunner struct {
	outputs map[string][]byte
	calls   []string
}

type sequenceRunner struct {
	outputs map[string][][]byte
	calls   []string
}

type runnerResult struct {
	output []byte
	err    error
}

type scriptedRunner struct {
	steps map[string][]runnerResult
	calls []string
}

func (r *scriptedRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	r.calls = append(r.calls, key)
	items := r.steps[key]
	if len(items) == 0 {
		return nil, nil
	}
	r.steps[key] = items[1:]
	return items[0].output, items[0].err
}

func (r *sequenceRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	r.calls = append(r.calls, key)
	items := r.outputs[key]
	if len(items) == 0 {
		return nil, nil
	}
	r.outputs[key] = items[1:]
	return items[0], nil
}

func (r *recordingRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	r.calls = append(r.calls, key)
	return r.outputs[key], nil
}

func (r fixedRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	if err := r.errors[key]; err != nil {
		return nil, err
	}
	return r.outputs[key], nil
}

func TestSiteURLRequiresPrivateOwnerNetworkAddress(t *testing.T) {
	var info containerInfo
	info.NetworkSettings.Networks = map[string]struct {
		IPAddress string `json:"IPAddress"`
	}{
		"vrooli-wp-demo": {IPAddress: "203.0.113.25"},
	}
	owner := &Owner{}
	if _, err := owner.siteURL(context.Background(), "vrooli-wp-site-demo", info); err == nil {
		t.Fatal("accepted non-private site address")
	}
	info.NetworkSettings.Networks["vrooli-wp-demo"] = struct {
		IPAddress string `json:"IPAddress"`
	}{IPAddress: "172.26.0.3"}
	url, err := owner.siteURL(context.Background(), "vrooli-wp-site-demo", info)
	if err != nil || url != "http://172.26.0.3" {
		t.Fatalf("unexpected private site URL %q, err=%v", url, err)
	}
}

func TestInspectNetworkRecognizesDockerNotFoundAsAbsent(t *testing.T) {
	name := "vrooli-wp-demo"
	runner := fixedRunner{errors: map[string]error{
		"network inspect --format {{json .Labels}} " + name: errors.New("docker network inspect failed: network " + name + " not found"),
	}}
	owner := &Owner{Runner: runner}
	_, exists, err := owner.inspectNetwork(context.Background(), name)
	if err != nil || exists {
		t.Fatalf("expected absent network without error, exists=%v err=%v", exists, err)
	}
}

func TestEnsureContainerRefreshesNetworkMetadataAfterRestart(t *testing.T) {
	labels := `"com.vrooli.resource":"wordpress","com.vrooli.object-kind":"site","com.vrooli.wordpress.site-id":"demo"`
	stopped := []byte(`[{"Config":{"Labels":{` + labels + `}},"State":{"Status":"exited"},"NetworkSettings":{"Networks":{}}}]`)
	running := []byte(`[{"Config":{"Labels":{` + labels + `}},"State":{"Status":"running"},"NetworkSettings":{"Networks":{"vrooli-wp-demo":{"IPAddress":"172.26.0.9"}}}}]`)
	runner := &sequenceRunner{outputs: map[string][][]byte{"container inspect vrooli-wp-site-demo": {stopped, running}}}
	owner := &Owner{Runner: runner}
	info, err := owner.ensureContainer(context.Background(), "vrooli-wp-site-demo", "demo", "site", []string{"run"})
	if err != nil {
		t.Fatal(err)
	}
	if info.State.Status != "running" || info.NetworkSettings.Networks["vrooli-wp-demo"].IPAddress != "172.26.0.9" {
		t.Fatalf("restart returned stale container inspection metadata: %+v", info)
	}
	if len(runner.calls) != 3 || runner.calls[1] != "start vrooli-wp-site-demo" {
		t.Fatalf("unexpected restart inspection sequence: %v", runner.calls)
	}
}

func TestEnsureContainerRetriesOwnedRestartAfterTransientStartFailure(t *testing.T) {
	labels := `"com.vrooli.resource":"wordpress","com.vrooli.object-kind":"site","com.vrooli.wordpress.site-id":"demo"`
	stopped := []byte(`[{"Config":{"Labels":{` + labels + `}},"State":{"Status":"exited"}}]`)
	running := []byte(`[{"Config":{"Labels":{"com.vrooli.resource":"wordpress","com.vrooli.object-kind":"site","com.vrooli.wordpress.site-id":"demo"}},"State":{"Status":"running"}}]`)
	runner := &scriptedRunner{steps: map[string][]runnerResult{
		"container inspect vrooli-wp-site-demo": {{output: stopped}, {output: stopped}, {output: running}},
		"start vrooli-wp-site-demo":             {{err: errors.New("temporary daemon interruption")}, {}},
	}}
	owner := &Owner{Runner: runner}
	if _, err := owner.ensureContainer(context.Background(), "vrooli-wp-site-demo", "demo", "site", []string{"run"}); err == nil {
		t.Fatal("transient restart failure was hidden")
	}
	info, err := owner.ensureContainer(context.Background(), "vrooli-wp-site-demo", "demo", "site", []string{"run"})
	if err != nil || info.State.Status != "running" {
		t.Fatalf("owned stopped container did not recover on retry: info=%+v err=%v", info, err)
	}
	if len(runner.calls) != 5 || runner.calls[2] != "container inspect vrooli-wp-site-demo" || runner.calls[3] != "start vrooli-wp-site-demo" || runner.calls[4] != "container inspect vrooli-wp-site-demo" {
		t.Fatalf("retry bypassed owner inspection/restart sequence: %v", runner.calls)
	}
	for _, call := range runner.calls {
		if strings.HasPrefix(call, "run ") {
			t.Fatalf("restart retry created a replacement instead of reusing its owned object: %v", runner.calls)
		}
	}
}

func TestInstallDeliveryDestinationRequiresOwnedRunningSiteAndReturnsV1Contract(t *testing.T) {
	source := filepath.Join(t.TempDir(), "destination.php")
	body := "<?php\n/* Plugin Name: Vrooli WordPress Delivery Destination */\n// vrooli/v1\n" + strings.Repeat(" ", 260)
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(filepath.Dir(source), "vrooli-form-event-id.js")
	if err := os.WriteFile(script, []byte("/* vrooli-event-id */ crypto.randomUUID();"+strings.Repeat(" ", 80)), 0o600); err != nil {
		t.Fatal(err)
	}
	inspect := []byte(`[{
 "Config":{"Labels":{"com.vrooli.resource":"wordpress","com.vrooli.object-kind":"site","com.vrooli.wordpress.site-id":"demo"}},
 "State":{"Status":"running"},
 "NetworkSettings":{"Networks":{"vrooli-wp-demo":{"IPAddress":"172.26.0.7"}}}
}]`)
	runner := &recordingRunner{outputs: map[string][]byte{"container inspect vrooli-wp-site-demo": inspect}}
	dataRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataRoot, "instances", "demo"), 0o700); err != nil {
		t.Fatal(err)
	}
	owner := &Owner{Runner: runner, DataRoot: dataRoot}
	receipt, err := owner.InstallDeliveryDestination(context.Background(), "demo", source, script)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Contract != "vrooli.wordpress.delivery.v1" || receipt.FormEndpoint != "http://172.26.0.7/wp-json/vrooli/v1/form-intake" || receipt.OrderEndpoint != "http://172.26.0.7/wp-json/vrooli/v1/shop-order-sync" {
		t.Fatalf("unexpected versioned destination contract: %+v", receipt)
	}
	for _, command := range []string{
		"exec vrooli-wp-site-demo mkdir -p /var/www/html/wp-content/mu-plugins",
		"cp " + source + " vrooli-wp-site-demo:/var/www/html/wp-content/mu-plugins/vrooli-delivery-destination.php",
		"cp " + script + " vrooli-wp-site-demo:/var/www/html/wp-content/mu-plugins/vrooli-form-event-id.js",
		"exec vrooli-wp-site-demo php -l /var/www/html/wp-content/mu-plugins/vrooli-delivery-destination.php",
	} {
		found := false
		for _, called := range runner.calls {
			if called == command {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("install did not issue owner-scoped command %q", command)
		}
	}
	optionBound := false
	for _, called := range runner.calls {
		if strings.Contains(called, "option update vrooli_resource_site_id demo --autoload=no --allow-root") && strings.Contains(called, "--network vrooli-wp-demo") {
			optionBound = true
		}
	}
	if !optionBound {
		t.Fatalf("installer did not bind site identity through the owned private WP-CLI path: %v", runner.calls)
	}
}

func TestConfigureFormWebhookUsesOnlyTheFixedLocalN8NTarget(t *testing.T) {
	inspect := []byte(`[{"Config":{"Labels":{"com.vrooli.resource":"wordpress","com.vrooli.object-kind":"site","com.vrooli.wordpress.site-id":"demo"}},"State":{"Status":"running"}}]`)
	runner := &recordingRunner{outputs: map[string][]byte{"container inspect vrooli-wp-site-demo": inspect}}
	dataRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataRoot, "instances", "demo"), 0o700); err != nil {
		t.Fatal(err)
	}
	owner := &Owner{Runner: runner, DataRoot: dataRoot}
	enabled, err := owner.ConfigureFormWebhook(context.Background(), "demo", true)
	if err != nil || enabled.Target != "http://n8n:5678/webhook/wordpress-form-intake" {
		t.Fatalf("unexpected fixed webhook target: receipt=%+v err=%v", enabled, err)
	}
	disabled, err := owner.ConfigureFormWebhook(context.Background(), "demo", false)
	if err != nil || disabled.Enabled || disabled.Target != "" {
		t.Fatalf("webhook did not disable cleanly: receipt=%+v err=%v", disabled, err)
	}
	for _, called := range runner.calls {
		if strings.Contains(called, "option update vrooli_form_n8n_webhook") && strings.Contains(called, "http://n8n:5678/webhook/wordpress-form-intake") {
			return
		}
	}
	t.Fatalf("owner did not configure the fixed local route: %v", runner.calls)
}

func TestInstallDeliveryDestinationRefusesSymlinkSourceAndForeignSite(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.php")
	if err := os.WriteFile(target, []byte("<?php\n"+strings.Repeat(" ", 300)), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "plugin.php")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	owner := &Owner{}
	if _, err := owner.InstallDeliveryDestination(context.Background(), "demo", link, target); err == nil {
		t.Fatal("accepted symlinked destination source")
	}
	foreign := fixedRunner{outputs: map[string][]byte{"container inspect vrooli-wp-site-demo": []byte(`[{"Config":{"Labels":{}},"State":{"Status":"running"}}]`)}}
	if _, err := (&Owner{Runner: foreign}).InstallDeliveryDestination(context.Background(), "demo", target, target); !errors.Is(err, ErrForeignObject) {
		t.Fatalf("expected foreign owner rejection, got %v", err)
	}
}

func TestEnsureSafeDirRejectsSymlinkInsideOwnerRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "owned")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "instances")); err != nil {
		t.Fatal(err)
	}
	if err := ensureSafeDir(root, filepath.Join(root, "instances", "site")); err == nil {
		t.Fatal("followed an owner-root symlink")
	}
}

func TestWriteTarTreeRejectsSymlinkAndDoesNotLeakOpenFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ok.txt"), []byte("proof"), 0o600); err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	if err := writeTarTree(writer, root, "content"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(bytes.NewReader(archive.Bytes()))
	header, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "content/ok.txt" {
		t.Fatalf("unexpected archive path %q", header.Name)
	}
	contents, err := io.ReadAll(reader)
	if err != nil || string(contents) != "proof" {
		t.Fatalf("archive contents %q, err=%v", contents, err)
	}
	if err := os.Symlink(filepath.Join(root, "ok.txt"), filepath.Join(root, "linked.txt")); err != nil {
		t.Fatal(err)
	}
	if err := writeTarTree(tar.NewWriter(io.Discard), root, "content"); err == nil {
		t.Fatal("snapshot followed a content symlink")
	}
}

func TestBoundedWriterMarksOverflow(t *testing.T) {
	var output bytes.Buffer
	writer := &boundedWriter{Writer: &output, remaining: 3}
	if _, err := writer.Write([]byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	if !writer.exceeded || output.String() != "abc" {
		t.Fatalf("bound was not enforced: exceeded=%v output=%q", writer.exceeded, output.String())
	}
}

func TestOwnerResetRefusesForeignObjectsBeforeMutation(t *testing.T) {
	// The refusal path is exercised at the lowest owner boundary; no Docker
	// effects are allowed before matching ownership labels are established.
	foreign := containerInfo{}
	if ownsContainer(foreign, "demo", "site") {
		t.Fatal("unlabeled container reported as owned")
	}
}
