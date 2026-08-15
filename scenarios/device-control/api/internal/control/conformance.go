package control

import (
	"context"
	"fmt"
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

type AndroidCapabilitySelfTestResult struct {
	PlanID        string                     `json:"plan_id"`
	DeviceID      string                     `json:"device_id"`
	Serial        string                     `json:"serial,omitempty"`
	HostNodeID    string                     `json:"host_node_id,omitempty"`
	RunID         string                     `json:"run_id"`
	Disposition   string                     `json:"disposition"`
	Reason        string                     `json:"reason,omitempty"`
	EvidenceClass string                     `json:"evidence_class,omitempty"`
	Chapters      []ConformanceChapterResult `json:"chapters"`
	Verdict       *commonv1.TargetVerdict    `json:"verdict,omitempty"`
}

func (s *Service) AndroidCapabilitySelfTestPlan() conformance.Plan {
	return conformance.AndroidCapabilityPlan()
}

func (s *Service) RunAndroidCapabilitySelfTest(ctx context.Context, deviceID, actor, leaseToken string) (AndroidCapabilitySelfTestResult, error) {
	plan := conformance.AndroidCapabilityPlan()
	result := AndroidCapabilitySelfTestResult{PlanID: plan.ID, DeviceID: deviceID, RunID: fmt.Sprintf("%s-%d", plan.ID, time.Now().UTC().UnixNano()), Disposition: "not_run", Chapters: make([]ConformanceChapterResult, 0, len(plan.Chapters))}
	_ = s.Devices(ctx)
	record, found := s.devices.Get(deviceID)
	if !found {
		result.Disposition = "unavailable"
		result.Reason = fmt.Sprintf("Android target %s is not present in device-control inventory", deviceID)
		result.Verdict = conformanceVerdict(result, nil)
		return result, nil
	}
	if found {
		result.Serial = record.Serial
		result.HostNodeID = record.HostNodeID
		if item, exists := s.registry.Get(record.StrategyID); exists {
			if declaration, describeErr := item.Describe(ctx); describeErr == nil {
				result.EvidenceClass = declaration.EvidenceClass
			}
		}
		if record.Kind != "" && record.Kind != "physical" && record.Kind != "emulator" {
			result.Disposition = "unavailable"
			result.Reason = fmt.Sprintf("Android capability self-test requires an Android device or emulator; %s is %s", deviceID, record.Kind)
			result.Verdict = conformanceVerdict(result, nil)
			return result, nil
		}
	}
	if err := plan.Validate(); err != nil {
		result.Disposition, result.Reason = "failed", err.Error()
		result.Verdict = conformanceVerdict(result, nil)
		return result, nil
	}
	if _, ok := s.strategyForDevice(deviceID); !ok {
		result.Disposition = "unavailable"
		result.Reason = fmt.Sprintf("device-control strategy for %s is unavailable", deviceID)
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
		flowResult, runErr := s.execute(ctx, chapter.Flow(), deviceID, actor, session, false)
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

func conformanceVerdict(result AndroidCapabilitySelfTestResult, refs []evidence.Reference) *commonv1.TargetVerdict {
	disposition := commonv1.Disposition_DISPOSITION_FAILED
	if result.Disposition == "passed" {
		disposition = commonv1.Disposition_DISPOSITION_PASSED
	}
	protoRefs := make([]*commonv1.EvidenceRef, 0, len(refs))
	for _, ref := range refs {
		protoRefs = append(protoRefs, &commonv1.EvidenceRef{Producer: ref.Producer, ArtifactId: ref.ID, Kind: ref.Kind, Checksum: ref.Checksum, SizeBytes: ref.SizeBytes, CreatedAt: timestamppb.New(ref.CreatedAt)})
	}
	detail := fmt.Sprintf("device_id=%s serial=%s host_node_id=%s disposition=%s", result.DeviceID, result.Serial, result.HostNodeID, result.Disposition)
	if result.Reason != "" {
		detail += " reason=" + result.Reason
	}
	deviceKind := commonv1.DeviceKind_DEVICE_KIND_PHYSICAL
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(result.Serial)), "emulator-") {
		deviceKind = commonv1.DeviceKind_DEVICE_KIND_EMULATOR
	}
	target := &commonv1.EvidenceTarget{Ramp: "device-control", Platform: "android", Os: "Android", DeviceKind: deviceKind}
	if strings.TrimSpace(result.HostNodeID) != "" {
		node := result.HostNodeID
		target.BridgeNodeId = &node
	}
	return &commonv1.TargetVerdict{Target: target, Disposition: disposition, Refs: protoRefs, RunId: result.RunID, Detail: detail, EvidenceClass: result.EvidenceClass}
}
