package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	apiV1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainV1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
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

func TestProjectDMLaneLeaseRenewalPreservesCMRefusalAndBlocksUnprovenRenewal(t *testing.T) {
	profile := &profilesv1.Profile{Id: "tool-profile", Scenario: "tool-sites"}
	op := &releasesv1.ReleaseOperation{OperationId: "release-op-release-7", ReleaseId: "release-7", ProfileId: profile.GetId(), CloudComputeManagerInstanceId: "cm-instance-7", CloudAcquireIdempotencyKey: "dm-acquire:release-op-release-7", CloudSpendQuoteIdempotencyKey: "dm-acquire:release-op-release-7"}
	op.CloudAcquireReceipt = validDMLaneAcquireReceipt(op)
	cm := op.CloudAcquireReceipt.GetSourceReceipt().GetInstance()
	cm.ExpiresAt = timestamppb.New(time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC))
	cm.Provider, cm.ProviderInstanceId = "hetzner", "server-7"
	op.CloudAcquireReceipt.GetSourceReceipt().SpendReceipt.PriceProvenanceFingerprint = "sha256:fixture"
	wire, _ := (proto.MarshalOptions{Deterministic: true}).Marshal(op.CloudAcquireReceipt.GetSourceReceipt())
	digest := sha256.Sum256(wire)
	op.CloudAcquireReceipt.SourceReceiptSha256 = "sha256:" + hex.EncodeToString(digest[:])
	renewalKey := "dm-lease-renew:" + op.GetOperationId()
	requested := cm.GetExpiresAt().AsTime().Add(30 * 24 * time.Hour)
	owner := &instancev1.RenewalReceipt{IdempotencyKey: renewalKey, Owner: "deployment-manager", InstanceId: cm.GetId(), Provider: cm.GetProvider(), ProviderInstanceId: cm.GetProviderInstanceId(), PriorExpiresAt: cm.GetExpiresAt(), RequestedUntil: timestamppb.New(requested), SpendIdempotencyKey: op.CloudAcquireReceipt.GetSourceReceipt().GetSpendReceipt().GetIdempotencyKey(), QuoteFingerprint: "quote-fingerprint", State: instancev1.RenewalState_RENEWAL_STATE_REFUSED, Reason: "provider adapter has no exact external renewal capability", CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}
	op.CloudLeaseRenewalReceipt = &releasesv1.CloudLeaseRenewalReceipt{SchemaVersion: 1, OperationId: op.GetOperationId(), ReleaseId: op.GetReleaseId(), ProfileId: profile.GetId(), RenewalIdempotencyKey: renewalKey, AcquisitionReceiptSha256: op.CloudAcquireReceipt.GetSourceReceiptSha256(), ComputeManagerInstanceId: cm.GetId(), ComputeManagerProvider: cm.GetProvider(), ComputeManagerProviderInstanceId: cm.GetProviderInstanceId(), OldExpiresAt: cm.GetExpiresAt(), RequestedUntil: timestamppb.New(requested), QuoteIdempotencyKey: owner.GetSpendIdempotencyKey(), QuoteFingerprint: owner.GetQuoteFingerprint(), AuthorizationFact: "only original acquisition admission is present; no renewal quote or authorization is present", State: releasesv1.CloudLeaseRenewalState_CLOUD_LEASE_RENEWAL_STATE_BLOCKED, Reason: owner.GetReason(), PersistedAt: timestamppb.Now(), ComputeManagerReceipt: owner}
	got, err := ProjectDMLaneVerdicts([]*profilesv1.Profile{profile}, []*releasesv1.ReleaseOperation{op})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Stages[4].Status != DMLaneBlocked || got[0].Stages[4].Reason != owner.GetReason() {
		t.Fatalf("CM refusal projection = %+v", got[0].Stages[4])
	}
	// Even an owner RENEWED receipt is not a DM pass without accepted renewal
	// authorization and an exact DM new-expiry standing.
	op.CloudLeaseRenewalReceipt.ComputeManagerReceipt.State = instancev1.RenewalState_RENEWAL_STATE_RENEWED
	op.CloudLeaseRenewalReceipt.Reason = "CM says renewed"
	got, err = ProjectDMLaneVerdicts([]*profilesv1.Profile{profile}, []*releasesv1.ReleaseOperation{op})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Stages[4].Status != DMLaneBlocked {
		t.Fatalf("unproven renewal passed: %+v", got[0].Stages[4])
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
	pathsAgain, err := WriteDMLaneVerdicts(dir, verdicts)
	if err != nil || len(pathsAgain) != 1 || pathsAgain[0] != paths[0] {
		t.Fatalf("identical immutable snapshot retry should be idempotent: %v, %v", pathsAgain, err)
	}
}

func TestDMLaneGapRoutingIsOwnerAssignedIdempotentAndEvidenceSpecific(t *testing.T) {
	verdict := DMLaneVerdict{SchemaVersion: 1, ProfileID: "p1", Scenario: "tool-sites", OperationID: "op-1", Verdict: DMLaneBlocked,
		Stages: []DMLaneStageVerdict{
			{Stage: "deploy", Status: DMLanePassed, EvidenceRef: "dm:deploy"},
			{Stage: "teardown", Status: DMLaneBlocked, Reason: "no exact CM release receipt is present", EvidenceRef: "deployment-manager:operation:op-1"},
		}}
	dir := filepath.Join(t.TempDir(), "gaps")
	first, err := WriteDMLaneGaps(dir, []DMLaneVerdict{verdict})
	if err != nil {
		t.Fatal(err)
	}
	second, err := WriteDMLaneGaps(dir, []DMLaneVerdict{verdict})
	if err != nil {
		t.Fatalf("identical gap replay should be idempotent: %v", err)
	}
	if len(first) != 1 || len(second) != 1 || first[0] != second[0] {
		t.Fatalf("gap replay paths differ: %v / %v", first, second)
	}
	raw, err := os.ReadFile(first[0])
	if err != nil {
		t.Fatal(err)
	}
	var got DMLaneGap
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.ScenarioOwner != "scenario:tool-sites" || got.WorkstreamOwner != "deployment-manager" || got.WorkstreamID != "CD6" || got.WorkstreamRef != "compute-and-deployment#CD6" || got.ProfileID != "p1" || got.OperationID != "op-1" || got.Stage != "teardown" || got.Status != DMLaneBlocked || got.Reason != verdict.Stages[1].Reason || got.EvidenceRef != verdict.Stages[1].EvidenceRef {
		t.Fatalf("owner-routed gap lost responsible owner or source evidence: %+v", got)
	}
	verdict.Stages[1].Reason = "changed exact reason"
	changed, err := WriteDMLaneGaps(dir, []DMLaneVerdict{verdict})
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] == first[0] {
		t.Fatalf("changed evidence did not produce a distinct durable gap: %v", changed)
	}
}

func TestDMLaneGapRoutingFailsClosedForUnknownStage(t *testing.T) {
	_, err := ProjectDMLaneGaps([]DMLaneVerdict{{ProfileID: "p1", Scenario: "tool-sites", Stages: []DMLaneStageVerdict{{Stage: "new-stage", Status: DMLaneBlocked}}}})
	if err == nil {
		t.Fatal("gap with no registered workstream owner was silently routed")
	}
}

func TestDMLaneOwnerRouteRegistryRejectsMissingAmbiguousAndUnknownOwners(t *testing.T) {
	routes, err := loadDMLaneOwnerRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if routes.Source != "research/topics/compute-and-deployment/MISSION.md" {
		t.Fatalf("owner source = %q", routes.Source)
	}
	for _, stage := range []string{"acquire", "deploy", "verify", "teardown", "lease-renewal"} {
		if _, err := resolveDMLaneWorkstreamOwner(stage, routes); err != nil {
			t.Fatalf("maintained owner for %s did not resolve: %v", stage, err)
		}
	}
	missing := routes
	missing.StageRoutes = map[string]string{}
	for stage, owner := range routes.StageRoutes {
		missing.StageRoutes[stage] = owner
	}
	delete(missing.StageRoutes, "teardown")
	if err := validateDMLaneOwnerRoutes(missing); err == nil {
		t.Fatal("missing stage owner did not fail closed")
	}
	unknown := routes
	unknown.StageRoutes = map[string]string{}
	for stage, owner := range routes.StageRoutes {
		unknown.StageRoutes[stage] = owner
	}
	unknown.StageRoutes["renew"] = "CD6"
	if err := validateDMLaneOwnerRoutes(unknown); err == nil {
		t.Fatal("unknown stage owner route did not fail closed")
	}
	ambiguous := routes
	ambiguous.Workstreams = append(append([]dmlaneWorkstream(nil), routes.Workstreams...), routes.Workstreams[0])
	if err := validateDMLaneOwnerRoutes(ambiguous); err == nil {
		t.Fatal("duplicate workstream identity did not fail closed")
	}
	unsafe := routes
	unsafe.Workstreams = append([]dmlaneWorkstream(nil), routes.Workstreams...)
	unsafe.Workstreams[0].OwnerScenario = "deployment manager"
	if err := validateDMLaneOwnerRoutes(unsafe); err == nil {
		t.Fatal("invalid workstream owner label did not fail closed")
	}
	wrongReference := routes
	wrongReference.Workstreams = append([]dmlaneWorkstream(nil), routes.Workstreams...)
	wrongReference.Workstreams[0].WorkstreamRef = "compute-and-deployment#CD5"
	if err := validateDMLaneOwnerRoutes(wrongReference); err == nil {
		t.Fatal("workstream owner label that disagrees with the maintained source did not fail closed")
	}
}

func TestReadDMLaneGapsRejectsTamperedDurableEvidence(t *testing.T) {
	verdict := DMLaneVerdict{SchemaVersion: 1, ProfileID: "p1", Scenario: "tool-sites", OperationID: "op-4", Verdict: DMLaneBlocked,
		Stages: []DMLaneStageVerdict{{Stage: "teardown", Status: DMLaneBlocked, Reason: "missing exact release receipt", EvidenceRef: "deployment-manager:operation:op-4"}}}
	paths, err := WriteDMLaneGaps(t.TempDir(), []DMLaneVerdict{verdict})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadDMLaneGaps(paths)
	if err != nil || len(loaded) != 1 {
		t.Fatalf("persisted gap could not be read: %+v err=%v", loaded, err)
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var changed DMLaneGap
	if err := json.Unmarshal(raw, &changed); err != nil {
		t.Fatal(err)
	}
	changed.EvidenceRef = "deployment-manager:operation:other"
	mutated, err := json.MarshalIndent(changed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[0], append(mutated, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDMLaneGaps(paths); err == nil {
		t.Fatal("tampered persisted evidence was accepted")
	}
}

func TestSwarmManagerDMLaneGapIntakeIsIdempotentAndPreservesOwnerEvidence(t *testing.T) {
	items := map[string]*domainV1.BacklogItem{}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/backlog/idea/")
		switch r.Method {
		case http.MethodGet:
			item := items[name]
			if item == nil {
				http.NotFound(w, r)
				return
			}
			writeDMLaneBacklogResponse(t, w, item)
		case http.MethodPost:
			posts++
			var request apiV1.CreateBacklogItemRequest
			raw, err := io.ReadAll(r.Body)
			if err != nil || protojson.Unmarshal(raw, &request) != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if items[request.GetName()] != nil {
				http.Error(w, "duplicate", http.StatusConflict)
				return
			}
			item := &domainV1.BacklogItem{Name: request.GetName(), Title: request.GetTitle(), Description: request.GetDescription(), Kind: request.GetKind(), Priority: request.GetPriority(), Tags: request.GetTags(), AcceptanceAllow: request.GetAcceptanceAllow(), Status: "backlog"}
			items[item.GetName()] = item
			w.WriteHeader(http.StatusCreated)
			writeDMLaneBacklogResponse(t, w, item)
		default:
			http.Error(w, "unsupported", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	intake := &SwarmManagerDMLaneGapIntake{HTTPClient: server.Client(), ResolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	verdict := DMLaneVerdict{SchemaVersion: 1, ProfileID: "p1", Scenario: "tool-sites", OperationID: "op-4", Verdict: DMLaneBlocked,
		Stages: []DMLaneStageVerdict{{Stage: "teardown", Status: DMLaneBlocked, Reason: "no exact CM release receipt", EvidenceRef: "deployment-manager:operation:op-4"}}}
	gaps, err := ProjectDMLaneGaps([]DMLaneVerdict{verdict})
	if err != nil {
		t.Fatal(err)
	}
	first, err := intake.Submit(context.Background(), gaps[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := intake.Submit(context.Background(), gaps[0])
	if err != nil {
		t.Fatalf("same gap retry should resolve existing owner intake: %v", err)
	}
	if first != second || posts != 1 || first.BacklogRef != "swarm-manager:backlog:idea/dm-lane-gap-"+strings.TrimPrefix(gaps[0].ID, "sha256:") || first.ScenarioOwner != "scenario:tool-sites" || first.WorkstreamID != "CD6" || first.WorkstreamOwner != "deployment-manager" {
		t.Fatalf("intake identity or retry differs: first=%+v second=%+v posts=%d", first, second, posts)
	}
	stored := items[first.BacklogName]
	if stored == nil || !strings.Contains(stored.GetDescription(), gaps[0].Reason) || !strings.Contains(stored.GetDescription(), gaps[0].EvidenceRef) || !containsDMLaneTag(stored.GetTags(), "owner-scenario:tool-sites") || !containsDMLaneTag(stored.GetTags(), "owner-workstream:CD6") || !containsDMLaneTag(stored.GetTags(), "owner-workstream-ref:compute-and-deployment#CD6") {
		t.Fatalf("Swarm Manager item lost exact owner/evidence: %+v", stored)
	}
	changed := gaps[0]
	changed.Reason = "changed source reason"
	changed.ID, err = dmlaneGapID(changed)
	if err != nil {
		t.Fatal(err)
	}
	changedReceipt, err := intake.Submit(context.Background(), changed)
	if err != nil || changedReceipt.BacklogName == first.BacklogName || posts != 2 {
		t.Fatalf("changed gap should create distinct owner intake, receipt=%+v posts=%d err=%v", changedReceipt, posts, err)
	}
}

func TestSwarmManagerDMLaneGapIntakeFailsClosedOnOwnerOrExistingRecordMismatch(t *testing.T) {
	gap, err := ProjectDMLaneGaps([]DMLaneVerdict{{SchemaVersion: 1, ProfileID: "p1", Scenario: "tool-sites", Stages: []DMLaneStageVerdict{{Stage: "teardown", Status: DMLaneBlocked, Reason: "missing receipt"}}}})
	if err != nil {
		t.Fatal(err)
	}
	badOwner := gap[0]
	badOwner.WorkstreamOwner = "some-other-scenario"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			item := &domainV1.BacklogItem{Name: strings.TrimPrefix(r.URL.Path, "/api/v1/backlog/idea/"), Title: "different", Kind: "idea", Priority: 5}
			writeDMLaneBacklogResponse(t, w, item)
			return
		}
		t.Fatal("mismatched existing work item must not trigger a replacement write")
	}))
	defer server.Close()
	intake := &SwarmManagerDMLaneGapIntake{HTTPClient: server.Client(), ResolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	if _, err := intake.Submit(context.Background(), badOwner); err == nil || !strings.Contains(err.Error(), "owner route") {
		t.Fatalf("unrecognized owner route was not rejected: %v", err)
	}
	if _, err := intake.Submit(context.Background(), gap[0]); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("conflicting existing intake was not rejected: %v", err)
	}
	unavailable := &SwarmManagerDMLaneGapIntake{ResolveURL: func(context.Context) (string, error) { return "", fmt.Errorf("no configured owner intake") }}
	if _, err := unavailable.Submit(context.Background(), gap[0]); err == nil {
		t.Fatal("missing intake destination should fail closed")
	}
}

func TestReadPersistAndRouteDMLaneVerdictsUsesPersistedGapFilesAndReceipts(t *testing.T) {
	items := map[string]*domainV1.BacklogItem{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/backlog/idea/")
		if r.Method == http.MethodGet {
			if item := items[name]; item != nil {
				writeDMLaneBacklogResponse(t, w, item)
				return
			}
			http.NotFound(w, r)
			return
		}
		var request apiV1.CreateBacklogItemRequest
		raw, _ := io.ReadAll(r.Body)
		if err := protojson.Unmarshal(raw, &request); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		item := &domainV1.BacklogItem{Name: request.GetName(), Title: request.GetTitle(), Description: request.GetDescription(), Kind: request.GetKind(), Priority: request.GetPriority(), Tags: request.GetTags(), AcceptanceAllow: request.GetAcceptanceAllow(), Status: "backlog"}
		items[item.GetName()] = item
		w.WriteHeader(http.StatusCreated)
		writeDMLaneBacklogResponse(t, w, item)
	}))
	defer server.Close()
	intake := &SwarmManagerDMLaneGapIntake{HTTPClient: server.Client(), ResolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	profiles := &dmLaneProfileReaderFake{pages: map[string]*profilesv1.ListProfilesResponse{"": {Profiles: []*profilesv1.Profile{{Id: "p1", Scenario: "tool-sites"}}}}}
	operations := &dmLaneOperationReaderFake{pages: map[string]*releasesv1.ListReleaseOperationsByProfileResponse{"p1|": {}}}
	dir := t.TempDir()
	verdicts, receipts, paths, err := ReadPersistAndRouteDMLaneVerdicts(context.Background(), profiles, operations, dir, intake)
	if err != nil {
		t.Fatal(err)
	}
	if len(verdicts) != 1 || len(receipts) != 5 || len(paths) != 11 {
		t.Fatalf("routed readback = %d verdicts, %d receipts, %d paths", len(verdicts), len(receipts), len(paths))
	}
	for _, receipt := range receipts {
		if _, err := os.Stat(filepath.Join(dir, "owner-intake", "dm-intake-"+strings.TrimPrefix(receipt.ReceiptID, "sha256:")+".json")); err != nil {
			t.Fatalf("durable owner-intake receipt missing: %v", err)
		}
	}
	profiles.requests = nil
	operations.listCalls = nil
	verdicts2, receipts2, paths2, err := ReadPersistAndRouteDMLaneVerdicts(context.Background(), profiles, operations, dir, intake)
	if err != nil {
		t.Fatalf("complete retry should be idempotent: %v", err)
	}
	if len(verdicts2) != 1 || len(receipts2) != len(receipts) || len(paths2) != len(paths) {
		t.Fatalf("retry changed intake results: %d verdicts, %d receipts, %d paths", len(verdicts2), len(receipts2), len(paths2))
	}
}

func writeDMLaneBacklogResponse(t *testing.T, w http.ResponseWriter, item *domainV1.BacklogItem) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	raw, err := protojson.Marshal(&apiV1.BacklogItemResponse{Item: item})
	if err != nil {
		t.Fatalf("encode fake Swarm Manager response: %v", err)
	}
	_, _ = w.Write(raw)
}

func containsDMLaneTag(tags []string, value string) bool {
	for _, tag := range tags {
		if tag == value {
			return true
		}
	}
	return false
}

func TestReadAndPersistDMLaneVerdictsAlsoPersistsOwnerRoutedGaps(t *testing.T) {
	profiles := &dmLaneProfileReaderFake{pages: map[string]*profilesv1.ListProfilesResponse{"": {Profiles: []*profilesv1.Profile{{Id: "p1", Scenario: "tool-sites"}}}}}
	operations := &dmLaneOperationReaderFake{pages: map[string]*releasesv1.ListReleaseOperationsByProfileResponse{"p1|": {}}}
	verdicts, paths, err := ReadAndPersistDMLaneVerdicts(context.Background(), profiles, operations, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(verdicts) != 1 || verdicts[0].Verdict != DMLaneBlocked || len(paths) != 6 {
		t.Fatalf("read/persist result = %d verdicts, %d paths: %+v", len(verdicts), len(paths), paths)
	}
	gapCount := 0
	for _, path := range paths {
		if strings.Contains(path, string(filepath.Separator)+"gaps"+string(filepath.Separator)) {
			gapCount++
		}
	}
	if gapCount != 5 {
		t.Fatalf("durable gap routes = %d, want five non-passed stages", gapCount)
	}
}
