package message

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"git-control-tower/cli/internal/callerheader"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory/advisory_v1connect"
)

type recordingOwner struct {
	advisory_v1connect.UnimplementedAdvisoryServiceHandler
	calls    []string
	start    *v1.StartStagedMessageRequest
	job      *v1.StagedMessageJobRequest
	evidence *v1.ReadStagedMessageEvidenceRequest
	caller   string
}

func (o *recordingOwner) CollectStagedMessage(_ context.Context, r *connect.Request[v1.CollectStagedMessageRequest]) (*connect.Response[v1.StagedMessageCollection], error) {
	o.calls = append(o.calls, "collect")
	o.caller = r.Header().Get("X-Vrooli-Caller")
	return connect.NewResponse(&v1.StagedMessageCollection{Id: "collection-1", Status: "no_suggestions", Snapshot: &v1.StagedMessageSnapshot{Digest: "snapshot-1", RepositoryId: r.Msg.GetRepositoryId()}}), nil
}
func (o *recordingOwner) CheckStagedMessage(_ context.Context, r *connect.Request[v1.CheckStagedMessageRequest]) (*connect.Response[v1.CheckStagedMessageResponse], error) {
	o.calls = append(o.calls, "check")
	return connect.NewResponse(&v1.CheckStagedMessageResponse{Applicable: r.Msg.GetSnapshotDigest() == "snapshot-1", CurrentSnapshotDigest: "snapshot-1"}), nil
}
func (o *recordingOwner) StartStagedMessage(_ context.Context, r *connect.Request[v1.StartStagedMessageRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	o.calls = append(o.calls, "start")
	o.start = r.Msg
	return connect.NewResponse(&v1.StagedMessageJob{Id: "job-1", Status: "queued", SnapshotDigest: r.Msg.GetSnapshotDigest()}), nil
}
func (o *recordingOwner) GetStagedMessageJob(_ context.Context, r *connect.Request[v1.StagedMessageJobRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	o.calls = append(o.calls, "status")
	o.job = r.Msg
	return connect.NewResponse(&v1.StagedMessageJob{Id: "job-1", Status: "running"}), nil
}
func (o *recordingOwner) WaitStagedMessageJob(_ context.Context, r *connect.Request[v1.StagedMessageJobRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	o.calls = append(o.calls, "wait")
	o.job = r.Msg
	return connect.NewResponse(&v1.StagedMessageJob{Id: "job-1", Status: "running"}), nil
}
func (o *recordingOwner) CancelStagedMessageJob(_ context.Context, r *connect.Request[v1.StagedMessageJobRequest]) (*connect.Response[v1.StagedMessageJob], error) {
	o.calls = append(o.calls, "cancel")
	o.job = r.Msg
	return connect.NewResponse(&v1.StagedMessageJob{Id: "job-1", Status: "cancelling", CancelRequested: true}), nil
}

func (o *recordingOwner) ReadStagedMessageEvidence(_ context.Context, r *connect.Request[v1.ReadStagedMessageEvidenceRequest]) (*connect.Response[v1.ReadStagedMessageEvidenceResponse], error) {
	o.calls = append(o.calls, "evidence")
	o.evidence = r.Msg
	o.caller = r.Header().Get("X-Vrooli-Caller")
	return connect.NewResponse(&v1.ReadStagedMessageEvidenceResponse{SnapshotDigest: r.Msg.GetSnapshotDigest(), NextCursor: 128, TotalFiles: 285, Receipt: "page-receipt", Files: []*v1.StagedMessageEvidenceFile{{FileId: 65, Path: "source.go", Patch: "@@ -1 +1 @@\n-old\n+staged\n", CompletePatch: true}}}), nil
}

func fixture(t *testing.T) (*recordingOwner, cliapp.SubcommandGroup) {
	t.Helper()
	owner := &recordingOwner{}
	path, handler := advisory_v1connect.NewAdvisoryServiceHandler(owner)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	before := clientFactory
	t.Cleanup(func() { clientFactory = before })
	clientFactory = func(*cliapp.ScenarioApp) advisory_v1connect.AdvisoryServiceClient {
		return advisory_v1connect.NewAdvisoryServiceClient(server.Client(), server.URL, connect.WithInterceptors(callerheader.New()))
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, raw)
	if err != nil {
		t.Fatal(err)
	}
	return owner, group
}
func invoke(group cliapp.SubcommandGroup, name string, args ...string) (string, error) {
	for _, command := range group.Subcommands {
		if command.Name == name {
			out := &bytes.Buffer{}
			ctx, err := cliapp.NewTestRunContextFromArgs(command.Args, args, nil, out, out)
			if err != nil {
				return out.String(), err
			}
			err = command.RunCtx(ctx)
			return out.String(), err
		}
	}
	panic("unknown test command")
}

func TestMessageCollectCheckNeverStartAndUseTypedCallerTransport(t *testing.T) {
	owner, group := fixture(t)
	out, err := invoke(group, "collect", "--repo-id", "42", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var collection v1.StagedMessageCollection
	if err = json.Unmarshal([]byte(out), &collection); err != nil {
		t.Fatal(err)
	}
	if collection.GetId() != "collection-1" || collection.GetSnapshot().GetRepositoryId() != "42" || owner.caller == "" {
		t.Fatalf("typed collection or caller metadata lost: %s", out)
	}
	if _, err := invoke(group, "check", "--snapshot", "snapshot-1", "--json"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(owner.calls, ",") != "collect,check" {
		t.Fatalf("read steps started agent: %v", owner.calls)
	}
}

func TestMessageStartRequiresExplicitImmutableInputAndRequestKey(t *testing.T) {
	owner, group := fixture(t)
	for _, args := range [][]string{{}, {"--collection", "c", "--snapshot", "s"}, {"--collection", "c", "--snapshot", "s", "--request-key", " "}, {"--collection", "c", "--snapshot", "s", "--request-key", "k", "--context-id", "a", "--context-id", "a"}} {
		if _, err := invoke(group, "start", args...); err == nil {
			t.Fatalf("invalid admission accepted: %v", args)
		}
	}
	if len(owner.calls) != 0 {
		t.Fatal("invalid arguments reached owner")
	}
	out, err := invoke(group, "start", "--collection", "collection-1", "--snapshot", "snapshot-1", "--request-key", "stable-1", "--context-id", "ctx-1", "--context-id", "ctx-2", "--note", "why\nchange", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if owner.start.GetRequestKey() != "stable-1" || len(owner.start.GetSelectedContextIds()) != 2 || owner.start.GetOperatorNote() != "why\nchange" || !strings.Contains(out, "queued") {
		t.Fatalf("explicit input lost: %+v %s", owner.start, out)
	}
}

func TestMessageReconnectWaitAndCancelNeverRestartOrPoll(t *testing.T) {
	owner, group := fixture(t)
	if _, err := invoke(group, "status", "--request-key", "stable-1", "--json"); err != nil {
		t.Fatal(err)
	}
	if owner.job.GetRequestKey() != "stable-1" || owner.job.GetJobId() != "" {
		t.Fatal("reconnect key lost")
	}
	if _, err := invoke(group, "wait", "job-1", "--timeout", "55", "--json"); err != nil {
		t.Fatal(err)
	}
	if owner.job.GetTimeoutSeconds() != 55 {
		t.Fatal("wait deadline lost")
	}
	out, err := invoke(group, "cancel", "job-1", "--json")
	if err != nil || !strings.Contains(out, "cancelling") {
		t.Fatalf("cancel falsely terminal: %s %v", out, err)
	}
	if strings.Join(owner.calls, ",") != "status,wait,cancel" {
		t.Fatalf("command repeated/started owner calls: %v", owner.calls)
	}
	for _, args := range [][]string{{"job-1", "--timeout", "0"}, {"job-1", "--timeout", "56"}, {"job-1", "--timeout", "not-a-number"}} {
		if _, err := invoke(group, "wait", args...); err == nil {
			t.Fatalf("invalid wait accepted: %v", args)
		}
	}
	if _, err := invoke(group, "status"); err == nil {
		t.Fatal("missing job identity accepted")
	}
	if len(owner.calls) != 3 {
		t.Fatal("invalid lifecycle arguments reached owner")
	}
}

func TestMessageContinuationPreservesParentAndUsesExplicitNewKey(t *testing.T) {
	owner, group := fixture(t)
	_, err := invoke(group, "start", "--repo-id", "42", "--continue-from", "partial-job", "--snapshot", "snapshot-1", "--request-key", "investigate-1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if owner.start.GetContinuationOfJobId() != "partial-job" || owner.start.GetRequestKey() != "investigate-1" || owner.start.GetSnapshotDigest() != "snapshot-1" || owner.start.GetRepositoryId() != "42" || owner.start.GetCollectionId() != "" {
		t.Fatalf("continuation identity lost: %v", owner.start)
	}
	for _, args := range [][]string{
		{"--continue-from", "partial-job", "--snapshot", "snapshot-1"},
		{"--continue-from", "partial-job", "--snapshot", "snapshot-1", "--request-key", "new", "--collection", "different"},
		{"--continue-from", "partial-job", "--snapshot", "snapshot-1", "--request-key", "new", "--note", "changed intent"},
		{"--continue-from", "partial-job", "--snapshot", "snapshot-1", "--request-key", "new", "--context-id", "different"},
	} {
		if _, err := invoke(group, "start", args...); err == nil {
			t.Fatalf("ambiguous continuation accepted: %v", args)
		}
	}
	if strings.Join(owner.calls, ",") != "start" {
		t.Fatalf("invalid continuation reached owner: %v", owner.calls)
	}
}

func TestMessageEvidenceReturnsOneTypedPageWithoutStartingOrFollowing(t *testing.T) {
	owner, group := fixture(t)
	out, err := invoke(group, "evidence", "investigation-job", "--repo-id", "42", "--snapshot", "snapshot-1", "--cursor", "64", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var page v1.ReadStagedMessageEvidenceResponse
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatal(err)
	}
	if page.GetSnapshotDigest() != "snapshot-1" || page.GetNextCursor() != 128 || page.GetTotalFiles() != 285 || page.GetComplete() || page.GetReceipt() != "page-receipt" || len(page.GetFiles()) != 1 || !page.GetFiles()[0].GetCompletePatch() || page.GetFiles()[0].GetFileId() != 65 {
		t.Fatalf("page identity or coverage lost: %s", out)
	}
	if owner.evidence.GetJobId() != "investigation-job" || owner.evidence.GetRepositoryId() != "42" || owner.evidence.GetCursor() != 64 || owner.evidence.GetMaxBytes() != 24576 || owner.caller == "" || strings.Join(owner.calls, ",") != "evidence" {
		t.Fatalf("page read changed identity, lost caller or followed cursor: %v %v", owner.evidence, owner.calls)
	}
	for _, args := range [][]string{
		{"investigation-job"},
		{"--snapshot", "snapshot-1"},
		{"investigation-job", "--snapshot", "snapshot-1", "--cursor", "-1"},
		{"investigation-job", "--snapshot", "snapshot-1", "--cursor", "4097"},
		{"investigation-job", "--snapshot", "snapshot-1", "--max-bytes", "0"},
		{"investigation-job", "--snapshot", "snapshot-1", "--max-bytes", "49153"},
	} {
		if _, err := invoke(group, "evidence", args...); err == nil {
			t.Fatalf("invalid evidence read accepted: %v", args)
		}
	}
	if strings.Join(owner.calls, ",") != "evidence" {
		t.Fatalf("invalid evidence read reached owner: %v", owner.calls)
	}
}
func TestMessageManifestCoversTypedAdvisorySurface(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, v1.File_git_control_tower_v1_advisory_advisory_proto, "AdvisoryService")
}

func TestEvidenceInventoryAndTargetedReadsRemainSingleTypedOperations(t *testing.T) {
	owner, group := fixture(t)
	if _, err := invoke(group, "evidence", "job-1", "--snapshot", "snapshot-1", "--inventory", "--cursor", "64", "--json"); err != nil {
		t.Fatal(err)
	}
	if !owner.evidence.InventoryOnly || owner.evidence.Cursor != 64 || len(owner.calls) != 1 {
		t.Fatal("inventory did not reach typed owner once")
	}
	if _, err := invoke(group, "evidence", "job-1", "--snapshot", "snapshot-1", "--file-id", "1", "--file-id", "4096", "--json"); err != nil {
		t.Fatal(err)
	}
	if owner.evidence.InventoryOnly || len(owner.evidence.FileIds) != 2 || owner.evidence.FileIds[1] != 4096 || len(owner.calls) != 2 {
		t.Fatal("targeted IDs lost or extra calls")
	}
	for _, args := range [][]string{{"--inventory", "--file-id", "1"}, {"--cursor", "1", "--file-id", "1"}, {"--file-id", "0"}, {"--file-id", "4097"}, {"--file-id", "1", "--file-id", "1"}} {
		args = append([]string{"job-1", "--snapshot", "snapshot-1"}, args...)
		if _, err := invoke(group, "evidence", args...); err == nil {
			t.Fatalf("invalid selection accepted: %v", args)
		}
	}
	if len(owner.calls) != 2 {
		t.Fatal("invalid arguments reached owner")
	}
}
