package deployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProviderPathCatalogIsFixtureBackedAndRecordsUnknowns(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "provider-path-catalog.json")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	catalog, err := LoadCatalog(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Records) != 7 {
		t.Fatalf("catalogue has %d rows; want 7", len(catalog.Records))
	}
	var bitlaunch *PathRecord
	for i := range catalog.Records {
		if catalog.Records[i].Provider == "bitlaunch" {
			bitlaunch = &catalog.Records[i]
			break
		}
	}
	if bitlaunch == nil || len(bitlaunch.Evidence) != 0 || bitlaunch.KYC == "" || !strings.Contains(bitlaunch.Result, "unqualified") {
		t.Fatalf("BitLaunch uncertainty was not kept explicit: %+v", bitlaunch)
	}
	freePath, err := catalog.Path("turbo", "ar-io-free-testnet-sandbox")
	if err != nil || freePath.UploadURL != "https://upload.services.ar-io.dev" || freePath.GatewayURL != "https://ar-io.dev" || freePath.MaxItemBytes != TurboSandboxMaxItemBytes || !strings.Contains(freePath.DeadEnd, "no guaranteed remote deletion") {
		t.Fatalf("Turbo free sandbox path is incomplete: %+v err=%v", freePath, err)
	}
}

type memoryStorage struct{ calls []string }

func (m *memoryStorage) Name() string { return "storage" }
func (m *memoryStorage) Deploy(context.Context, Artifact) (StoredContent, error) {
	m.calls = append(m.calls, "deploy")
	return StoredContent{Provider: "storage", CID: "bafy-test", URI: "ipfs://bafy-test"}, nil
}
func (m *memoryStorage) Renew(context.Context, StoredContent) error {
	m.calls = append(m.calls, "renew")
	return nil
}
func (m *memoryStorage) Teardown(context.Context, StoredContent) error {
	m.calls = append(m.calls, "teardown")
	return nil
}
func (m *memoryStorage) Resolve(context.Context, StoredContent) ([]byte, error) {
	m.calls = append(m.calls, "resolve")
	return []byte("reference site"), nil
}

type routedFixtureStorage struct {
	provider string
	calls    []string
}

func (r *routedFixtureStorage) Name() string { return r.provider }
func (r *routedFixtureStorage) Deploy(_ context.Context, artifact Artifact) (StoredContent, error) {
	r.calls = append(r.calls, "deploy:"+artifact.StorageProvider)
	return StoredContent{Provider: r.provider, CID: "fixture", URI: "fixture://" + r.provider}, nil
}
func (r *routedFixtureStorage) Renew(context.Context, StoredContent) error {
	r.calls = append(r.calls, "renew")
	return nil
}
func (r *routedFixtureStorage) Teardown(context.Context, StoredContent) error {
	r.calls = append(r.calls, "teardown")
	return nil
}
func (r *routedFixtureStorage) Resolve(context.Context, StoredContent) ([]byte, error) {
	r.calls = append(r.calls, "resolve")
	return []byte(r.provider), nil
}

func TestStorageRouterDispatchesPersistedProviderAndProviderFaults(t *testing.T) {
	kubo := &routedFixtureStorage{provider: "kubo"}
	turbo := &routedFixtureStorage{provider: "turbo-sandbox"}
	router := StorageRouter{Default: "kubo", Adapters: map[string]StorageAdapter{"kubo": kubo, "turbo-sandbox": turbo}}
	faults := NewFaults()
	service, err := NewService(Gate{Mode: GateStub, AllowFreeSandbox: true}, faults, router)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := faults.Set("turbo-sandbox", FaultRefuse); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Deploy(ctx, "refused", Artifact{Path: "/fixture", StorageProvider: "turbo-sandbox"}, "sandbox-name"); err == nil {
		t.Fatal("provider-specific refusal hook did not stop the selected storage adapter")
	}
	if len(turbo.calls) != 0 {
		t.Fatalf("refused provider was called: %v", turbo.calls)
	}
	if err := faults.Set("turbo-sandbox", FaultNone); err != nil {
		t.Fatal(err)
	}
	record, err := service.Deploy(ctx, "accepted", Artifact{Path: "/fixture", StorageProvider: "turbo-sandbox"}, "sandbox-name")
	if err != nil || record.Storage.Provider != "turbo-sandbox" {
		t.Fatalf("record=%+v err=%v", record, err)
	}
	if _, body, err := service.Resolve(ctx, "accepted"); err != nil || string(body) != "turbo-sandbox" {
		t.Fatalf("resolve=%q err=%v", body, err)
	}
	if _, err := service.Renew(ctx, "accepted"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Teardown(ctx, "accepted"); err != nil {
		t.Fatal(err)
	}
	if len(kubo.calls) != 0 || strings.Join(turbo.calls, ",") != "deploy:turbo-sandbox,resolve,renew,teardown" {
		t.Fatalf("provider routing kubo=%v turbo=%v", kubo.calls, turbo.calls)
	}
}

type memoryNames struct {
	name   string
	target string
	calls  []string
}

func (m *memoryNames) Name() string { return m.name }
func (m *memoryNames) Deploy(_ context.Context, name, target string) (NamedContent, error) {
	m.calls = append(m.calls, "deploy")
	m.target = target
	return NamedContent{Provider: m.name, Name: name, Target: target}, nil
}
func (m *memoryNames) Renew(context.Context, NamedContent) error {
	m.calls = append(m.calls, "renew")
	return nil
}
func (m *memoryNames) Teardown(context.Context, NamedContent) error {
	m.calls = append(m.calls, "teardown")
	return nil
}
func (m *memoryNames) Resolve(context.Context, NamedContent) (string, error) {
	m.calls = append(m.calls, "resolve")
	return m.target, nil
}

func TestLifecycleDeployResolveRenewTeardown(t *testing.T) {
	storage := &memoryStorage{}
	ens := &memoryNames{name: "ens"}
	arns := &memoryNames{name: "arns"}
	service, err := NewService(Gate{Mode: GateStub}, NewFaults(), storage, ens, arns)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	record, err := service.Deploy(ctx, "demo", Artifact{Path: "/reference", Name: "site"}, "ci1-demo")
	if err != nil || record.State != "deployed" || len(record.Names) != 2 {
		t.Fatalf("deploy record=%+v err=%v", record, err)
	}
	if _, body, err := service.Resolve(ctx, "demo"); err != nil || string(body) != "reference site" {
		t.Fatalf("resolve body=%q err=%v", body, err)
	}
	if _, err := service.Renew(ctx, "demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Teardown(ctx, "demo"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Resolve(ctx, "demo"); err == nil {
		t.Fatal("resolve after teardown succeeded")
	}
	if storage.calls[len(storage.calls)-1] != "teardown" {
		t.Fatalf("storage operations = %v", storage.calls)
	}
	if ens.calls[len(ens.calls)-1] != "teardown" || arns.calls[len(arns.calls)-1] != "teardown" {
		t.Fatalf("naming teardown calls: ens=%v arns=%v", ens.calls, arns.calls)
	}
}

func TestPersistedNamesRouteAcrossAdapterReorderingAndRestart(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRecordRepository()
	storage := &memoryStorage{}
	ens := &memoryNames{name: "ens"}
	arns := &memoryNames{name: "arns"}
	first, err := NewServiceWithRecords(Gate{Mode: GateStub}, NewFaults(), storage, repository, ens, arns)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Deploy(ctx, "restartable", Artifact{Path: "/reference"}, "site"); err != nil {
		t.Fatal(err)
	}
	// A restarted process may construct adapters in a different order. Persisted
	// provider identities, not slice positions, decide which adapter is called.
	ensAfterRestart := &memoryNames{name: "ens", target: "ipfs://bafy-test"}
	arnsAfterRestart := &memoryNames{name: "arns", target: "ipfs://bafy-test"}
	second, err := NewServiceWithRecords(Gate{Mode: GateStub}, NewFaults(), storage, repository, arnsAfterRestart, ensAfterRestart)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := second.Resolve(ctx, "restartable"); err != nil {
		t.Fatalf("resolve after adapter reordering: %v", err)
	}
	if _, err := second.Renew(ctx, "restartable"); err != nil {
		t.Fatalf("renew after adapter reordering: %v", err)
	}
	if _, err := second.Teardown(ctx, "restartable"); err != nil {
		t.Fatalf("teardown after adapter reordering: %v", err)
	}
	for provider, adapter := range map[string]*memoryNames{"ens": ensAfterRestart, "arns": arnsAfterRestart} {
		calls := strings.Join(adapter.calls, ",")
		if calls != "resolve,renew,teardown" {
			t.Fatalf("provider %s calls after restart = %s", provider, calls)
		}
	}
}

func TestRenewFailsBeforeStorageEffectWhenPersistedNameAdapterMissing(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRecordRepository()
	if err := repository.Put(ctx, Record{ID: "old", Storage: StoredContent{Provider: "storage"}, Names: []NamedContent{{Provider: "ens", Name: "old", Target: "fixture"}}, State: "deployed"}); err != nil {
		t.Fatal(err)
	}
	storage := &memoryStorage{}
	service, err := NewServiceWithRecords(Gate{Mode: GateStub}, NewFaults(), storage, repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Renew(ctx, "old"); err == nil || !strings.Contains(err.Error(), `naming provider "ens"`) {
		t.Fatalf("expected explicit missing-provider error, got %v", err)
	}
	if strings.Contains(strings.Join(storage.calls, ","), "renew") {
		t.Fatalf("storage renewal ran before missing-name preflight: %v", storage.calls)
	}
}

func TestEffectGateRefusesWithoutAuthorization(t *testing.T) {
	for _, mode := range []GateMode{GateStub, GateLive} {
		if err := (Gate{Mode: mode}).Check(context.Background(), EffectFunded, "turbo"); err == nil {
			t.Fatalf("mode %s admitted funded effect without authorization", mode)
		}
	}
	if err := (Gate{Mode: GateStub}).Check(context.Background(), EffectDevnet, "kubo"); err != nil {
		t.Fatalf("devnet effect refused: %v", err)
	}
	freeSandbox := Gate{Mode: GateStub, AllowFreeSandbox: true}
	if err := freeSandbox.Check(context.Background(), EffectSandbox, "turbo-sandbox"); err != nil {
		t.Fatalf("mission-approved free sandbox effect refused: %v", err)
	}
	if err := freeSandbox.Check(context.Background(), EffectFunded, "turbo-sandbox"); err == nil {
		t.Fatal("free sandbox allowance opened a funded effect")
	}
	called := false
	err := (Gate{Mode: GateLive, Authorize: func(context.Context, Effect, string) error {
		called = true
		return errors.New("no runbook")
	}}).Check(context.Background(), EffectPublic, "arns")
	if err == nil || !called {
		t.Fatalf("authorization error=%v callback called=%v", err, called)
	}
}

func TestFaultHooksRefuseTimeoutAndVanish(t *testing.T) {
	faults := NewFaults()
	for _, fault := range []Fault{FaultRefuse, FaultVanish} {
		if err := faults.Set("kubo", fault); err != nil {
			t.Fatal(err)
		}
		if err := faults.Before(context.Background(), "kubo"); err == nil {
			t.Fatalf("fault %s did not fail", fault)
		}
	}
	if err := faults.Set("kubo", FaultTimeout); err != nil {
		t.Fatal(err)
	}
	faults.Timeout = time.Millisecond
	if err := faults.Before(context.Background(), "kubo"); err == nil {
		t.Fatal("timeout fault did not fail")
	}
	if err := faults.Set("kubo", FaultNone); err != nil {
		t.Fatal(err)
	}
	if err := faults.Before(context.Background(), "kubo"); err != nil {
		t.Fatalf("cleared fault remains active: %v", err)
	}
}

func TestDeployCompensatesStorageWhenNameProviderRefuses(t *testing.T) {
	storage := &memoryStorage{}
	names := &memoryNames{name: "ens"}
	faults := NewFaults()
	if err := faults.Set("ens", FaultRefuse); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(Gate{Mode: GateStub}, faults, storage, names)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Deploy(context.Background(), "demo", Artifact{Path: "/reference", Name: "site"}, "demo.eth"); err == nil {
		t.Fatal("provider refusal did not fail deployment")
	}
	if len(storage.calls) != 2 || storage.calls[1] != "teardown" {
		t.Fatalf("storage compensation missing: %v", storage.calls)
	}
}
