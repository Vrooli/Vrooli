package convert

import (
	investigationspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func InvestigationToProto(inv *models.Investigation) *investigationspb.Investigation {
	if inv == nil {
		return nil
	}
	pb := &investigationspb.Investigation{
		Id:        inv.ID,
		Status:    investigationStatusToProto(inv.Status),
		AnomalyId: inv.AnomalyID,
		StartTime: timestamppb.New(inv.StartTime),
		Findings:  inv.Findings,
		Progress:  int32(inv.Progress),
	}
	if inv.EndTime != nil {
		pb.EndTime = timestamppb.New(*inv.EndTime)
	}
	if inv.Details != nil {
		if s, err := structpb.NewStruct(inv.Details); err == nil {
			pb.Details = s
		}
	}
	if len(inv.Steps) > 0 {
		pb.Steps = make([]*investigationspb.InvestigationStep, len(inv.Steps))
		for i, step := range inv.Steps {
			pb.Steps[i] = InvestigationStepToProto(step)
		}
	}
	return pb
}

func InvestigationsToProto(invs []*models.Investigation) []*investigationspb.Investigation {
	result := make([]*investigationspb.Investigation, len(invs))
	for i, inv := range invs {
		result[i] = InvestigationToProto(inv)
	}
	return result
}

func InvestigationStepToProto(step models.InvestigationStep) *investigationspb.InvestigationStep {
	pb := &investigationspb.InvestigationStep{
		Name:      step.Name,
		Status:    investigationStepStatusToProto(step.Status),
		StartTime: timestamppb.New(step.StartTime),
		Findings:  step.Findings,
	}
	if step.EndTime != nil {
		pb.EndTime = timestamppb.New(*step.EndTime)
	}
	return pb
}

func CooldownStatusToProto(cs *models.CooldownStatus) *investigationspb.CooldownStatus {
	if cs == nil {
		return nil
	}
	return &investigationspb.CooldownStatus{
		CooldownPeriodSeconds: int32(cs.CooldownPeriodSeconds),
		RemainingSeconds:      int32(cs.RemainingSeconds),
		LastTriggerTime:       timestamppb.New(cs.LastTriggerTime),
		IsReady:               cs.IsReady,
	}
}

func TriggerConfigToProto(tc *models.TriggerConfig) *investigationspb.TriggerConfig {
	if tc == nil {
		return nil
	}
	return &investigationspb.TriggerConfig{
		Id:          tc.ID,
		Name:        tc.Name,
		Description: tc.Description,
		Icon:        tc.Icon,
		Enabled:     tc.Enabled,
		AutoFix:     tc.AutoFix,
		Threshold:   tc.Threshold,
		Unit:        tc.Unit,
		Condition:   triggerConditionToProto(tc.Condition),
	}
}

func TriggerConfigsMapToProto(tcs map[string]*models.TriggerConfig) map[string]*investigationspb.TriggerConfig {
	if tcs == nil {
		return nil
	}
	result := make(map[string]*investigationspb.TriggerConfig, len(tcs))
	for k, v := range tcs {
		result[k] = TriggerConfigToProto(v)
	}
	return result
}
