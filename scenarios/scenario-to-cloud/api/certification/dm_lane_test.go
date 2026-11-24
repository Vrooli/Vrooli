package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type dmLaneProfileReaderFake struct {
	pages    map[string]*profilesv1.ListProfilesResponse
	requests []*profilesv1.ListProfilesRequest
}

func (f *dmLaneProfileReaderFake) ListProfiles(_ context.Context, req *connect.Request[profilesv1.ListProfilesRequest]) (*connect.Response[profilesv1.ListProfilesResponse], error) {
	f.requests = append(f.requests, req.Msg)
	return connect.NewResponse(f.pages[req.Msg.GetPageToken()]), nil
}

type dmLaneOperationReaderFake struct {
	operations map[string]*releasesv1.ReleaseOperation
	calls      []string
	pages      map[string]*releasesv1.ListReleaseOperationsByProfileResponse
	listCalls  []*releasesv1.ListReleaseOperationsByProfileRequest
}

func (f *dmLaneOperationReaderFake) ListOperationsByProfile(_ context.Context, req *connect.Request[releasesv1.ListReleaseOperationsByProfileRequest]) (*connect.Response[releasesv1.ListReleaseOperationsByProfileResponse], error) {
	f.listCalls = append(f.listCalls, req.Msg)
	return connect.NewResponse(f.pages[req.Msg.GetProfileId()+"|"+req.Msg.GetPageToken()]), nil
}

func (f *dmLaneOperationReaderFake) GetOperation(_ context.Context, req *connect.Request[releasesv1.GetReleaseOperationRequest]) (*connect.Response[releasesv1.GetReleaseOperationResponse], error) {
	f.calls = append(f.calls, req.Msg.GetOperationId())
	return connect.NewResponse(&releasesv1.GetReleaseOperationResponse{Operation: f.operations[req.Msg.GetOperationId()]}), nil
}

func TestReadDMLaneVerdictsUsesGeneratedReadbackOnlyAndPaginatesBothIndexes(t *testing.T) {
	profiles := &dmLaneProfileReaderFake{pages: map[string]*profilesv1.ListProfilesResponse{
		"":     {Profiles: []*profilesv1.Profile{{Id: "p1", Scenario: "tool-sites"}}, NextPageToken: "next"},
		"next": {Profiles: []*profilesv1.Profile{{Id: "p2", Scenario: "web-console"}}},
	}}
	operations := &dmLaneOperationReaderFake{pages: map[string]*releasesv1.ListReleaseOperationsByProfileResponse{
		"p1|":        {OperationIds: []string{"op-1"}, NextPageToken: "next-op"},
		"p1|next-op": {OperationIds: []string{"op-2"}},
		"p2|":        {},
	}, operations: map[string]*releasesv1.ReleaseOperation{
		"op-1": {OperationId: "op-1", ProfileId: "p1"},
		"op-2": {OperationId: "op-2", ProfileId: "p1"},
	}}
	got, err := ReadDMLaneVerdicts(context.Background(), profiles, operations)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles.requests) != 2 || profiles.requests[0].GetPageSize() != 100 || profiles.requests[1].GetPageToken() != "next" {
		t.Fatalf("profile discovery requests = %+v", profiles.requests)
	}
	if len(operations.listCalls) != 3 || operations.listCalls[0].GetPageSize() != 100 || operations.listCalls[1].GetPageToken() != "next-op" || operations.listCalls[2].GetProfileId() != "p2" {
		t.Fatalf("operation index requests = %+v", operations.listCalls)
	}
	if len(operations.calls) != 2 || operations.calls[0] != "op-1" || operations.calls[1] != "op-2" {
		t.Fatalf("operation readback calls = %v", operations.calls)
	}
	if len(got) != 2 || got[0].ProfileID != "p1" || got[1].ProfileID != "p2" || got[1].Verdict != DMLaneBlocked {
		t.Fatalf("profile verdicts = %+v", got)
	}
}

func TestProjectDMLaneVerdictsMissingOperationAndTeardownContract(t *testing.T) {
	got, err := ProjectDMLaneVerdicts([]*profilesv1.Profile{{Id: "tool-profile", Scenario: "tool-sites"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Verdict != DMLaneBlocked || got[0].Stages[0].Status != DMLaneNotRun || got[0].Stages[3].Status != DMLaneBlocked {
		t.Fatalf("missing operation verdict = %+v", got)
	}
	if got[0].Stages[3].Reason != "no DM teardown receipt is present" {
		t.Fatalf("teardown gap = %+v", got[0].Stages[3])
	}
}

func TestProjectDMLaneVerdictsUseGeneratedOperationEvidence(t *testing.T) {
	profile := &profilesv1.Profile{Id: "tool-profile", Scenario: "tool-sites"}
	op := &releasesv1.ReleaseOperation{
		OperationId: "op-7", ProfileId: "tool-profile", Status: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_COMPLETE,
		UpdatedAt: timestamppb.New(timestamppb.Now().AsTime()), CloudComputeManagerInstanceId: "cm-instance-7", CloudAcquireIdempotencyKey: "dm-acquire:op-7",
		CloudStep: &releasesv1.ReleaseCloudStep{OperationId: "op-7", DeploymentId: "dep-7", ReceiptJson: `{"receipt":"owner"}`, VerificationState: "passed", VerificationKey: "verify:op-7", VerificationReceiptJson: `{"healthy":true}`},
	}
	got, err := ProjectDMLaneVerdicts([]*profilesv1.Profile{profile}, []*releasesv1.ReleaseOperation{op})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].OperationID != "op-7" || got[0].Stages[0].Status != DMLaneBlocked || got[0].Stages[1].Status != DMLanePassed || got[0].Stages[2].Status != DMLanePassed || got[0].Stages[3].Status != DMLaneBlocked || got[0].Verdict != DMLaneBlocked {
		t.Fatalf("operation projection = %+v", got[0])
	}
	if got[0].Stages[1].EvidenceRef != "deployment-manager:operation:op-7:cloud-step:op-7" || got[0].Stages[2].EvidenceRef != "deployment-manager:operation:op-7:verification:verify:op-7" {
		t.Fatalf("operation evidence refs = %+v", got[0].Stages)
	}
}

func TestProjectDMLaneTeardownRequiresExactReleasedCMReceipt(t *testing.T) {
	profile := &profilesv1.Profile{Id: "tool-profile", Scenario: "tool-sites"}
	op := &releasesv1.ReleaseOperation{OperationId: "release-op-release-7", ReleaseId: "release-7", ProfileId: profile.GetId(), CloudComputeManagerInstanceId: "cm-instance-7", CloudAcquireIdempotencyKey: "dm-acquire:release-op-release-7", CloudSpendQuoteIdempotencyKey: "dm-acquire:release-op-release-7"}
	op.CloudAcquireReceipt = validDMLaneAcquireReceipt(op)
	instance := op.CloudAcquireReceipt.GetSourceReceipt().GetInstance()
	instance.Provider, instance.ProviderInstanceId = "hetzner", "server-7"
	// Recompute the source fingerprint after setting the exact target identity.
	wire, _ := (proto.MarshalOptions{Deterministic: true}).Marshal(op.CloudAcquireReceipt.GetSourceReceipt())
	digest := sha256.Sum256(wire)
	op.CloudAcquireReceipt.SourceReceiptSha256 = "sha256:" + hex.EncodeToString(digest[:])
	op.CloudTeardownReceipt = &releasesv1.CloudTeardownReceipt{SchemaVersion: 1, OperationId: op.GetOperationId(), ReleaseId: op.GetReleaseId(), ProfileId: profile.GetId(), TeardownIdempotencyKey: "dm-release:" + op.GetOperationId(), AcquisitionReceiptSha256: op.CloudAcquireReceipt.GetSourceReceiptSha256(), ComputeManagerInstanceId: instance.GetId(), Provider: "hetzner", ProviderInstanceId: "server-7", StoredTargetSha256: op.CloudAcquireReceipt.GetStoredTargetSha256(), State: releasesv1.CloudTeardownState_CLOUD_TEARDOWN_STATE_RELEASED, OwnerReceipt: &instancev1.ReleaseReceipt{IdempotencyKey: "dm-release:" + op.GetOperationId(), Owner: "deployment-manager", InstanceId: instance.GetId(), Provider: "hetzner", ProviderInstanceId: "server-7", Action: "release", State: instancev1.ReleaseState_RELEASE_STATE_RELEASED}}
	got, err := ProjectDMLaneVerdicts([]*profilesv1.Profile{profile}, []*releasesv1.ReleaseOperation{op})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Stages[3].Status != DMLanePassed {
		t.Fatalf("exact released receipt not passed: %+v", got[0].Stages[3])
	}
	bad := proto.Clone(op).(*releasesv1.ReleaseOperation)
	bad.CloudTeardownReceipt.OwnerReceipt.ProviderInstanceId = "other-server"
	got, err = ProjectDMLaneVerdicts([]*profilesv1.Profile{profile}, []*releasesv1.ReleaseOperation{bad})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Stages[3].Status != DMLaneBlocked || got[0].Verdict == DMLanePassed {
		t.Fatalf("mismatched teardown was not blocked: %+v", got[0])
	}
}

func TestProjectDMLaneVerdictsRequiresValidTypedAcquisitionReceipt(t *testing.T) {
	profile := &profilesv1.Profile{Id: "tool-profile", Scenario: "tool-sites"}
	op := &releasesv1.ReleaseOperation{
		OperationId: "release-op-release-7", ReleaseId: "release-7", ProfileId: profile.GetId(),
		CloudComputeManagerInstanceId: "cm-instance-7", CloudAcquireIdempotencyKey: "dm-acquire:release-op-release-7", CloudSpendQuoteIdempotencyKey: "dm-acquire:release-op-release-7",
	}
	op.CloudAcquireReceipt = validDMLaneAcquireReceipt(op)
	got, err := ProjectDMLaneVerdicts([]*profilesv1.Profile{profile}, []*releasesv1.ReleaseOperation{op})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Stages[0].Status != DMLanePassed || got[0].Stages[0].Reason != "" || got[0].Stages[3].Status != DMLaneBlocked || got[0].Verdict == DMLanePassed {
		t.Fatalf("valid receipt projection = %+v", got[0])
	}

	for name, mutate := range map[string]func(*releasesv1.CloudAcquireReceipt){
		"wrong operation": func(r *releasesv1.CloudAcquireReceipt) { r.OperationId = "release-op-other" },
		"wrong profile":   func(r *releasesv1.CloudAcquireReceipt) { r.ProfileId = "other-profile" },
		"wrong key":       func(r *releasesv1.CloudAcquireReceipt) { r.AcquisitionIdempotencyKey = "other-key" },
		"bad target hash": func(r *releasesv1.CloudAcquireReceipt) { r.StoredTargetSha256 = "not-a-hash" },
		"bad source hash": func(r *releasesv1.CloudAcquireReceipt) {
			r.SourceReceiptSha256 = "sha256:" + hex.EncodeToString(make([]byte, sha256.Size))
		},
		"unapproved": func(r *releasesv1.CloudAcquireReceipt) { r.SourceReceipt.SpendReceipt.AdmissionDecision = "blocked" },
		"stale quote": func(r *releasesv1.CloudAcquireReceipt) {
			r.SourceReceipt.SpendReceipt.ExpiresAt = timestamppb.New(r.GetPersistedAt().AsTime().Add(-time.Second))
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := proto.Clone(validDMLaneAcquireReceipt(op)).(*releasesv1.CloudAcquireReceipt)
			mutate(copy)
			if name != "bad source hash" {
				wire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(copy.GetSourceReceipt())
				if err != nil {
					t.Fatal(err)
				}
				fingerprint := sha256.Sum256(wire)
				copy.SourceReceiptSha256 = "sha256:" + hex.EncodeToString(fingerprint[:])
			}
			bad := proto.Clone(op).(*releasesv1.ReleaseOperation)
			bad.CloudAcquireReceipt = copy
			got, err := ProjectDMLaneVerdicts([]*profilesv1.Profile{profile}, []*releasesv1.ReleaseOperation{bad})
			if err != nil {
				t.Fatal(err)
			}
			if got[0].Stages[0].Status != DMLaneBlocked || got[0].Verdict == DMLanePassed {
				t.Fatalf("invalid receipt was not blocked: %+v", got[0])
			}
		})
	}
}

func validDMLaneAcquireReceipt(operation *releasesv1.ReleaseOperation) *releasesv1.CloudAcquireReceipt {
	key := operation.GetCloudAcquireIdempotencyKey()
	now := time.Now().UTC()
	source := &instancev1.GetAcquireReceiptResponse{
		Instance:     &instancev1.Instance{Id: operation.GetCloudComputeManagerInstanceId(), Tenant: "deployment-manager", Provider: "hetzner", ProviderInstanceId: "server-fixture"},
		SpendReceipt: &instancev1.SpendReceipt{IdempotencyKey: key, InstanceId: operation.GetCloudComputeManagerInstanceId(), QuoteState: "approved", AdmissionDecision: "approved", ObservedAt: timestamppb.New(now.Add(-time.Minute)), ExpiresAt: timestamppb.New(now.Add(time.Hour))},
		State:        "acquired",
	}
	wire, _ := (proto.MarshalOptions{Deterministic: true}).Marshal(source)
	fingerprint := sha256.Sum256(wire)
	target := sha256.Sum256([]byte("stored-target"))
	return &releasesv1.CloudAcquireReceipt{
		SchemaVersion: 1, OperationId: operation.GetOperationId(), ReleaseId: operation.GetReleaseId(), ProfileId: operation.GetProfileId(),
		AcquisitionIdempotencyKey: key, ComputeManagerInstanceId: operation.GetCloudComputeManagerInstanceId(),
		StoredTargetSha256: "sha256:" + hex.EncodeToString(target[:]), SourceReceiptSha256: "sha256:" + hex.EncodeToString(fingerprint[:]),
		PersistedAt: timestamppb.New(now), SourceReceipt: source,
	}
}

func TestProjectDMLaneVerdictsRejectOperationProfileMismatch(t *testing.T) {
	_, err := ProjectDMLaneVerdicts([]*profilesv1.Profile{{Id: "profile-a", Scenario: "scenario-a"}}, []*releasesv1.ReleaseOperation{{OperationId: "op-b", ProfileId: "profile-b"}})
	if err == nil {
		t.Fatal("operation for undiscovered profile was accepted")
	}
}

func TestWriteDMLaneVerdictsAppendsContentAddressedSnapshot(t *testing.T) {
	dir := t.TempDir()
	verdicts := []DMLaneVerdict{{SchemaVersion: 1, ProfileID: "p1", Scenario: "tool-sites", Verdict: DMLaneBlocked, Stages: []DMLaneStageVerdict{{Stage: "teardown", Status: DMLaneBlocked, Reason: "missing generated stage"}}}}
	paths, err := WriteDMLaneVerdicts(dir, verdicts)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths = %v", paths)
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var replay []DMLaneVerdict
	if err := json.Unmarshal(raw, &replay); err != nil {
		t.Fatal(err)
	}
	if len(replay) != 1 || replay[0].Verdict != DMLaneBlocked {
		t.Fatalf("replayed verdict = %+v", replay)
	}
	if filepath.Base(paths[0]) == "" {
		t.Fatal("snapshot path has empty filename")
	}
	if _, err := WriteDMLaneVerdicts(dir, verdicts); err == nil {
		t.Fatal("duplicate immutable snapshot unexpectedly overwrote prior evidence")
	}
}
