// [REQ:GCT-OT-P1-002]
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/config"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/proposals"
	"git-control-tower/internal/reviewjobs"
	"git-control-tower/internal/stagedmessages"
	testdb "github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory/advisory_v1connect"
)

type messageTransportOwner struct {
	starts, cancels          int
	status                   string
	result                   *v1.StagedMessageSuggestion
	lastInput, lastReadInput stagedmessages.DraftInput
}

func (p *messageTransportOwner) Search(context.Context, *v1.StagedMessageSnapshot) ([]*v1.StagedMessageContext, []*v1.Omission, error) {
	return nil, nil, nil
}

func (p *messageTransportOwner) Start(_ context.Context, input stagedmessages.DraftInput, _ string) (stagedmessages.Execution, error) {
	p.starts++
	p.lastInput = input
	return stagedmessages.Execution{ID: "owner-1", Status: "running"}, nil
}

func (p *messageTransportOwner) Get(_ context.Context, _ string, input stagedmessages.DraftInput) (stagedmessages.Execution, error) {
	p.lastReadInput = input
	return stagedmessages.Execution{ID: "owner-1", Status: p.status, Result: p.result}, nil
}

func TestStagedMessageTypedTransportPreserves4096RealIndexEntriesAndCompleteResults(t *testing.T) {
	ctx, dir := context.Background(), t.TempDir()
	git := func(input string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	git("", "init", "-q")
	blob := strings.TrimSpace(git("selected staged bytes\n", "hash-object", "-w", "--stdin"))
	var entries strings.Builder
	for i := 0; i < stagedmessages.MaxManifestFiles; i++ {
		fmt.Fprintf(&entries, "100644 %s\tscenarios/scope-%02d/%sfile-%04d.go\n", blob, i%32, strings.Repeat("long-directory/", 12), i)
	}
	git(entries.String(), "update-index", "--index-info")
	beforeIndex := git("", "ls-files", "--stage", "-z")
	repos := newTestRepoStore(t)
	record, err := repos.Upsert(ctx, RepoRecord{Path: dir, Name: "large-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := reviewjobs.New(testdb.OpenSQLiteMemory(t))
	if err != nil {
		t.Fatal(err)
	}
	owner := &messageTransportOwner{status: "running"}
	s := &Server{repos: NewRepoService(repos, NewFakeGitRunner()), stagedMessages: stagedmessages.New(store, stagedmessages.GitSnapshotter{}, nil, owner)}
	path, handler := rpc.NewAdvisoryServiceHandler(advisoryConnectServer{server: s}, connect.WithInterceptors(policygate.NewInterceptor(config.PolicyConfig{AgentAccess: config.AgentAccessDeny}, nil)))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := rpc.NewAdvisoryServiceClient(server.Client(), server.URL)
	repoID := repositoryIDFor(&record)
	started := time.Now()
	collection, err := client.CollectStagedMessage(ctx, connect.NewRequest(&v1.CollectStagedMessageRequest{RepositoryId: repoID}))
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Msg.Snapshot.Files) != stagedmessages.MaxManifestFiles || owner.starts != 0 {
		t.Fatalf("typed full selection lost: files=%d starts=%d", len(collection.Msg.Snapshot.Files), owner.starts)
	}
	collectionTime := time.Since(started)
	job, err := client.StartStagedMessage(ctx, connect.NewRequest(&v1.StartStagedMessageRequest{RepositoryId: repoID, CollectionId: collection.Msg.Id, SnapshotDigest: collection.Msg.Snapshot.Digest, RequestKey: "full-selection"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(owner.lastInput.Snapshot.Files) != stagedmessages.MaxManifestFiles || owner.lastInput.FormatVersion != 2 || owner.starts != 1 {
		t.Fatal("explicit owner admission lost canonical input")
	}
	reattached, err := client.GetStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, RequestKey: "full-selection"}))
	if err != nil || reattached.Msg.Id != job.Msg.Id || len(owner.lastReadInput.Snapshot.Files) != stagedmessages.MaxManifestFiles {
		t.Fatal("typed reattachment lost canonical input", err)
	}
	cv := &v1.StagedMessageCoverage{TotalEdits: stagedmessages.MaxManifestFiles}
	for _, file := range collection.Msg.Snapshot.Files {
		if file.OmissionReason != "" {
			cv.UncoveredPaths = append(cv.UncoveredPaths, file.Path)
		} else {
			cv.CoveredPaths = append(cv.CoveredPaths, file.Path)
			cv.MatchedEdits++
		}
	}
	cv.Partial = len(cv.UncoveredPaths) > 0
	owner.status, owner.result = "succeeded", &v1.StagedMessageSuggestion{Title: "Update staged source groups", Coverage: cv}
	done, err := client.WaitStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, JobId: job.Msg.Id, TimeoutSeconds: 1}))
	if err != nil || done.Msg.Result == nil || done.Msg.Status != "succeeded" || !done.Msg.Applicable {
		t.Fatalf("typed result failed: %v %v", done, err)
	}
	coverage := done.Msg.Result.Coverage
	if len(coverage.CoveredPaths)+len(coverage.UncoveredPaths) != stagedmessages.MaxManifestFiles || len(done.Msg.Snapshot.Files) != stagedmessages.MaxManifestFiles {
		t.Fatal("typed result accounting dropped files")
	}
	if git("", "ls-files", "--stage", "-z") != beforeIndex {
		t.Fatal("message RPCs changed fixture index")
	}
	distant := collection.Msg.Snapshot.Files[len(collection.Msg.Snapshot.Files)-2]
	if distant.OmissionReason == "" {
		t.Fatal("distant drift fixture must be omitted")
	}
	changed := strings.TrimSpace(git("different bytes\n", "hash-object", "-w", "--stdin"))
	git("100644 "+changed+"\t"+distant.Path+"\n", "update-index", "--index-info")
	check, err := client.CheckStagedMessage(ctx, connect.NewRequest(&v1.CheckStagedMessageRequest{RepositoryId: repoID, SnapshotDigest: collection.Msg.Snapshot.Digest}))
	if err != nil || check.Msg.Applicable {
		t.Fatal("typed drift missed omitted tail identity", err)
	}
	stale, err := client.GetStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, JobId: job.Msg.Id}))
	if err != nil || stale.Msg.Applicable || stale.Msg.Result == nil {
		t.Fatal("typed stale result was lost", err)
	}
	git("100644 "+changed+"\t4097th.go\n", "update-index", "--index-info")
	_, err = client.CollectStagedMessage(ctx, connect.NewRequest(&v1.CollectStagedMessageRequest{RepositoryId: repoID}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || !strings.Contains(err.Error(), "4096") {
		t.Fatalf("typed4097 bound: %v", err)
	}
	t.Logf("real index manifest=%d; collection latency=%s; typed result covered=%d uncovered=%d", len(collection.Msg.Snapshot.Files), collectionTime, len(coverage.CoveredPaths), len(coverage.UncoveredPaths))
}

func TestStagedMessageRetainedReaderUses4096ExactIdentitiesWithoutOpeningOmittedBytes(t *testing.T) {
	ctx := context.Background()
	store, err := proposals.NewStore(testdb.OpenSQLiteMemory(t))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &v1.StagedMessageSnapshot{RepositoryId: "repo"}
	for source := 0; source < 4; source++ {
		p := proposals.Proposal{ID: fmt.Sprintf("source-%d", source), RepositoryID: "repo", State: proposals.StateWithdrawn, Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(), Message: proposals.Message{Subject: "Retained scoped update"}}
		for i := 0; i < 1024; i++ {
			path := fmt.Sprintf("scenarios/scope-%d/file-%04d.go", source, i)
			p.Files = append(p.Files, proposals.File{Path: path, BaseBlobID: "base", BlobID: "candidate", BaseMode: "100644", Mode: "100644", Captured: true})
			snapshot.Files = append(snapshot.Files, &v1.StagedMessageFile{Path: path, OldBlobId: "base", NewBlobId: "candidate", OldMode: "100644", NewMode: "100644", OmissionReason: "detail omitted"})
		}
		if err := store.Insert(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	// No blob table entries are stored. Opening ReviewFile would fail and emit
	// unavailable-byte omissions; exact capture identities need no object read.
	reader := stagedRetainedReader{server: &Server{proposalService: proposals.NewService(store)}}
	messages, omissions, err := reader.Messages(ctx, "repo", snapshot)
	if err != nil || len(messages) != 4 || len(omissions) != 0 {
		t.Fatalf("exact retained metadata failed: messages=%d omissions=%v err=%v", len(messages), omissions, err)
	}
	total := 0
	for _, message := range messages {
		for _, file := range message.Files {
			if !file.Captured || file.BytesAvailable {
				t.Fatal("reader lost retained identity or claimed unavailable bytes")
			}
			total++
		}
	}
	if total != stagedmessages.MaxManifestFiles {
		t.Fatalf("retained identities truncated: %d", total)
	}
}

func (p *messageTransportOwner) Wait(ctx context.Context, id string, input stagedmessages.DraftInput, _ time.Duration) (stagedmessages.Execution, error) {
	return p.Get(ctx, id, input)
}

func (p *messageTransportOwner) Cancel(ctx context.Context, id string, input stagedmessages.DraftInput) (stagedmessages.Execution, error) {
	p.cancels++
	return p.Get(ctx, id, input)
}

func TestStagedMessageTypedTransportUsesImmutableIndexAndDurableOwner(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("git %v: %v %s", args, e, out)
		}
		return string(out)
	}
	write := func(value string) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(dir, "file.txt"), []byte(value), 0o600); e != nil {
			t.Fatal(e)
		}
	}
	git("init", "-q")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	write("base\n")
	git("add", "file.txt")
	git("commit", "-qm", "base")
	write("staged A\n")
	git("add", "file.txt")
	write("working B\n")
	beforeIndex, beforeHead := git("ls-files", "-s"), git("rev-parse", "HEAD")
	repos := newTestRepoStore(t)
	record, e := repos.Upsert(ctx, RepoRecord{Path: dir, Name: "fixture"})
	if e != nil {
		t.Fatal(e)
	}
	other, e := repos.Upsert(ctx, RepoRecord{Path: "/fake/other", Name: "other"})
	if e != nil {
		t.Fatal(e)
	}
	store, e := reviewjobs.New(testdb.OpenSQLiteMemory(t))
	if e != nil {
		t.Fatal(e)
	}
	owner := &messageTransportOwner{status: "running"}
	s := &Server{repos: NewRepoService(repos, NewFakeGitRunner()), stagedMessages: stagedmessages.New(store, stagedmessages.GitSnapshotter{}, nil, owner)}
	path, handler := rpc.NewAdvisoryServiceHandler(advisoryConnectServer{server: s}, connect.WithInterceptors(policygate.NewInterceptor(config.PolicyConfig{AgentAccess: config.AgentAccessDeny}, nil)))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := rpc.NewAdvisoryServiceClient(server.Client(), server.URL)
	repoID := repositoryIDFor(&record)
	collection, e := client.CollectStagedMessage(ctx, connect.NewRequest(&v1.CollectStagedMessageRequest{RepositoryId: repoID}))
	if e != nil {
		t.Fatal(e)
	}
	if owner.starts != 0 || collection.Msg.Status != "no_suggestions" || len(collection.Msg.Snapshot.Files) != 1 || !strings.Contains(collection.Msg.Snapshot.Files[0].Patch, "staged A") || strings.Contains(collection.Msg.Snapshot.Files[0].Patch, "working B") {
		t.Fatalf("collection %v starts=%d", collection.Msg, owner.starts)
	}
	start, e := client.StartStagedMessage(ctx, connect.NewRequest(&v1.StartStagedMessageRequest{RepositoryId: repoID, CollectionId: collection.Msg.Id, SnapshotDigest: collection.Msg.Snapshot.Digest, RequestKey: "explicit"}))
	if e != nil {
		t.Fatal(e)
	}
	recovered, e := client.GetStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, RequestKey: "explicit"}))
	if e != nil || recovered.Msg.Id != start.Msg.Id || owner.starts != 1 {
		t.Fatalf("recovery %v %v", recovered, e)
	}
	_, e = client.GetStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repositoryIDFor(&other), JobId: start.Msg.Id}))
	if connect.CodeOf(e) != connect.CodeNotFound {
		t.Fatalf("cross-repository job disclosed %v", e)
	}
	owner.status = "succeeded"
	owner.result = &v1.StagedMessageSuggestion{Title: "Update staged behavior", Coverage: &v1.StagedMessageCoverage{CoveredPaths: []string{"file.txt"}, MatchedEdits: 1, TotalEdits: 1}}
	done, e := client.WaitStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, JobId: start.Msg.Id, TimeoutSeconds: 1}))
	if e != nil || done.Msg.Result == nil || !done.Msg.Applicable {
		t.Fatalf("wait %v %v", done, e)
	}
	owner.status = "running"
	second, e := client.StartStagedMessage(ctx, connect.NewRequest(&v1.StartStagedMessageRequest{RepositoryId: repoID, CollectionId: collection.Msg.Id, SnapshotDigest: collection.Msg.Snapshot.Digest, RequestKey: "cancel"}))
	if e != nil {
		t.Fatal(e)
	}
	pending, e := client.CancelStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, JobId: second.Msg.Id}))
	if e != nil || pending.Msg.Status != "cancelling" {
		t.Fatalf("cancel %v %v", pending, e)
	}
	owner.status = "cancelled"
	cancelled, e := client.WaitStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, JobId: second.Msg.Id}))
	if e != nil || cancelled.Msg.Status != "cancelled" {
		t.Fatalf("cancel confirmation %v %v", cancelled, e)
	}
	if git("ls-files", "-s") != beforeIndex || git("rev-parse", "HEAD") != beforeHead {
		t.Fatal("advisory RPC mutated fixture repository")
	}
	git("add", "file.txt") // deliberate operator change, only in the isolated fixture.
	check, e := client.CheckStagedMessage(ctx, connect.NewRequest(&v1.CheckStagedMessageRequest{RepositoryId: repoID, SnapshotDigest: collection.Msg.Snapshot.Digest}))
	if e != nil || check.Msg.Applicable {
		t.Fatalf("stale input %v %v", check, e)
	}
	retained, e := client.GetStagedMessageJob(ctx, connect.NewRequest(&v1.StagedMessageJobRequest{RepositoryId: repoID, JobId: start.Msg.Id}))
	if e != nil || retained.Msg.Result == nil || retained.Msg.Applicable {
		t.Fatalf("stale result lost %v %v", retained, e)
	}
}
