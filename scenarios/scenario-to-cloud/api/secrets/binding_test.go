package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
)

// memoryTarget is the smallest Distributor that behaves like a target.
type memoryTarget struct {
	values   map[string]string
	versions map[string]int64
	argv     []string
	revoked  []int64
}

func (m *memoryTarget) Transport() string { return identity.TransportSSH }
func (m *memoryTarget) Probe(_ context.Context, _ credentials.Target, b domain.CredentialBinding) (credentials.ProbeResult, error) {
	_, ok := m.values[b.ID]
	return credentials.ProbeResult{Configured: ok, StoreUnlocked: true, StoreState: "available"}, nil
}

func (m *memoryTarget) Deliver(_ context.Context, req credentials.DeliverRequest) (credentials.Receipt, error) {
	m.argv = append(m.argv, "ingest "+req.Binding.ID+" v"+string(rune('0'+req.Version.Number)))
	m.values[req.Binding.ID] = req.Value
	m.versions[req.Binding.ID] = req.Version.Number
	return credentials.Receipt{Transport: m.Transport(), Ref: req.OperationID + "/" + req.Step}, nil
}

func (m *memoryTarget) Acknowledge(_ context.Context, req credentials.AckRequest) (credentials.Receipt, error) {
	if m.versions[req.Binding.ID] != req.Version {
		return credentials.Receipt{}, errors.New("version mismatch")
	}
	return credentials.Receipt{Transport: m.Transport(), Ref: "ack"}, nil
}

func (m *memoryTarget) Revoke(_ context.Context, req credentials.RevokeRequest) (credentials.Receipt, error) {
	m.revoked = append(m.revoked, req.Version)
	if m.versions[req.Binding.ID] == req.Version {
		delete(m.values, req.Binding.ID)
	}
	return credentials.Receipt{Transport: m.Transport(), Ref: "revoke", Limitations: []string{credentials.RevocationLimitation}}, nil
}

func fixtureLifecycle(t *testing.T) (BindingLifecycle, *memoryTarget, *credentials.MemoryStore) {
	t.Helper()
	target := &memoryTarget{values: map[string]string{}, versions: map[string]int64{}}
	store := credentials.NewMemoryStore()
	service := &credentials.Service{Store: store, Distributor: target, Providers: credentials.DefaultProviders(nil, "fixture")}
	service.Providers[domain.CredentialClassExternalAPICredential] = &credentials.ExternalAPIProvider{Name: "external", Probe: credentials.ShapeProbe, RevokeAPI: func(context.Context, domain.CredentialBinding, domain.CredentialVersion) error { return nil }}
	manifest := domain.CloudManifest{Scenario: domain.ManifestScenario{ID: "demo-app"}, Secrets: &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{{ID: "mailer-token", Class: domain.SecretClassUserPrompt, Target: domain.BundleSecretTarget{Type: "env", Name: "MAILER_TOKEN"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/mailer", Field: "api-token"}}}}}
	bound := credentials.Target{DeploymentID: "dep-1", Ref: identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10"}}}
	return NewBindingLifecycle(service, bound, manifest), target, store
}

// [REQ:STC-P0-032] Post-deploy secrets management runs on the binding model:
// create materialises a version, replace rotates, delete revokes, keys map to
// exact descriptors (declared ones by target name), and reads carry
// references only.
func TestManagementRunsOnTheBindingModel(t *testing.T) {
	lifecycle, target, _ := fixtureLifecycle(t)
	ctx := context.Background()
	created, err := lifecycle.Create(ctx, "MAILER_TOKEN", "canary-mailer-1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Descriptor.Address() != "fixture/mailer:api-token" || created.Version.Number != 1 || created.Class != domain.CredentialClassExternalAPICredential {
		t.Fatalf("created = %+v", created)
	}
	if _, err := lifecycle.Create(ctx, "MAILER_TOKEN", "canary-mailer-dup"); err == nil || !strings.Contains(err.Error(), "already has an active") {
		t.Fatalf("duplicate create = %v", err)
	}
	extra, err := lifecycle.Create(ctx, "EXTRA_API_KEY", "canary-extra-1")
	if err != nil || extra.Descriptor.Address() != "vrooli/demo-app:extra-api-key" {
		t.Fatalf("operator-added key = %+v err=%v", extra, err)
	}
	entries, err := lifecycle.List(ctx)
	if err != nil || len(entries) != 2 || entries[0].Key != "EXTRA_API_KEY" || entries[1].Key != "MAILER_TOKEN" || !entries[1].Masked || entries[1].Version != 1 {
		t.Fatalf("list = %+v err=%v", entries, err)
	}
	raw, _ := json.Marshal(entries)
	if strings.Contains(string(raw), "canary-") {
		t.Fatal("a value leaked into the listing")
	}
	rotation, err := lifecycle.Replace(ctx, "MAILER_TOKEN", "canary-mailer-2")
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if rotation.State != domain.RotationComplete || rotation.ToVersion != 2 || target.values[created.ID] != "canary-mailer-2" {
		t.Fatalf("replace rotation = %+v", rotation)
	}
	entry, err := lifecycle.Get(ctx, "MAILER_TOKEN")
	if err != nil || entry.Version != 2 || entry.BindingID != created.ID {
		t.Fatalf("get = %+v err=%v", entry, err)
	}
	revoke, err := lifecycle.Delete(ctx, "MAILER_TOKEN")
	if err != nil || revoke.State != domain.RotationComplete {
		t.Fatalf("delete = %+v err=%v", revoke, err)
	}
	if _, still := target.values[created.ID]; still {
		t.Fatal("delete left the value on the target")
	}
	if entry, err := lifecycle.Get(ctx, "MAILER_TOKEN"); err != nil || entry.State != string(domain.CredentialBindingRevoked) {
		t.Fatalf("revoked get = %+v err=%v", entry, err)
	}
	if _, err := lifecycle.Get(ctx, "NOPE"); err == nil || !strings.Contains(err.Error(), "no credential binding") {
		t.Fatalf("unknown key = %v", err)
	}
	for _, a := range target.argv {
		if strings.Contains(a, "canary-") {
			t.Fatalf("value in argv: %s", a)
		}
	}
}

// [REQ:STC-P0-032] The HTTP handlers reject reveal, validate keys and values,
// and never return values.
func TestManagementHandlersNeverRevealValues(t *testing.T) {
	lifecycle, _, _ := fixtureLifecycle(t)
	deps := ManagementDeps{Lifecycle: func(context.Context, string) (BindingLifecycle, error) { return lifecycle, nil }}
	router := mux.NewRouter()
	router.HandleFunc("/deployments/{id}/secrets", HandleListVPSSecrets(deps)).Methods("GET")
	router.HandleFunc("/deployments/{id}/secrets", HandleCreateVPSSecret(deps)).Methods("POST")
	router.HandleFunc("/deployments/{id}/secrets/{key}", HandleGetVPSSecret(deps)).Methods("GET")
	router.HandleFunc("/deployments/{id}/secrets/{key}", HandleUpdateVPSSecret(deps)).Methods("PUT")
	router.HandleFunc("/deployments/{id}/secrets/{key}", HandleDeleteVPSSecret(deps)).Methods("DELETE")
	do := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	if rec := do(http.MethodPost, "/deployments/dep-1/secrets", `{"key":"bad key","value":"x"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad key = %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(http.MethodPost, "/deployments/dep-1/secrets", `{"key":"MY_KEY","value":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty value = %d", rec.Code)
	}
	rec := do(http.MethodPost, "/deployments/dep-1/secrets", `{"key":"MY_KEY","value":"canary-http-1"}`)
	if rec.Code != http.StatusCreated || strings.Contains(rec.Body.String(), "canary-http-1") {
		t.Fatalf("create = %d %s", rec.Code, rec.Body.String())
	}
	rec = do(http.MethodGet, "/deployments/dep-1/secrets/MY_KEY?reveal=true", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("reveal = %d %s", rec.Code, rec.Body.String())
	}
	rec = do(http.MethodGet, "/deployments/dep-1/secrets", "")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "canary-http-1") || !strings.Contains(rec.Body.String(), `"binding_id"`) {
		t.Fatalf("list = %d %s", rec.Code, rec.Body.String())
	}
	rec = do(http.MethodPut, "/deployments/dep-1/secrets/MY_KEY", `{"value":"canary-http-2"}`)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "canary-http-2") || !strings.Contains(rec.Body.String(), `"rotation_id"`) {
		t.Fatalf("update = %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(http.MethodDelete, "/deployments/dep-1/secrets/MY_KEY", `{"confirmation":"no"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("delete without confirmation = %d", rec.Code)
	}
	rec = do(http.MethodDelete, "/deployments/dep-1/secrets/MY_KEY", `{"confirmation":"DELETE"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"operation_state":"complete"`) {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body.String())
	}
}
