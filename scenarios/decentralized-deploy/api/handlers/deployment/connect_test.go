package deployment

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	core "decentralized-deploy/internal/deployment"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	deploymentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment"
	deploymentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment/deployment_v1connect"
	_ "modernc.org/sqlite"
)

type connectStorage struct{}

func (connectStorage) Name() string { return "test-storage" }
func (s connectStorage) Deploy(_ context.Context, artifact core.Artifact) (core.StoredContent, error) {
	provider := artifact.StorageProvider
	if provider == "" {
		provider = s.Name()
	}
	return core.StoredContent{Provider: provider, CID: "bafy-connect", URI: "ipfs://bafy-connect"}, nil
}
func (connectStorage) Renew(context.Context, core.StoredContent) error    { return nil }
func (connectStorage) Teardown(context.Context, core.StoredContent) error { return nil }
func (connectStorage) Resolve(context.Context, core.StoredContent) ([]byte, error) {
	return []byte("connect reference bytes"), nil
}

func TestDeploymentConnectLifecycle(t *testing.T) {
	service, err := core.NewService(core.Gate{Mode: core.GateStub}, core.NewFaults(), connectStorage{})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	ConnectModule(service, service.Records, "/fixtures/reference").Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := deploymentconnect.NewDeploymentServiceClient(http.DefaultClient, server.URL)

	deployed, err := client.Deploy(context.Background(), connect.NewRequest(&deploymentv1.DeployRequest{Id: "connect-1", Name: "reference.eth", ComputeReference: "compute://manager/ref-1", StorageProvider: "turbo-sandbox"}))
	if err != nil || deployed.Msg.GetDeployment().GetContentUri() != "ipfs://bafy-connect" || deployed.Msg.GetDeployment().GetComputeReference() != "compute://manager/ref-1" || deployed.Msg.GetDeployment().GetStorageProvider() != "turbo-sandbox" {
		t.Fatalf("deploy=%v err=%v", deployed, err)
	}
	resolved, err := client.Resolve(context.Background(), connect.NewRequest(&deploymentv1.ResolveRequest{Id: "connect-1"}))
	if err != nil || string(resolved.Msg.GetContent()) != "connect reference bytes" {
		t.Fatalf("resolve=%v err=%v", resolved, err)
	}
	renewed, err := client.Renew(context.Background(), connect.NewRequest(&deploymentv1.RenewRequest{Id: "connect-1"}))
	if err != nil || renewed.Msg.GetDeployment().GetState() != "renewed" {
		t.Fatalf("renew=%v err=%v", renewed, err)
	}
	teardown, err := client.Teardown(context.Background(), connect.NewRequest(&deploymentv1.TeardownRequest{Id: "connect-1"}))
	if err != nil || teardown.Msg.GetDeployment().GetState() != "torn-down" {
		t.Fatalf("teardown=%v err=%v", teardown, err)
	}
	got, err := client.Get(context.Background(), connect.NewRequest(&deploymentv1.GetRequest{Id: "connect-1"}))
	if err != nil || got.Msg.GetDeployment().GetState() != "torn-down" {
		t.Fatalf("get=%v err=%v", got, err)
	}
}

func TestDeploymentConnectRealKuboReferenceLifecycle(t *testing.T) {
	binary := os.Getenv("CI1_KUBO_BINARY")
	if binary == "" {
		t.Skip("set CI1_KUBO_BINARY to exercise the Connect lifecycle against managed offline Kubo")
	}
	scenarioRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	artifactRoot := filepath.Join(scenarioRoot, "examples", "reference")
	expected, err := os.ReadFile(filepath.Join(artifactRoot, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Join(t.TempDir(), "kubo-repos")
	storage := core.KuboAdapter{Binary: binary, Repo: repoRoot, Entry: "site/index.html", Gate: core.Gate{Mode: core.GateStub}}
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "records.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	records, err := core.NewSQLRecordRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	service, err := core.NewServiceWithRecords(core.Gate{Mode: core.GateStub}, core.NewFaults(), storage, records)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	ConnectModule(service, service.Records, artifactRoot).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := deploymentconnect.NewDeploymentServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()
	deployed, err := client.Deploy(ctx, connect.NewRequest(&deploymentv1.DeployRequest{Id: "connect-real-kubo", Name: "reference.eth", ComputeReference: "compute://manager/devnet-reference"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := deployed.Msg.GetDeployment(); got.GetState() != "deployed" || got.GetComputeReference() != "compute://manager/devnet-reference" || got.GetContentUri() == "" {
		t.Fatalf("deployment receipt=%+v", got)
	}
	resolved, err := client.Resolve(ctx, connect.NewRequest(&deploymentv1.ResolveRequest{Id: "connect-real-kubo"}))
	if err != nil || string(resolved.Msg.GetContent()) != string(expected) {
		got := 0
		if resolved != nil && resolved.Msg != nil {
			got = len(resolved.Msg.GetContent())
		}
		t.Fatalf("resolved=%d bytes want=%d err=%v", got, len(expected), err)
	}
	renewed, err := client.Renew(ctx, connect.NewRequest(&deploymentv1.RenewRequest{Id: "connect-real-kubo"}))
	if err != nil || renewed.Msg.GetDeployment().GetState() != "renewed" {
		t.Fatalf("renew receipt=%v err=%v", renewed, err)
	}
	if _, err := client.Teardown(ctx, connect.NewRequest(&deploymentv1.TeardownRequest{Id: "connect-real-kubo"})); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Records.Get(ctx, "connect-real-kubo")
	if err != nil || stored.State != "torn-down" {
		t.Fatalf("stored receipt=%+v err=%v", stored, err)
	}
	if _, err := os.Stat(repoRoot); !os.IsNotExist(err) {
		t.Fatalf("empty Kubo repo root remains after Connect teardown: %v", err)
	}
	t.Logf("Connect lifecycle deployed %s (%d exact bytes), renewed it, persisted the Compute Manager reference, and removed its Kubo repo", deployed.Msg.GetDeployment().GetContentUri(), len(expected))
}
