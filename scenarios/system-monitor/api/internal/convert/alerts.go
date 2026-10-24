package convert

import (
	"encoding/json"

	alertspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AlertToProto projects a persisted alert. Rule metadata lives in Details
// for condition alerts and is lifted into first-class fields.
func AlertToProto(a *models.Alert) *alertspb.Alert {
	if a == nil {
		return nil
	}
	pb := &alertspb.Alert{
		Id:          a.ID,
		Type:        a.Type,
		Severity:    a.Severity,
		Message:     a.Message,
		MetricName:  a.MetricName,
		MetricValue: a.MetricValue,
		AckedBy:     a.AckedBy,
	}
	if !a.Timestamp.IsZero() {
		pb.OpenedAt = timestamppb.New(a.Timestamp)
	}
	if a.ResolvedAt != nil {
		pb.ResolvedAt = timestamppb.New(*a.ResolvedAt)
	}
	if a.AckedAt != nil {
		pb.AckedAt = timestamppb.New(*a.AckedAt)
	}
	pb.RuleId, _ = a.Details["rule_id"].(string)
	pb.Key, _ = a.Details["key"].(string)
	pb.Escalated, _ = a.Details["escalated"].(bool)
	pb.AnomalyId, _ = a.Details["anomaly_id"].(string)
	if threshold, ok := a.Details["threshold"].(float64); ok {
		pb.Threshold = threshold
	}
	pb.Details = detailsStruct(a.Details)
	return pb
}

// detailsStruct normalizes details through JSON so in-memory values (ints,
// typed slices) and values decoded from storage project identically.
func detailsStruct(details map[string]interface{}) *structpb.Struct {
	if len(details) == 0 {
		return nil
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return nil
	}
	var normalized map[string]interface{}
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil
	}
	out, err := structpb.NewStruct(normalized)
	if err != nil {
		return nil
	}
	return out
}

// RuleStatusesToProto projects the evaluator's rule states.
func RuleStatusesToProto(statuses []services.RuleStatus) []*alertspb.RuleState {
	out := make([]*alertspb.RuleState, 0, len(statuses))
	for _, s := range statuses {
		state := &alertspb.RuleState{RuleId: s.RuleID, Status: s.Status, Reason: s.Reason, ConditionActive: s.ConditionActive}
		if !s.PendingSince.IsZero() {
			state.PendingSince = timestamppb.New(s.PendingSince)
		}
		if !s.EvaluatedAt.IsZero() {
			state.EvaluatedAt = timestamppb.New(s.EvaluatedAt)
		}
		out = append(out, state)
	}
	return out
}
