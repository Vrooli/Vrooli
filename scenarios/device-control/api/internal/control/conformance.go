package control

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"device-control/internal/conformance"
	"device-control/internal/evidence"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConformanceChapterResult struct {
	ID          string               `json:"id"`
	Purpose     string               `json:"purpose"`
	Expected    string               `json:"expected"`
	Disposition string               `json:"disposition"`
	Message     string               `json:"message"`
	DurationMS  int64                `json:"duration_ms"`
	Evidence    []evidence.Reference `json:"evidence,omitempty"`
}

type AndroidConformanceResult struct {
	PlanID      string                     `json:"plan_id"`
	Fixture     conformance.Fixture        `json:"fixture"`
	DeviceID    string                     `json:"device_id"`
	Serial      string                     `json:"serial,omitempty"`
	HostNodeID  string                     `json:"host_node_id,omitempty"`
	RunID       string                     `json:"run_id"`
	Disposition string                     `json:"disposition"`
	Reason      string                     `json:"reason,omitempty"`
	Chapters    []ConformanceChapterResult `json:"chapters"`
	Verdict     *commonv1.TargetVerdict    `json:"verdict,omitempty"`
}

func (s *Service) AndroidConformancePlan() conformance.Plan { return conformance.AndroidPlan() }

func (s *Service) RunAndroidConformance(ctx context.Context, fixture conformance.Fixture, deviceID, actor, leaseToken string) (AndroidConformanceResult, error) {
	plan := conformance.AndroidPlan()
	result := AndroidConformanceResult{PlanID: plan.ID, Fixture: fixture, DeviceID: deviceID, RunID: fmt.Sprintf("%s-%d", plan.ID, time.Now().UTC().UnixNano()), Disposition: "not_run", Chapters: make([]ConformanceChapterResult, 0, len(plan.Chapters))}
	// Conformance is a standalone operator command. Refresh the bridge and
	// strategy inventory before resolving the requested identity so a service
	// restart or a newly attached phone does not require a preceding `device
	// list` call to make the device leasable.
	_ = s.Devices(ctx)
	// Bind identity before any fail-closed validation so an unavailable run is
	// still attributable to the requested physical serial and host node.
	if record, ok := s.devices.Get(deviceID); ok {
		result.Serial = record.Serial
		result.HostNodeID = record.HostNodeID
		if record.Kind != "" && record.Kind != "physical" {
			result.Disposition = "unavailable"
			result.Reason = fmt.Sprintf("Android physical conformance requires a physical device; %s is %s", deviceID, record.Kind)
			result.Verdict = conformanceVerdict(result, nil)
			return result, nil
		}
	}
	if err := plan.Validate(); err != nil {
		result.Disposition, result.Reason = "failed", err.Error()
		result.Verdict = conformanceVerdict(result, nil)
		return result, nil
	}
	if err := fixture.Validate(); err != nil {
		result.Disposition, result.Reason = "unavailable", err.Error()
		result.Verdict = conformanceVerdict(result, nil)
		return result, nil
	}
	if _, err := os.Stat(fixture.APKPath); err != nil {
		result.Disposition = "unavailable"
		result.Reason = fmt.Sprintf("hello-mobile fixture APK is unavailable at %s: %v", fixture.APKPath, err)
		result.Verdict = conformanceVerdict(result, nil)
		return result, nil
	}

	var session Session
	var err error
	if strings.TrimSpace(leaseToken) != "" {
		session, err = s.sessionForLease(ctx, deviceID, leaseToken)
	} else {
		session, err = s.AcquireContext(ctx, deviceID, actor, 15*time.Minute)
	}
	if err != nil {
		return result, err
	}
	if leaseToken == "" {
		defer func() { _, _ = s.ReleaseContext(ctx, session.ID) }()
	}
	result.RunID = fmt.Sprintf("%s-%s", plan.ID, session.ID)

	allPassed := true
	var refs []evidence.Reference
	for _, chapter := range plan.Chapters {
		chapterResult := ConformanceChapterResult{ID: chapter.ID, Purpose: chapter.Purpose, Expected: chapter.Expected, Disposition: "not_run"}
		chapterStarted := time.Now()
		flowResult, runErr := s.execute(ctx, chapter.Flow(fixture), deviceID, actor, session, false)
		chapterResult.DurationMS = time.Since(chapterStarted).Milliseconds()
		if runErr != nil {
			chapterResult.Disposition, chapterResult.Message = "failed", runErr.Error()
			allPassed = false
		} else {
			chapterResult.Disposition = flowResult.Disposition
			chapterResult.Message = chapter.Expected
			chapterResult.Evidence = append(chapterResult.Evidence, flowResult.Evidence...)
			refs = append(refs, flowResult.Evidence...)
			if flowResult.Disposition != "passed" {
				if flowResult.Disposition == "capability_gap" {
					chapterResult.Disposition = "unavailable"
					if len(flowResult.Chapters) > 0 {
						chapterResult.Message = flowResult.Chapters[0].Message
					}
				}
				allPassed = false
			}
		}
		result.Chapters = append(result.Chapters, chapterResult)
	}
	result.Disposition = "failed"
	if allPassed {
		result.Disposition = "passed"
	}
	result.Verdict = conformanceVerdict(result, refs)
	return result, nil
}

func conformanceVerdict(result AndroidConformanceResult, refs []evidence.Reference) *commonv1.TargetVerdict {
	disposition := commonv1.Disposition_DISPOSITION_FAILED
	if result.Disposition == "passed" {
		disposition = commonv1.Disposition_DISPOSITION_PASSED
	}
	protoRefs := make([]*commonv1.EvidenceRef, 0, len(refs))
	for _, ref := range refs {
		protoRefs = append(protoRefs, &commonv1.EvidenceRef{Producer: ref.Producer, ArtifactId: ref.ID, Kind: ref.Kind, Checksum: ref.Checksum, SizeBytes: ref.SizeBytes, CreatedAt: timestamppb.New(ref.CreatedAt)})
	}
	detail := fmt.Sprintf("fixture=%s device_id=%s serial=%s host_node_id=%s disposition=%s", result.Fixture.ID, result.DeviceID, result.Serial, result.HostNodeID, result.Disposition)
	if result.Reason != "" {
		detail += " reason=" + result.Reason
	}
	target := &commonv1.EvidenceTarget{Ramp: "device-control", Platform: "android", Os: "Android", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_PHYSICAL}
	if strings.TrimSpace(result.HostNodeID) != "" {
		node := result.HostNodeID
		target.BridgeNodeId = &node
	}
	return &commonv1.TargetVerdict{Target: target, Disposition: disposition, Refs: protoRefs, RunId: result.RunID, Detail: detail}
}
