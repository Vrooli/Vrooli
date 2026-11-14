package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

var validHTTPFixtures = []handlers.HTTPMockFixture{
	{Host: "CRM.SYNTHETIC.INVALID", Method: "post", Path: "/v1/contacts/42", Status: 201, ContentType: "application/json", Body: `{"id":"contact_synthetic_42","email":"person@example.invalid","status":"created"}`},
	{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/contacts", Query: "sort=created&limit=2", Status: 200, ContentType: "application/json", Body: `[{"id":"contact_synthetic_42"}]`},
	{Host: "sheets.synthetic.invalid", Method: "GET", Path: "/v1/rows", Status: 200, ContentType: "application/json", Body: `[{"row":1,"value":"synthetic"}]`},
}

func syntheticAPIOperationProfile() handlers.HTTPMockProfile {
	return handlers.HTTPMockProfile{SchemaVersion: 1, Operations: []handlers.HTTPMockOperation{
		{
			Selector: handlers.HTTPMockOperationSelector{NodeType: "n8n-nodes-base.googleSheets", Resource: "sheet", Operation: "append"},
			Route:    handlers.HTTPMockFixture{Host: "sheets.synthetic.invalid", Method: http.MethodPost, Path: "/v1/rows", Query: "spreadsheetId=synthetic-sheet", Status: 201, ContentType: "application/json", Body: `{"rowId":"row_synthetic_1"}`},
		},
		{
			Selector: handlers.HTTPMockOperationSelector{NodeType: "n8n-nodes-base.gmail", Resource: "message", Operation: "send"},
			Route:    handlers.HTTPMockFixture{Host: "gmail.synthetic.invalid", Method: http.MethodPost, Path: "/v1/messages", Status: 202, ContentType: "application/json", Body: `{"messageId":"message_synthetic_1","status":"queued"}`},
		},
	}}
}

type n8nHTTPMockOwnerFake struct {
	receipt                  handlers.N8NHTTPMockReceipt
	startErr                 error
	stopErr                  error
	starts                   int
	stops                    int
	sandboxID                uuid.UUID
	name, version, runtimeID string
	fixtures                 []handlers.HTTPMockFixture
}

func (f *n8nHTTPMockOwnerFake) StartHTTPMock(_ context.Context, sandboxID uuid.UUID, name, version string, fixtures []handlers.HTTPMockFixture) (handlers.N8NHTTPMockReceipt, error) {
	f.starts++
	f.sandboxID, f.name, f.version = sandboxID, name, version
	f.fixtures = append([]handlers.HTTPMockFixture(nil), fixtures...)
	return f.receipt, f.startErr
}

func (f *n8nHTTPMockOwnerFake) StopHTTPMock(_ context.Context, sandboxID uuid.UUID, runtimeID string) error {
	f.stops++
	f.sandboxID, f.runtimeID = sandboxID, runtimeID
	return f.stopErr
}

func TestValidateHTTPMockFixturesRequiresBoundedExactSyntheticRoutes(t *testing.T) {
	fixtures := append([]handlers.HTTPMockFixture(nil), validHTTPFixtures...)
	if err := handlers.ValidateHTTPMockFixtures(fixtures); err != nil {
		t.Fatal(err)
	}
	if fixtures[0].Host != "crm.synthetic.invalid" || fixtures[0].Method != "POST" || fixtures[0].ContentType != "application/json" {
		t.Fatalf("fixture normalization = %+v", fixtures[0])
	}
	invalid := []struct {
		name     string
		fixtures []handlers.HTTPMockFixture
	}{
		{"unsupported method", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "TRACE", Path: "/v1/x", Status: 200, Body: `{}`}}},
		{"external host", []handlers.HTTPMockFixture{{Host: "crm.example.com", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`}}},
		{"URL host", []handlers.HTTPMockFixture{{Host: "https://crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`}}},
		{"arbitrary port", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid:9999", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`}}},
		{"query path", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x?q=1", Status: 200, Body: `{}`}}},
		{"duplicate query values", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Query: "id=1&id=2", Status: 200, Body: `{}`}}},
		{"invalid query encoding", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Query: "id=%xx", Status: 200, Body: `{}`}}},
		{"oversized query", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Query: "q=" + strings.Repeat("x", 2050), Status: 200, Body: `{}`}}},
		{"credential query key", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Query: "access_token=synthetic", Status: 200, Body: `{}`}}},
		{"credential-shaped query value", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Query: "q=Bearer%20abcdefghijklmnop", Status: 200, Body: `{}`}}},
		{"real email query value", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Query: "email=real.person%40example.com", Status: 200, Body: `{}`}}},
		{"wildcard path", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/{id}", Status: 200, Body: `{}`}}},
		{"traversal path", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/../admin", Status: 200, Body: `{}`}}},
		{"redirect", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 302, Body: `{}`}}},
		{"invalid JSON", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{broken`}}},
		{"secret field", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{"access_token":"synthetic"}`}}},
		{"camel case secret field", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{"accessToken":"synthetic"}`}}},
		{"secret XML tag", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, ContentType: "application/xml", Body: `<response><password>synthetic</password></response>`}}},
		{"credential shaped text", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, ContentType: "text/plain", Body: `Authorization: Bearer abcdefghijklmnop`}}},
		{"real-looking email", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{"email":"real.person@example.com"}`}}},
		{"duplicate", []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`}, {Host: "CRM.synthetic.invalid", Method: "get", Path: "/v1/x", Status: 201, Body: `{}`}}},
		{"too many", make([]handlers.HTTPMockFixture, 129)},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if err := handlers.ValidateHTTPMockFixtures(test.fixtures); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
	large := []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/large", Status: 200, ContentType: "text/plain", Body: strings.Repeat("x", 64<<10+1)}}
	if err := handlers.ValidateHTTPMockFixtures(large); err == nil {
		t.Fatal("expected per-response cap")
	}
	aggregate := make([]handlers.HTTPMockFixture, 17)
	for i := range aggregate {
		aggregate[i] = handlers.HTTPMockFixture{Host: "crm.synthetic.invalid", Method: http.MethodGet, Path: fmt.Sprintf("/rows/%d", i), Status: 200, ContentType: "text/plain", Body: strings.Repeat("x", 64<<10-1)}
	}
	if err := handlers.ValidateHTTPMockFixtures(aggregate); err == nil {
		t.Fatal("expected aggregate response cap")
	}
}

func TestNormalizeHTTPMockProfileIsDeterministicAndRejectsAmbiguousSelectorsAndRoutes(t *testing.T) {
	profile := syntheticAPIOperationProfile()
	_, hash, err := handlers.NormalizeHTTPMockProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	reordered := profile
	reordered.Operations = append([]handlers.HTTPMockOperation(nil), profile.Operations...)
	reordered.Operations[0], reordered.Operations[1] = reordered.Operations[1], reordered.Operations[0]
	_, reorderedHash, err := handlers.NormalizeHTTPMockProfile(reordered)
	if err != nil || hash != reorderedHash {
		t.Fatalf("profile hash must be stable regardless of declaration ordering: hashes=%q/%q err=%v", hash, reorderedHash, err)
	}
	duplicateSelector := profile
	duplicateSelector.Operations = append([]handlers.HTTPMockOperation(nil), profile.Operations...)
	duplicateSelector.Operations[1].Selector = duplicateSelector.Operations[0].Selector
	if _, _, err := handlers.NormalizeHTTPMockProfile(duplicateSelector); err == nil {
		t.Fatal("expected duplicate selector rejection")
	}
	duplicateRoute := profile
	duplicateRoute.Operations = append([]handlers.HTTPMockOperation(nil), profile.Operations...)
	duplicateRoute.Operations[1].Route = duplicateRoute.Operations[0].Route
	if _, _, err := handlers.NormalizeHTTPMockProfile(duplicateRoute); err == nil {
		t.Fatal("expected duplicate exact-route rejection")
	}
	unsafeSelector := profile
	unsafeSelector.Operations = append([]handlers.HTTPMockOperation(nil), profile.Operations...)
	unsafeSelector.Operations[0].Selector.Operation = "send;proxy"
	if _, _, err := handlers.NormalizeHTTPMockProfile(unsafeSelector); err == nil {
		t.Fatal("expected unsafe selector rejection")
	}
	unsafeRoute := profile
	unsafeRoute.Operations = append([]handlers.HTTPMockOperation(nil), profile.Operations...)
	unsafeRoute.Operations[0].Route.Host = "api.example.com"
	if _, _, err := handlers.NormalizeHTTPMockProfile(unsafeRoute); err == nil {
		t.Fatal("expected non-synthetic route rejection")
	}
}

func TestNormalizeHTTPMockProfileSupportsDisjointInputSelectorsOnOneNativeOperation(t *testing.T) {
	profile := handlers.HTTPMockProfile{SchemaVersion: 1, Operations: []handlers.HTTPMockOperation{
		{Selector: handlers.HTTPMockOperationSelector{NodeType: "n8n-nodes-base.googleSheets", Resource: "sheet", Operation: "read", InputMatch: map[string]any{"sheetName": "products"}}, Route: handlers.HTTPMockFixture{Host: "sheets.synthetic.invalid", Method: http.MethodGet, Path: "/v1/spreadsheets", Query: "sheetName=products", Status: http.StatusOK, ContentType: "application/json", Body: `[]`}},
		{Selector: handlers.HTTPMockOperationSelector{NodeType: "n8n-nodes-base.googleSheets", Resource: "sheet", Operation: "read", InputMatch: map[string]any{"sheetName": "recipients"}}, Route: handlers.HTTPMockFixture{Host: "sheets.synthetic.invalid", Method: http.MethodGet, Path: "/v1/spreadsheets", Query: "sheetName=recipients", Status: http.StatusOK, ContentType: "application/json", Body: `[]`}},
	}}
	normalized, _, err := handlers.NormalizeHTTPMockProfile(profile)
	if err != nil || len(normalized.Operations) != 2 {
		t.Fatalf("disjoint exact selector routes must normalize: operations=%+v err=%v", normalized.Operations, err)
	}
	overlap := profile
	overlap.Operations = append([]handlers.HTTPMockOperation(nil), profile.Operations...)
	overlap.Operations[1].Selector.InputMatch = map[string]any{"sheetName": "products"}
	overlap.Operations[1].Route.Query = "sheetName=products"
	if _, _, err := handlers.NormalizeHTTPMockProfile(overlap); err == nil {
		t.Fatal("overlapping conditions for the same generic native operation must be rejected")
	}
}

func TestN8NHTTPMockProviderRequiresAttestedExactAliasesAndStopsThroughOwner(t *testing.T) {
	sandboxID := uuid.New()
	owner := &n8nHTTPMockOwnerFake{receipt: handlers.N8NHTTPMockReceipt{
		SandboxID: sandboxID.String(), CompanionRuntimeID: "sandbox.http.crm", NetworkAliases: []string{"crm.synthetic.invalid", "sheets.synthetic.invalid"},
		Port: 18080, Status: "attached", NoEgressVerified: true,
	}}
	provider := handlers.N8NHTTPMockServiceProvider{Owner: owner}
	spec := handlers.CompanionServiceSpec{Name: "workflow-apis", Kind: "http-mock", Version: "1.0.0", HTTPFixtures: append([]handlers.HTTPMockFixture(nil), validHTTPFixtures...)}
	runtime, err := provider.Start(context.Background(), sandboxID, spec)
	if err != nil {
		t.Fatal(err)
	}
	if owner.starts != 1 || owner.sandboxID != sandboxID || owner.name != spec.Name || owner.version != spec.Version || len(owner.fixtures) != 3 {
		t.Fatalf("owner start did not receive normalized fixture contract: %+v", owner)
	}
	if runtime.ID != "sandbox.http.crm" || runtime.Addresses["crm.synthetic.invalid"] != "http://crm.synthetic.invalid:18080" || runtime.Addresses["sheets.synthetic.invalid"] != "http://sheets.synthetic.invalid:18080" {
		t.Fatalf("runtime address map = %+v", runtime)
	}
	state := handlers.CompanionServiceState{Spec: spec, RuntimeID: runtime.ID}
	if err := provider.Stop(context.Background(), sandboxID, state); err != nil {
		t.Fatal(err)
	}
	if owner.stops != 1 || owner.sandboxID != sandboxID || owner.runtimeID != runtime.ID {
		t.Fatalf("stop owner args mismatch: %+v", owner)
	}
}

func TestN8NHTTPMockProviderFailsClosedAndCleansUpInvalidReceipt(t *testing.T) {
	sandboxID := uuid.New()
	owner := &n8nHTTPMockOwnerFake{receipt: handlers.N8NHTTPMockReceipt{
		SandboxID: sandboxID.String(), CompanionRuntimeID: "sandbox.http.crm", NetworkAliases: []string{"crm.synthetic.invalid", "public.example.net"},
		Port: 18080, Status: "attached", NoEgressVerified: true,
	}}
	provider := handlers.N8NHTTPMockServiceProvider{Owner: owner}
	spec := handlers.CompanionServiceSpec{Name: "workflow-apis", Kind: "http-mock", Version: "1.0.0", HTTPFixtures: validHTTPFixtures}
	if _, err := provider.Start(context.Background(), sandboxID, spec); err == nil {
		t.Fatal("expected unexpected host alias to be rejected")
	}
	if owner.stops != 1 || owner.runtimeID != "sandbox.http.crm" {
		t.Fatalf("invalid receipt cleanup mismatch: %+v", owner)
	}
	owner.receipt.NetworkAliases = []string{"crm.synthetic.invalid", "sheets.synthetic.invalid"}
	owner.receipt.NoEgressVerified = false
	if _, err := provider.Start(context.Background(), sandboxID, spec); err == nil {
		t.Fatal("expected missing no-egress attestation to be rejected")
	}
	owner.receipt.NoEgressVerified = true
	owner.receipt.Port = 18081
	if _, err := provider.Start(context.Background(), sandboxID, spec); err == nil {
		t.Fatal("expected non-fixed HTTP mock port to be rejected")
	}
	if owner.stops != 3 {
		t.Fatalf("invalid HTTP mock receipts were not cleaned up through owner: stops=%d", owner.stops)
	}
}

func TestN8NHTTPMockCLISendsFixtureConfigOnStdin(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "fake-vrooli")
	inputFile, argsFile := filepath.Join(root, "input"), filepath.Join(root, "args")
	script := "#!/bin/sh\ncat > '" + inputFile + "'\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\nprintf '%s\\n' '{\"sandboxId\":\"" + uuid.NewString() + "\",\"companionRuntimeId\":\"sandbox.http.crm\",\"networkAliases\":[\"crm.synthetic.invalid\",\"sheets.synthetic.invalid\"],\"port\":18080,\"status\":\"attached\",\"noEgressVerified\":true}'\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cli := handlers.N8NHTTPMockCLI{Binary: binary}
	_, err := cli.StartHTTPMock(context.Background(), uuid.New(), "workflow-apis", "1.0.0", validHTTPFixtures)
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(input), `"host":"crm.synthetic.invalid"`) || !strings.Contains(string(input), `"body":"{\"id\":`) {
		t.Fatalf("fixture config missing from stdin payload: %s", input)
	}
	var ownerConfig struct {
		Fixtures []handlers.HTTPMockFixture `json:"fixtures"`
	}
	if err := json.Unmarshal(input, &ownerConfig); err != nil {
		t.Fatalf("decode owner fixture config: %v", err)
	}
	queryFixtureFound := false
	for _, fixture := range ownerConfig.Fixtures {
		if fixture.Path == "/v1/contacts" {
			queryFixtureFound = fixture.Query == "limit=2&sort=created"
		}
	}
	if !queryFixtureFound {
		t.Fatalf("canonical query was not preserved across owner stdin: %+v", ownerConfig.Fixtures)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(args), "contact_synthetic") || !strings.Contains(string(args), "companion-start-http-mock") || !strings.Contains(string(args), "--config-stdin") {
		t.Fatalf("CLI args contain payload or miss fixed operation: %s", args)
	}
	if err := cli.StopHTTPMock(context.Background(), uuid.New(), "sandbox.http.crm"); err != nil {
		t.Fatal(err)
	}
	args, err = os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "runtime\ncompanion-stop-http-mock") || !strings.Contains(string(args), "--companion-runtime-id\nsandbox.http.crm") {
		t.Fatalf("HTTP mock stop must use the typed owner lifecycle: %s", args)
	}
}

func TestHTTPMockDeclarationAPIStoresExactFixturesAndSandboxDeleteCleansOwner(t *testing.T) {
	store := newCompanionStore()
	owner := &n8nHTTPMockOwnerFake{receipt: handlers.N8NHTTPMockReceipt{
		SandboxID: store.sandbox.ID.String(), CompanionRuntimeID: "sandbox.http.crm", NetworkAliases: []string{"crm.synthetic.invalid", "sheets.synthetic.invalid"},
		Port: 18080, Status: "attached", NoEgressVerified: true,
	}}
	provider := handlers.N8NHTTPMockServiceProvider{Owner: owner}
	svc := &sandboxiface.FakeService{DeleteFn: func(context.Context, uuid.UUID) error { return nil }}
	live := newLive(t, svc, withCompanionStore(store), withCompanionProvider(handlers.N8NCompanionServiceRouter{HTTPMock: provider}))
	decl := struct {
		Services []handlers.CompanionServiceSpec `json:"services"`
	}{Services: []handlers.CompanionServiceSpec{{Name: "workflow-apis", Kind: "http-mock", Version: "1.0.0", HTTPFixtures: validHTTPFixtures}}}
	encoded, err := json.Marshal(decl)
	if err != nil {
		t.Fatal(err)
	}
	resp, body := live.DoJSON(t, http.MethodPut, sandboxesPath(store.sandbox.ID, "/companion-services"), string(encoded))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("declare status=%d body=%s", resp.StatusCode, body)
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(store.sandbox.ID, "/companion-services/workflow-apis/start"), "{}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start status=%d body=%s", resp.StatusCode, body)
	}
	resp, body = live.DoJSON(t, http.MethodDelete, sandboxesPath(store.sandbox.ID, ""), "")
	if resp.StatusCode != http.StatusNoContent || owner.stops != 1 {
		t.Fatalf("delete status=%d stops=%d body=%s", resp.StatusCode, owner.stops, body)
	}
}

type routeHTTPMockOwner struct {
	sandboxID uuid.UUID
	server    *httptest.Server
	starts    int
	stops     int
}

func (o *routeHTTPMockOwner) StartHTTPMock(_ context.Context, sandboxID uuid.UUID, name, version string, fixtures []handlers.HTTPMockFixture) (handlers.N8NHTTPMockReceipt, error) {
	o.starts++
	o.sandboxID = sandboxID
	if err := handlers.ValidateHTTPMockFixtures(fixtures); err != nil {
		return handlers.N8NHTTPMockReceipt{}, err
	}
	routes := make(map[string]handlers.HTTPMockFixture, len(fixtures))
	aliasSet := map[string]struct{}{}
	for _, fixture := range fixtures {
		routes[fixture.Method+" "+fixture.Host+" "+fixture.Path+"?"+fixture.Query] = fixture
		aliasSet[fixture.Host] = struct{}{}
	}
	aliases := make([]string, 0, len(aliasSet))
	for alias := range aliasSet {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	o.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + strings.TrimSuffix(r.Host, ":18080") + " " + r.URL.Path + "?" + r.URL.Query().Encode()
		fixture, ok := routes[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", fixture.ContentType)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(fixture.Status)
		_, _ = io.WriteString(w, fixture.Body)
	}))
	return handlers.N8NHTTPMockReceipt{
		SandboxID: sandboxID.String(), CompanionRuntimeID: "sandbox.http.route-test",
		NetworkAliases: aliases, Port: 18080,
		Status: "attached", NoEgressVerified: true,
	}, nil
}

func TestHTTPMockProfileRoutesSelectorsToExactSyntheticResponsesThroughLifecycle(t *testing.T) {
	store := newCompanionStore()
	owner := &routeHTTPMockOwner{}
	provider := handlers.N8NHTTPMockServiceProvider{Owner: owner}
	live := newLive(t, &sandboxiface.FakeService{}, withCompanionStore(store), withCompanionProvider(handlers.N8NCompanionServiceRouter{HTTPMock: provider}))
	profile := syntheticAPIOperationProfile()
	invalidProfile := profile
	invalidProfile.Operations = append([]handlers.HTTPMockOperation(nil), profile.Operations...)
	invalidProfile.Operations[1].Selector = invalidProfile.Operations[0].Selector
	invalidDeclaration, err := json.Marshal(struct {
		Services []handlers.CompanionServiceSpec `json:"services"`
	}{Services: []handlers.CompanionServiceSpec{{Name: "public-apis", Kind: "http-mock", Version: "1.0.0", HTTPProfile: &invalidProfile}}})
	if err != nil {
		t.Fatal(err)
	}
	resp, body := live.DoJSON(t, http.MethodPut, sandboxesPath(store.sandbox.ID, "/companion-services"), string(invalidDeclaration))
	if resp.StatusCode != http.StatusBadRequest || owner.starts != 0 {
		t.Fatalf("duplicate-selector profile was not rejected before owner start: status=%d starts=%d body=%s", resp.StatusCode, owner.starts, body)
	}
	declaration, err := json.Marshal(struct {
		Services []handlers.CompanionServiceSpec `json:"services"`
	}{Services: []handlers.CompanionServiceSpec{{Name: "public-apis", Kind: "http-mock", Version: "1.0.0", HTTPProfile: &profile}}})
	if err != nil {
		t.Fatal(err)
	}
	resp, body = live.DoJSON(t, http.MethodPut, sandboxesPath(store.sandbox.ID, "/companion-services"), string(declaration))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile declaration status=%d body=%s", resp.StatusCode, body)
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(store.sandbox.ID, "/companion-services/public-apis/start"), "{}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile start status=%d body=%s", resp.StatusCode, body)
	}
	t.Cleanup(func() {
		if owner.server != nil {
			owner.server.Close()
		}
	})
	var started struct {
		OperationAddresses []handlers.HTTPMockOperationAddress `json:"operationAddresses"`
		ProfileSHA256      string                              `json:"profileSha256"`
	}
	if err := json.Unmarshal([]byte(body), &started); err != nil {
		t.Fatalf("decode profile start result: %v; body=%s", err, body)
	}
	_, expectedHash, err := handlers.NormalizeHTTPMockProfile(profile)
	if err != nil || started.ProfileSHA256 != expectedHash || len(started.OperationAddresses) != 2 {
		t.Fatalf("profile digest or operation map mismatch: result=%+v expectedHash=%q err=%v", started, expectedHash, err)
	}
	byOperation := map[string]handlers.HTTPMockOperationAddress{}
	for _, address := range started.OperationAddresses {
		byOperation[address.Selector.NodeType+"/"+address.Selector.Resource+"/"+address.Selector.Operation] = address
	}
	gmail := byOperation["n8n-nodes-base.gmail/message/send"]
	sheets := byOperation["n8n-nodes-base.googleSheets/sheet/append"]
	if gmail.Method != http.MethodPost || gmail.URL != "http://gmail.synthetic.invalid:18080/v1/messages" || sheets.Method != http.MethodPost || sheets.URL != "http://sheets.synthetic.invalid:18080/v1/rows?spreadsheetId=synthetic-sheet" {
		t.Fatalf("selector addresses are not exact: %+v", started.OperationAddresses)
	}
	if _, found := byOperation["n8n-nodes-base.hubspot/contact/upsert"]; found {
		t.Fatal("unmatched selector unexpectedly received a route")
	}
	resp, body = live.DoJSON(t, http.MethodGet, sandboxesPath(store.sandbox.ID, "/companion-services"), "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), expectedHash) || !strings.Contains(string(body), "operationAddresses") {
		t.Fatalf("persisted profile start state is missing selector routes/hash: status=%d body=%s", resp.StatusCode, body)
	}

	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		dialer := net.Dialer{}
		return dialer.DialContext(ctx, "tcp", owner.server.Listener.Addr().String())
	}}
	client := &http.Client{Transport: transport}
	defer transport.CloseIdleConnections()
	for _, expected := range []struct {
		address, body string
		status        int
	}{
		{gmail.URL, `{"messageId":"message_synthetic_1","status":"queued"}`, http.StatusAccepted},
		{sheets.URL, `{"rowId":"row_synthetic_1"}`, http.StatusCreated},
	} {
		response, err := client.Post(expected.address, "application/json", strings.NewReader(`{"synthetic":true}`))
		if err != nil {
			t.Fatal(err)
		}
		responseBody, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil || response.StatusCode != expected.status || string(responseBody) != expected.body {
			t.Fatalf("profile response status=%d body=%q err=%v", response.StatusCode, responseBody, readErr)
		}
	}
	missing, err := client.Get("http://gmail.synthetic.invalid:18080/v1/unknown")
	if err != nil {
		t.Fatal(err)
	}
	missingBody, err := io.ReadAll(missing.Body)
	missing.Body.Close()
	if err != nil || missing.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown route must fail closed: status=%d body=%q err=%v", missing.StatusCode, missingBody, err)
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(store.sandbox.ID, "/companion-services/public-apis/stop"), "{}")
	if resp.StatusCode != http.StatusOK || owner.stops != 1 {
		t.Fatalf("profile stop status=%d stops=%d body=%s", resp.StatusCode, owner.stops, body)
	}
	var stopped struct {
		Status             string                              `json:"status"`
		ProfileSHA256      string                              `json:"profileSha256"`
		OperationAddresses []handlers.HTTPMockOperationAddress `json:"operationAddresses"`
	}
	if err := json.Unmarshal([]byte(body), &stopped); err != nil || stopped.Status != "declared" || stopped.ProfileSHA256 != "" || len(stopped.OperationAddresses) != 0 {
		t.Fatalf("stopped profile state retained runtime-only mapping: %+v err=%v body=%s", stopped, err, body)
	}
}

// TestTaskOwnedSandboxCreateAssociateProfileAndCleanup proves the existing
// Sandbox acquisition and companion routes compose into one task-owned
// profile lifecycle. No rehearsal-specific creation endpoint is required:
// the caller creates a sandbox with its owner/idempotency identity, then
// uses the returned UUID with the existing companion declaration routes.
func TestTaskOwnedSandboxCreateAssociateProfileAndCleanup(t *testing.T) {
	id := uuid.New()
	store := &companionStore{sandbox: &types.Sandbox{ID: id, Status: types.StatusActive, Metadata: map[string]interface{}{}}}
	owner := &routeHTTPMockOwner{}
	provider := handlers.N8NHTTPMockServiceProvider{Owner: owner}
	var createReq *types.CreateRequest
	deletedID := uuid.Nil
	svc := &sandboxiface.FakeService{
		CreateFn: func(_ context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
			createReq = req
			return &types.Sandbox{ID: id, ScopePath: req.ScopePath, ProjectRoot: req.ProjectRoot, Owner: req.Owner, Status: types.StatusActive}, nil
		},
		DeleteFn: func(_ context.Context, sandboxID uuid.UUID) error {
			deletedID = sandboxID
			return nil
		},
	}
	live := newLive(t, svc, withCompanionStore(store), withCompanionProvider(handlers.N8NCompanionServiceRouter{HTTPMock: provider}))

	resp, body := live.DoJSON(t, http.MethodPost, "/api/v1/sandboxes", `{"name":"f7-rehearsal-task-42","scopePath":"/synthetic-rehearsal","projectRoot":"/workspace/client-rehearsal","owner":"client-rehearsal:task-42","tags":["f7","task-owned"],"idempotencyKey":"client-rehearsal:task-42"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("task-owned sandbox create status=%d body=%s", resp.StatusCode, body)
	}
	var created types.Sandbox
	if err := json.Unmarshal(body, &created); err != nil || created.ID != id || createReq == nil || createReq.Owner != "client-rehearsal:task-42" || createReq.IdempotencyKey != "client-rehearsal:task-42" {
		t.Fatalf("created sandbox identity/ownership mismatch: sandbox=%+v request=%+v err=%v", created, createReq, err)
	}

	profile := syntheticAPIOperationProfile()
	declaration, err := json.Marshal(struct {
		Services []handlers.CompanionServiceSpec `json:"services"`
	}{Services: []handlers.CompanionServiceSpec{{Name: "workflow-apis", Kind: "http-mock", Version: "1.0.0", HTTPProfile: &profile}}})
	if err != nil {
		t.Fatal(err)
	}
	resp, body = live.DoJSON(t, http.MethodPut, sandboxesPath(created.ID, "/companion-services"), string(declaration))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("associate normalized profile with created sandbox status=%d body=%s", resp.StatusCode, body)
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(created.ID, "/companion-services/workflow-apis/start"), "{}")
	if resp.StatusCode != http.StatusOK || owner.sandboxID != created.ID || owner.starts != 1 {
		t.Fatalf("start on created task sandbox status=%d owner sandbox=%s starts=%d body=%s", resp.StatusCode, owner.sandboxID, owner.starts, body)
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(created.ID, "/companion-services/workflow-apis/stop"), "{}")
	if resp.StatusCode != http.StatusOK || owner.stops != 1 {
		t.Fatalf("stop task profile status=%d stops=%d body=%s", resp.StatusCode, owner.stops, body)
	}
	resp, body = live.Do(t, http.MethodDelete, sandboxesPath(created.ID, ""), nil)
	if resp.StatusCode != http.StatusNoContent || deletedID != created.ID || owner.stops != 1 {
		t.Fatalf("delete created task sandbox status=%d deleted=%s stops=%d body=%s", resp.StatusCode, deletedID, owner.stops, body)
	}
}

func (o *routeHTTPMockOwner) StopHTTPMock(_ context.Context, sandboxID uuid.UUID, runtimeID string) error {
	if sandboxID != o.sandboxID || runtimeID != "sandbox.http.route-test" {
		return fmt.Errorf("foreign HTTP mock cleanup target")
	}
	o.stops++
	o.server.Close()
	return nil
}

func TestHTTPMockWorkspaceRouteStartsExactSyntheticServiceAndStopsIt(t *testing.T) {
	store := newCompanionStore()
	owner := &routeHTTPMockOwner{}
	provider := handlers.N8NHTTPMockServiceProvider{Owner: owner}
	live := newLive(t, &sandboxiface.FakeService{}, withCompanionStore(store), withCompanionProvider(handlers.N8NCompanionServiceRouter{HTTPMock: provider}))
	declaration, err := json.Marshal(struct {
		Services []handlers.CompanionServiceSpec `json:"services"`
	}{Services: []handlers.CompanionServiceSpec{{Name: "workflow-apis", Kind: "http-mock", Version: "1.0.0", HTTPFixtures: validHTTPFixtures}}})
	if err != nil {
		t.Fatal(err)
	}
	resp, body := live.DoJSON(t, http.MethodPut, sandboxesPath(store.sandbox.ID, "/companion-services"), string(declaration))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("declare status=%d body=%s", resp.StatusCode, body)
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(store.sandbox.ID, "/companion-services/workflow-apis/start"), "{}")
	if resp.StatusCode != http.StatusOK || owner.starts != 1 || owner.sandboxID != store.sandbox.ID {
		t.Fatalf("start status=%d owner starts=%d body=%s", resp.StatusCode, owner.starts, body)
	}
	t.Cleanup(func() {
		if owner.server != nil {
			owner.server.Close()
		}
	})
	var started struct {
		Addresses map[string]string `json:"addresses"`
	}
	if err := json.Unmarshal([]byte(body), &started); err != nil {
		t.Fatalf("decode started service contract: %v; body=%s", err, body)
	}
	address := started.Addresses["crm.synthetic.invalid"]
	if address != "http://crm.synthetic.invalid:18080" || owner.server == nil {
		t.Fatalf("synthetic route map = %+v", started.Addresses)
	}

	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		dialer := net.Dialer{}
		return dialer.DialContext(ctx, "tcp", owner.server.Listener.Addr().String())
	}}
	client := &http.Client{Transport: transport}
	defer transport.CloseIdleConnections()
	request, err := http.NewRequest(http.MethodPost, address+"/v1/contacts/42", strings.NewReader(`{"name":"Synthetic Contact"}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	responseBody, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated || string(responseBody) != `{"id":"contact_synthetic_42","email":"person@example.invalid","status":"created"}` || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("exact synthetic route response status=%d headers=%v body=%s", response.StatusCode, response.Header, responseBody)
	}
	queryResponse, err := client.Get(started.Addresses["crm.synthetic.invalid"] + "/v1/contacts?limit=2&sort=created")
	if err != nil {
		t.Fatal(err)
	}
	queryBody, err := io.ReadAll(queryResponse.Body)
	queryResponse.Body.Close()
	if err != nil || queryResponse.StatusCode != http.StatusOK || string(queryBody) != `[{"id":"contact_synthetic_42"}]` {
		t.Fatalf("declared query route status=%d body=%q err=%v", queryResponse.StatusCode, queryBody, err)
	}

	for _, target := range []struct{ method, url string }{
		{http.MethodPost, "http://crm.synthetic.invalid:18080/v1/contacts/other"},
		{http.MethodPost, "http://crm.synthetic.invalid:18080/v1/contacts/42?expand=all"},
		{http.MethodGet, "http://crm.synthetic.invalid:18080/v1/contacts/42"},
		{http.MethodGet, "http://crm.synthetic.invalid:18080/v1/contacts?limit=3&sort=created"},
		{http.MethodGet, "http://crm.synthetic.invalid:18080/v1/contacts?limit=2&sort=created&extra=true"},
		{http.MethodPost, "http://crm.synthetic.invalid:9999/v1/contacts/42"},
		{http.MethodPost, "http://unmapped.example.net:18080/v1/contacts/42"},
	} {
		request, err := http.NewRequest(target.method, target.url, nil)
		if err != nil {
			t.Fatal(err)
		}
		missing, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		missingBody, readErr := io.ReadAll(missing.Body)
		missing.Body.Close()
		if readErr != nil || missing.StatusCode != http.StatusNotFound || string(missingBody) != "404 page not found\n" {
			t.Fatalf("unmapped request %s %q status=%d body=%q err=%v", target.method, target.url, missing.StatusCode, missingBody, readErr)
		}
	}
	resp, body = live.DoJSON(t, http.MethodPost, sandboxesPath(store.sandbox.ID, "/companion-services/workflow-apis/stop"), "{}")
	if resp.StatusCode != http.StatusOK || owner.stops != 1 {
		t.Fatalf("stop status=%d owner stops=%d body=%s", resp.StatusCode, owner.stops, body)
	}
	var stopped struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &stopped); err != nil || stopped.Status != "declared" {
		t.Fatalf("stopped service state=%+v err=%v body=%s", stopped, err, body)
	}
	if _, err := client.Get(address + "/v1/contacts/42"); err == nil {
		t.Fatal("service remained reachable after supported stop route")
	}
}

func ExampleNormalizeFixtureHost() {
	got, err := handlers.NormalizeFixtureHost("CRM.synthetic.invalid")
	fmt.Println(got, err)
	// Output: crm.synthetic.invalid <nil>
}

var _ sandbox.ServiceAPI = (*sandboxiface.FakeService)(nil)
var _ handlers.N8NHTTPMockOwner = (*n8nHTTPMockOwnerFake)(nil)

func TestHTTPMockCLIRejectsOversizedPayloadBeforeCommand(t *testing.T) {
	cli := handlers.N8NHTTPMockCLI{Binary: filepath.Join(t.TempDir(), "missing")}
	fixtures := []handlers.HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: http.MethodPost, Path: "/v1/x", Status: 200, ContentType: "text/plain", Body: strings.Repeat("x", maxHTTPBodyForTest())}}
	if _, err := cli.StartHTTPMock(context.Background(), uuid.New(), "crm", "1.0.0", fixtures); err == nil {
		t.Fatal("expected fixture limit rejection")
	}
}

func maxHTTPBodyForTest() int { return (64 << 10) + 1 }
