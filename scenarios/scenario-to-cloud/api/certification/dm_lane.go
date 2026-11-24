package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles"
	profilesv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles/profilesv1connect"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	"google.golang.org/protobuf/proto"
)

// DMLaneReadbackClients narrows the generated DM APIs to their read-only
// discovery/readback calls. Generated clients implement these interfaces;
// Start/Resume/Recover are intentionally not reachable from the lane runner.
type DMLaneProfileReader interface {
	ListProfiles(context.Context, *connect.Request[profilesv1.ListProfilesRequest]) (*connect.Response[profilesv1.ListProfilesResponse], error)
}

type DMLaneOperationReader interface {
	ListOperationsByProfile(context.Context, *connect.Request[releasesv1.ListReleaseOperationsByProfileRequest]) (*connect.Response[releasesv1.ListReleaseOperationsByProfileResponse], error)
	GetOperation(context.Context, *connect.Request[releasesv1.GetReleaseOperationRequest]) (*connect.Response[releasesv1.GetReleaseOperationResponse], error)
}

var _ DMLaneProfileReader = (profilesv1connect.ProfilesServiceClient)(nil)
var _ DMLaneOperationReader = (releasesv1connect.ReleasesServiceClient)(nil)

// ReadDMLaneVerdicts discovers profiles and operation IDs through generated DM
// read-only indexes, then fetches each canonical standing by exact operation
// ID. Lifecycle operations are intentionally outside the reader interface.
func ReadDMLaneVerdicts(ctx context.Context, profileReader DMLaneProfileReader, operationReader DMLaneOperationReader) ([]DMLaneVerdict, error) {
	if profileReader == nil || operationReader == nil {
		return nil, fmt.Errorf("generated DM readback clients are required")
	}
	var profiles []*profilesv1.Profile
	pageToken := ""
	for {
		response, err := profileReader.ListProfiles(ctx, connect.NewRequest(&profilesv1.ListProfilesRequest{PageSize: 100, PageToken: pageToken}))
		if err != nil {
			return nil, fmt.Errorf("generated DM profile discovery: %w", err)
		}
		profiles = append(profiles, response.Msg.GetProfiles()...)
		pageToken = response.Msg.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	known := make(map[string]bool, len(profiles))
	for _, profile := range profiles {
		if profile != nil {
			known[profile.GetId()] = true
		}
	}
	var operations []*releasesv1.ReleaseOperation
	for _, profile := range profiles {
		if profile == nil || !known[profile.GetId()] {
			continue
		}
		profileID := profile.GetId()
		pageToken := ""
		seenTokens := map[string]bool{}
		seenIDs := map[string]bool{}
		for {
			page, err := operationReader.ListOperationsByProfile(ctx, connect.NewRequest(&releasesv1.ListReleaseOperationsByProfileRequest{ProfileId: profileID, PageSize: 100, PageToken: pageToken}))
			if err != nil {
				return nil, fmt.Errorf("generated DM operation index for profile %q: %w", profileID, err)
			}
			for _, operationID := range page.Msg.GetOperationIds() {
				if strings.TrimSpace(operationID) == "" {
					return nil, fmt.Errorf("empty DM operation ID supplied for profile %q", profileID)
				}
				if seenIDs[operationID] {
					return nil, fmt.Errorf("duplicate DM operation ID %q in profile %q index", operationID, profileID)
				}
				seenIDs[operationID] = true
				response, err := operationReader.GetOperation(ctx, connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: operationID}))
				if err != nil {
					return nil, fmt.Errorf("generated DM operation readback %q: %w", operationID, err)
				}
				if response.Msg.GetOperation() == nil {
					return nil, fmt.Errorf("generated DM operation readback %q is empty", operationID)
				}
				if response.Msg.GetOperation().GetProfileId() != profileID {
					return nil, fmt.Errorf("DM operation %q belongs to profile %q, expected %q", operationID, response.Msg.GetOperation().GetProfileId(), profileID)
				}
				operations = append(operations, response.Msg.GetOperation())
			}
			pageToken = page.Msg.GetNextPageToken()
			if pageToken == "" {
				break
			}
			if seenTokens[pageToken] {
				return nil, fmt.Errorf("generated DM operation index repeated page token %q for profile %q", pageToken, profileID)
			}
			seenTokens[pageToken] = true
		}
	}
	return ProjectDMLaneVerdicts(profiles, operations)
}

// ReadAndPersistDMLaneVerdicts completes the read-only lane pass and appends
// its exact generated-readback projection as an immutable snapshot.
func ReadAndPersistDMLaneVerdicts(ctx context.Context, profileReader DMLaneProfileReader, operationReader DMLaneOperationReader, evidenceDir string) ([]DMLaneVerdict, []string, error) {
	verdicts, err := ReadDMLaneVerdicts(ctx, profileReader, operationReader)
	if err != nil {
		return nil, nil, err
	}
	paths, err := WriteDMLaneVerdicts(evidenceDir, verdicts)
	if err != nil {
		return nil, nil, err
	}
	return verdicts, paths, nil
}

// DMLaneVerdict is the steward-readable projection of generated Deployment
// Manager profile and operation readback. It records observed evidence only;
// it never starts or resumes an operation.
type DMLaneVerdict struct {
	SchemaVersion   int                  `json:"schema_version"`
	ProfileID       string               `json:"profile_id"`
	Scenario        string               `json:"scenario"`
	OperationID     string               `json:"operation_id,omitempty"`
	OperationStatus string               `json:"operation_status,omitempty"`
	ActiveStage     string               `json:"active_stage,omitempty"`
	UpdatedAt       string               `json:"updated_at,omitempty"`
	Stages          []DMLaneStageVerdict `json:"stages"`
	Verdict         DMLaneState          `json:"verdict"`
	Reason          string               `json:"reason,omitempty"`
}

type DMLaneStageVerdict struct {
	Stage       string      `json:"stage"`
	Status      DMLaneState `json:"status"`
	EvidenceRef string      `json:"evidence_ref,omitempty"`
	Reason      string      `json:"reason,omitempty"`
}

// DMLaneState deliberately remains separate from certification Receipt.Verdict:
// blocked/not_run operational observations cannot satisfy a certification cell.
type DMLaneState string

const (
	DMLanePassed  DMLaneState = "passed"
	DMLaneFailed  DMLaneState = "failed"
	DMLaneBlocked DMLaneState = "blocked"
	DMLaneNotRun  DMLaneState = "not_run"
)

// ProjectDMLaneVerdicts projects generated DM data. Every supplied operation
// must match a discovered profile. The newest readback is selected per profile
// using the DM updated_at timestamp; equal timestamps use operation ID for a
// stable result. A missing generated teardown stage is always blocked.
func ProjectDMLaneVerdicts(profiles []*profilesv1.Profile, operations []*releasesv1.ReleaseOperation) ([]DMLaneVerdict, error) {
	byProfile := make(map[string][]*releasesv1.ReleaseOperation)
	for _, operation := range operations {
		if operation == nil || strings.TrimSpace(operation.GetOperationId()) == "" || strings.TrimSpace(operation.GetProfileId()) == "" {
			return nil, fmt.Errorf("DM operation readback is missing operation/profile identity")
		}
		byProfile[operation.GetProfileId()] = append(byProfile[operation.GetProfileId()], operation)
	}
	seen := map[string]bool{}
	result := make([]DMLaneVerdict, 0, len(profiles))
	for _, profile := range profiles {
		if profile == nil || strings.TrimSpace(profile.GetId()) == "" || strings.TrimSpace(profile.GetScenario()) == "" {
			return nil, fmt.Errorf("generated DM profile is missing profile/scenario identity")
		}
		if seen[profile.GetId()] {
			return nil, fmt.Errorf("duplicate DM profile %q", profile.GetId())
		}
		seen[profile.GetId()] = true
		ops := byProfile[profile.GetId()]
		if len(ops) == 0 {
			result = append(result, DMLaneVerdict{
				SchemaVersion: 1, ProfileID: profile.GetId(), Scenario: profile.GetScenario(),
				Stages: []DMLaneStageVerdict{
					{Stage: "acquire", Status: DMLaneNotRun, Reason: "no DM operation readback supplied"},
					{Stage: "deploy", Status: DMLaneNotRun, Reason: "no DM operation readback supplied"},
					{Stage: "verify", Status: DMLaneNotRun, Reason: "no DM operation readback supplied"},
					{Stage: "teardown", Status: DMLaneBlocked, Reason: "no DM teardown receipt is present"},
				},
				Verdict: DMLaneBlocked, Reason: "no DM operation readback; teardown receipt is unavailable",
			})
			continue
		}
		sort.Slice(ops, func(i, j int) bool {
			a, b := ops[i].GetUpdatedAt(), ops[j].GetUpdatedAt()
			if a != nil && b != nil && !a.AsTime().Equal(b.AsTime()) {
				return a.AsTime().After(b.AsTime())
			}
			if a != nil && b == nil {
				return true
			}
			if a == nil && b != nil {
				return false
			}
			return ops[i].GetOperationId() > ops[j].GetOperationId()
		})
		operation := ops[0]
		if err := validateOperationProfile(operation, profile.GetId()); err != nil {
			return nil, err
		}
		verdict := projectOperation(profile, operation)
		result = append(result, verdict)
	}
	for profileID := range byProfile {
		if !seen[profileID] {
			return nil, fmt.Errorf("DM operation %q references undiscovered profile %q", byProfile[profileID][0].GetOperationId(), profileID)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ProfileID < result[j].ProfileID })
	return result, nil
}

func validateOperationProfile(operation *releasesv1.ReleaseOperation, profileID string) error {
	if operation.GetProfileId() != profileID {
		return fmt.Errorf("DM operation/profile mismatch: %q != %q", operation.GetProfileId(), profileID)
	}
	return nil
}

func projectOperation(profile *profilesv1.Profile, operation *releasesv1.ReleaseOperation) DMLaneVerdict {
	ref := "deployment-manager:operation:" + operation.GetOperationId()
	acquireStatus, acquireReason := validateDMLaneAcquireReceipt(profile, operation)
	stages := []DMLaneStageVerdict{
		{Stage: "acquire", Status: acquireStatus, EvidenceRef: ref, Reason: acquireReason},
		{Stage: "deploy", Status: DMLaneNotRun, EvidenceRef: ref, Reason: "DM readback has no durable deployment receipt"},
		{Stage: "verify", Status: DMLaneNotRun, EvidenceRef: ref, Reason: "DM readback has no passed verification receipt"},
		projectDMLaneTeardown(profile, operation, ref),
	}
	cloud := operation.GetCloudStep()
	if cloud != nil {
		if cloud.GetDeploymentId() != "" && cloud.GetReceiptJson() != "" {
			stages[1] = DMLaneStageVerdict{Stage: "deploy", Status: DMLanePassed, EvidenceRef: ref + ":cloud-step:" + cloud.GetOperationId()}
		}
		if cloud.GetVerificationState() == "passed" && cloud.GetVerificationReceiptJson() != "" {
			stages[2] = DMLaneStageVerdict{Stage: "verify", Status: DMLanePassed, EvidenceRef: ref + ":verification:" + cloud.GetVerificationKey()}
		} else if cloud.GetVerificationState() == "failed" || cloud.GetVerificationState() == "unhealthy" {
			stages[2] = DMLaneStageVerdict{Stage: "verify", Status: DMLaneFailed, EvidenceRef: ref + ":verification:" + cloud.GetVerificationKey(), Reason: cloud.GetReason()}
		}
	}
	if operation.GetStatus() == releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED || operation.GetStatus() == releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_AMBIGUOUS {
		stage := operation.GetActiveStage()
		switch stage {
		case "dm-acquire":
			stage = "acquire"
		case "dm-cloud":
			stage = "deploy"
		}
		for i := range stages {
			if stages[i].Stage == stage && stages[i].Status != DMLanePassed {
				stages[i].Status, stages[i].Reason = DMLaneFailed, operation.GetError()
			}
		}
	}
	aggregate := DMLanePassed
	reason := ""
	for _, stage := range stages {
		if stage.Status == DMLaneFailed {
			aggregate, reason = DMLaneFailed, stage.Reason
			break
		}
		if stage.Status == DMLaneBlocked && aggregate != DMLaneFailed {
			aggregate, reason = DMLaneBlocked, stage.Reason
		}
		if stage.Status == DMLaneNotRun && aggregate == DMLanePassed {
			aggregate, reason = DMLaneNotRun, stage.Reason
		}
	}
	updated := ""
	if operation.GetUpdatedAt() != nil {
		updated = operation.GetUpdatedAt().AsTime().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	return DMLaneVerdict{SchemaVersion: 1, ProfileID: profile.GetId(), Scenario: profile.GetScenario(), OperationID: operation.GetOperationId(), OperationStatus: operation.GetStatus().String(), ActiveStage: operation.GetActiveStage(), UpdatedAt: updated, Stages: stages, Verdict: aggregate, Reason: reason}
}

func projectDMLaneTeardown(profile *profilesv1.Profile, operation *releasesv1.ReleaseOperation, ref string) DMLaneStageVerdict {
	blocked := DMLaneStageVerdict{Stage: "teardown", Status: DMLaneBlocked, EvidenceRef: ref, Reason: "no exact CM release receipt is present"}
	standing := operation.GetCloudTeardownReceipt()
	acquire := operation.GetCloudAcquireReceipt()
	if standing == nil {
		return blocked
	}
	if standing.GetSchemaVersion() != 1 || standing.GetOperationId() != operation.GetOperationId() || standing.GetReleaseId() != operation.GetReleaseId() || standing.GetProfileId() != profile.GetId() || standing.GetTeardownIdempotencyKey() != "dm-release:"+operation.GetOperationId() || acquire == nil || standing.GetAcquisitionReceiptSha256() != acquire.GetSourceReceiptSha256() || standing.GetComputeManagerInstanceId() != acquire.GetComputeManagerInstanceId() || standing.GetStoredTargetSha256() != acquire.GetStoredTargetSha256() {
		blocked.Reason = "DM teardown receipt does not bind the exact operation and acquisition receipt"
		return blocked
	}
	cm := standing.GetOwnerReceipt()
	if standing.GetState() != releasesv1.CloudTeardownState_CLOUD_TEARDOWN_STATE_RELEASED || cm == nil {
		if standing.GetState() == releasesv1.CloudTeardownState_CLOUD_TEARDOWN_STATE_FAILED {
			return DMLaneStageVerdict{Stage: "teardown", Status: DMLaneFailed, EvidenceRef: ref + ":teardown:" + standing.GetTeardownIdempotencyKey(), Reason: standing.GetReason()}
		}
		blocked.Reason = "DM teardown is not released with an exact CM owner receipt: " + standing.GetReason()
		return blocked
	}
	acquired := acquire.GetSourceReceipt()
	instance := acquired.GetInstance()
	if instance == nil || cm.GetIdempotencyKey() != standing.GetTeardownIdempotencyKey() || cm.GetOwner() != instance.GetTenant() || cm.GetInstanceId() != instance.GetId() || cm.GetInstanceId() != standing.GetComputeManagerInstanceId() || cm.GetProvider() != instance.GetProvider() || cm.GetProvider() != standing.GetProvider() || cm.GetProviderInstanceId() != instance.GetProviderInstanceId() || cm.GetProviderInstanceId() != standing.GetProviderInstanceId() || cm.GetAction() != "release" || cm.GetState() != instancev1.ReleaseState_RELEASE_STATE_RELEASED {
		blocked.Reason = "CM release receipt does not match the exact acquired owner and target"
		return blocked
	}
	return DMLaneStageVerdict{Stage: "teardown", Status: DMLanePassed, EvidenceRef: ref + ":teardown:" + standing.GetTeardownIdempotencyKey()}
}

func validateDMLaneAcquireReceipt(profile *profilesv1.Profile, operation *releasesv1.ReleaseOperation) (DMLaneState, string) {
	blocked := func(reason string) (DMLaneState, string) { return DMLaneBlocked, reason }
	receipt := operation.GetCloudAcquireReceipt()
	if receipt == nil {
		return blocked("DM operation has no typed Compute Manager acquisition receipt")
	}
	if receipt.GetSchemaVersion() != 1 || receipt.GetOperationId() != operation.GetOperationId() || receipt.GetReleaseId() == "" || operation.GetReleaseId() != receipt.GetReleaseId() || operation.GetOperationId() != "release-op-"+receipt.GetReleaseId() || receipt.GetProfileId() != profile.GetId() || operation.GetProfileId() != receipt.GetProfileId() {
		return blocked("typed acquisition receipt does not bind the canonical DM operation, release, and profile")
	}
	key := receipt.GetAcquisitionIdempotencyKey()
	instanceID := receipt.GetComputeManagerInstanceId()
	if key == "" || key != operation.GetCloudAcquireIdempotencyKey() || key != operation.GetCloudSpendQuoteIdempotencyKey() || instanceID == "" || instanceID != operation.GetCloudComputeManagerInstanceId() {
		return blocked("typed acquisition receipt does not bind the DM acquisition key, quote key, and exact instance")
	}
	if !validSHA256(receipt.GetStoredTargetSha256()) || !validSHA256(receipt.GetSourceReceiptSha256()) || receipt.GetPersistedAt() == nil || !receipt.GetPersistedAt().IsValid() {
		return blocked("typed acquisition receipt has an invalid target/source fingerprint or persistence time")
	}
	source := receipt.GetSourceReceipt()
	if source == nil || source.GetState() != "acquired" || source.GetInstance() == nil || source.GetSpendReceipt() == nil {
		return blocked("typed acquisition source is not a complete acquired CM readback")
	}
	cmInstance, spend := source.GetInstance(), source.GetSpendReceipt()
	if cmInstance.GetId() != instanceID || cmInstance.GetTenant() != "deployment-manager" || spend.GetInstanceId() != instanceID || spend.GetIdempotencyKey() != key || spend.GetQuoteState() != "approved" || spend.GetAdmissionDecision() != "approved" {
		return blocked("nested CM spend receipt does not match the DM owner, instance, key, and approved admission")
	}
	if spend.GetObservedAt() == nil || spend.GetExpiresAt() == nil || !spend.GetObservedAt().IsValid() || !spend.GetExpiresAt().IsValid() {
		return blocked("nested CM spend receipt has no valid source freshness interval")
	}
	observedAt, expiresAt, persistedAt := spend.GetObservedAt().AsTime(), spend.GetExpiresAt().AsTime(), receipt.GetPersistedAt().AsTime()
	if observedAt.After(persistedAt) || !expiresAt.After(persistedAt) || expiresAt.Before(observedAt) {
		return blocked("nested CM source quote was stale when the acquisition receipt was persisted")
	}
	wire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(source)
	if err != nil {
		return blocked("nested CM source receipt cannot be fingerprinted")
	}
	fingerprint := sha256.Sum256(wire)
	if receipt.GetSourceReceiptSha256() != "sha256:"+hex.EncodeToString(fingerprint[:]) {
		return blocked("nested CM source receipt fingerprint does not match the typed envelope")
	}
	return DMLanePassed, ""
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(decoded) == sha256.Size
}

// WriteDMLaneVerdicts appends an immutable content-addressed snapshot. It is
// intentionally separate from pass-receipt evidence: a blocked/not_run fleet
// projection must never satisfy a certification cell.
func WriteDMLaneVerdicts(dir string, verdicts []DMLaneVerdict) ([]string, error) {
	if len(verdicts) == 0 {
		return nil, fmt.Errorf("no DM lane verdicts to persist")
	}
	raw, err := json.MarshalIndent(verdicts, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode DM lane verdicts: %w", err)
	}
	sum := sha256.Sum256(raw)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create DM lane verdict directory: %w", err)
	}
	path := filepath.Join(dir, "dm-lane-"+hex.EncodeToString(sum[:])+".json")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, fmt.Errorf("append DM lane verdict snapshot: %w", err)
	}
	if _, err = f.Write(append(raw, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("write DM lane verdict snapshot: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close DM lane verdict snapshot: %w", closeErr)
	}
	return []string{path}, nil
}
