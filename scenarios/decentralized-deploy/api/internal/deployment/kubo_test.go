package deployment

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type recordingKubo struct {
	calls [][]string
}

func (r *recordingKubo) Run(_ context.Context, _ string, repo string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if len(args) > 0 && args[0] == "init" {
		return nil, os.WriteFile(filepath.Join(repo, "config"), []byte("{}"), 0o600)
	}
	switch {
	case len(args) > 0 && args[0] == "add":
		return []byte("bafy-root\n"), nil
	case len(args) > 1 && args[0] == "dag" && args[1] == "import":
		return []byte(`[{"Root":{"Cid":{"/":"bafy-root"}}}]`), nil
	case len(args) > 1 && args[0] == "cat":
		return []byte("reference content"), nil
	default:
		return []byte("ok\n"), nil
	}
}

func TestKuboCARImportRequiresAReportedPinnedRoot(t *testing.T) {
	rootCID, err := importedRootCID([]byte(`[{"Root":{"Cid":{"/":"bafy-root"}}}]`))
	if err != nil || rootCID != "bafy-root" {
		t.Fatalf("root=%q err=%v", rootCID, err)
	}
	if _, err := importedRootCID([]byte(`[{"Root":{"Cid":{"/":"bafy-root"},"PinErrorMsg":"missing block"}}]`)); err == nil || !strings.Contains(err.Error(), "did not pin") {
		t.Fatalf("pin error was not rejected: %v", err)
	}
}

func TestKuboAdapterLifecycleUsesOwnedOfflineRepo(t *testing.T) {
	root := t.TempDir()
	artifactDir := filepath.Join(root, "reference")
	if err := os.Mkdir(artifactDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "index.html"), []byte("site"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "kubo-repo")
	runner := &recordingKubo{}
	adapter := KuboAdapter{Binary: "/managed/kubo/ipfs", Repo: repo, Entry: "index.html", Gate: Gate{Mode: GateStub}, Runner: runner}
	content, err := adapter.Deploy(context.Background(), Artifact{Path: artifactDir, Name: "reference"})
	if err != nil || content.CID != "bafy-root" || content.URI != "ipfs://bafy-root" {
		t.Fatalf("content=%+v err=%v", content, err)
	}
	body, err := adapter.Resolve(context.Background(), content)
	if err != nil || string(body) != "reference content" {
		t.Fatalf("body=%q err=%v", body, err)
	}
	if err := adapter.Renew(context.Background(), content); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Teardown(context.Background(), content); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(repo); !os.IsNotExist(err) {
		t.Fatalf("owned repo remains after teardown: %v", err)
	}
	if len(runner.calls) != 10 {
		t.Fatalf("Kubo calls=%v", runner.calls)
	}
	if !reflect.DeepEqual(runner.calls[0], []string{"init", "--empty-repo=true"}) || !reflect.DeepEqual(runner.calls[1], []string{"config", "Bootstrap", "[]", "--json"}) || !reflect.DeepEqual(runner.calls[2], []string{"config", "Addresses.Swarm", "[]", "--json"}) || !reflect.DeepEqual(runner.calls[3], []string{"config", "Routing.Type", "none"}) {
		t.Fatalf("offline configuration commands=%v", runner.calls[:4])
	}
	if !reflect.DeepEqual(runner.calls[4], []string{"add", "--recursive", "--quieter", "--cid-version=1", artifactDir}) {
		t.Fatalf("add command=%v", runner.calls[4])
	}
	if runner.calls[5][0] != "pin" || runner.calls[6][0] != "cat" || runner.calls[7][0] != "pin" || runner.calls[8][0] != "pin" || runner.calls[9][0] != "repo" {
		t.Fatalf("lifecycle order=%v", runner.calls)
	}
}

func TestKuboAdapterRejectsUnownedRepositoriesAndArtifactSymlinks(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "unowned")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, "artifact")
	if err := os.Mkdir(artifact, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(artifact, "leak")); err != nil {
		t.Fatal(err)
	}
	adapter := KuboAdapter{Binary: "/managed/kubo/ipfs", Repo: repo, Entry: "index.html", Gate: Gate{Mode: GateStub}, Runner: &recordingKubo{}}
	if _, err := adapter.Deploy(context.Background(), Artifact{Path: artifact, Name: "site"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink error=%v", err)
	}
	if _, err := adapter.Deploy(context.Background(), Artifact{Path: t.TempDir(), Name: "site"}); err == nil || !strings.Contains(err.Error(), "unowned") {
		t.Fatalf("unowned repo error=%v", err)
	}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, kuboOwnershipMarker), []byte(kuboOwnershipValue), 0o600); err != nil {
		t.Fatal(err)
	}
	linkedRepo := filepath.Join(root, "linked-repo")
	if err := os.Symlink(target, linkedRepo); err != nil {
		t.Fatal(err)
	}
	adapter.Repo = linkedRepo
	if _, err := adapter.Deploy(context.Background(), Artifact{Path: t.TempDir(), Name: "site"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("repository symlink error=%v", err)
	}
}

func TestKuboAdapterIsolatesRepositoriesPerDeployment(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "reference")
	if err := os.Mkdir(artifact, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "index.html"), []byte("site"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := KuboAdapter{Binary: "/managed/kubo/ipfs", Repo: filepath.Join(root, "kubo-repos"), Entry: "index.html", Gate: Gate{Mode: GateStub}, Runner: &recordingKubo{}}
	first, err := adapter.Deploy(context.Background(), Artifact{Path: artifact, Name: "reference", DeploymentID: "one"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Deploy(context.Background(), Artifact{Path: artifact, Name: "reference", DeploymentID: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if first.RepoID == "" || second.RepoID == "" || first.RepoID == second.RepoID {
		t.Fatalf("per-deployment repo ids are not distinct: first=%q second=%q", first.RepoID, second.RepoID)
	}
	firstRepo, err := adapter.contentRepo(first.RepoID)
	if err != nil {
		t.Fatal(err)
	}
	secondRepo, err := adapter.contentRepo(second.RepoID)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Teardown(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(firstRepo); !os.IsNotExist(err) {
		t.Fatalf("first deployment repo still exists after teardown: %v", err)
	}
	if _, err := os.Stat(secondRepo); err != nil {
		t.Fatalf("tearing down first deployment affected second: %v", err)
	}
	if _, err := adapter.Resolve(context.Background(), second); err != nil {
		t.Fatalf("second deployment no longer resolves: %v", err)
	}
	if err := adapter.Teardown(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(adapter.Repo); !os.IsNotExist(err) {
		t.Fatalf("empty shared repo root remains after last deployment teardown: %v", err)
	}
}

func TestKuboAdapterRealOfflineDevnet(t *testing.T) {
	binary := os.Getenv("CI1_KUBO_BINARY")
	if binary == "" {
		t.Skip("set CI1_KUBO_BINARY to exercise the pinned local Kubo executable")
	}
	root := t.TempDir()
	artifact := filepath.Join(root, "reference")
	if err := os.Mkdir(artifact, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("CI1 local Kubo devnet\n")
	if err := os.WriteFile(filepath.Join(artifact, "index.html"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := KuboAdapter{Binary: binary, Repo: filepath.Join(root, "repo"), Entry: "index.html", Gate: Gate{Mode: GateStub}}
	content, err := adapter.Deploy(context.Background(), Artifact{Path: artifact, Name: "reference"})
	if err != nil {
		t.Fatal(err)
	}
	repo, err := adapter.contentRepo(content.RepoID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.run(context.Background(), repo, "repo", "gc"); err != nil {
		t.Fatalf("collect unpinned blocks before exact retrieval: %v", err)
	}
	got, err := adapter.Resolve(context.Background(), content)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("retrieved=%q err=%v", got, err)
	}
	if err := adapter.Renew(context.Background(), content); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Teardown(context.Background(), content); err != nil {
		t.Fatal(err)
	}
	t.Logf("offline Kubo devnet deployed and retrieved %d bytes at %s; renew and teardown succeeded", len(got), content.CID)
}

func TestKuboAdapterRealOfflineCARImport(t *testing.T) {
	binary := os.Getenv("CI1_KUBO_BINARY")
	if binary == "" {
		t.Skip("set CI1_KUBO_BINARY to exercise the pinned local Kubo executable")
	}
	root := t.TempDir()
	carPath := filepath.Join("testdata", "site.car")
	archive, err := os.ReadFile(carPath)
	if err != nil || len(archive) == 0 {
		t.Fatalf("read CAR fixture (%d bytes): %v", len(archive), err)
	}
	expected, err := os.ReadFile(filepath.Join("testdata", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	adapter := KuboAdapter{Binary: binary, Repo: filepath.Join(root, "repo"), Gate: Gate{Mode: GateStub}}
	content, err := adapter.Deploy(context.Background(), Artifact{Path: carPath, Name: "reference"})
	if err != nil {
		t.Fatal(err)
	}
	repo, err := adapter.contentRepo(content.RepoID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.run(context.Background(), repo, "repo", "gc"); err != nil {
		t.Fatalf("collect unpinned blocks before exact CAR retrieval: %v", err)
	}
	got, err := adapter.Resolve(context.Background(), content)
	if err != nil || string(got) != string(expected) {
		t.Fatalf("retrieved=%d bytes, want=%d, err=%v", len(got), len(expected), err)
	}
	if err := adapter.Renew(context.Background(), content); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Teardown(context.Background(), content); err != nil {
		t.Fatal(err)
	}
	t.Logf("offline Kubo imported a %d-byte CAR, pinned root %s, retrieved exact %d-byte entry after GC, renewed and tore down", len(archive), content.CID, len(got))
}

func TestKuboProviderRefusalHookPreventsOfflineEffects(t *testing.T) {
	binary := os.Getenv("CI1_KUBO_BINARY")
	if binary == "" {
		t.Skip("set CI1_KUBO_BINARY to exercise the provider refusal hook against managed offline Kubo")
	}
	root := t.TempDir()
	artifact := filepath.Join(root, "reference")
	if err := os.Mkdir(artifact, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "index.html"), []byte("synthetic refusal fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "kubo-repos")
	faults := NewFaults()
	if err := faults.Set("kubo", FaultRefuse); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(Gate{Mode: GateStub}, faults, KuboAdapter{Binary: binary, Repo: repo, Entry: "index.html", Gate: Gate{Mode: GateStub}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Deploy(context.Background(), "refused-kubo", Artifact{Path: artifact, Name: "reference"}, "local-reference"); err == nil || !strings.Contains(err.Error(), "provider kubo refused") {
		t.Fatalf("refusal error=%v", err)
	}
	if _, err := os.Stat(repo); !os.IsNotExist(err) {
		t.Fatalf("refused storage provider left a repository: %v", err)
	}
}
