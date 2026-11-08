package protoconv

import (
	"agent-manager/internal/stepback"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// StepBackTriggersToProto converts transcript step-back triggers for the run
// tree and the effort board.
func StepBackTriggersToProto(triggers []stepback.Trigger) []*pb.StepBackTrigger {
	if len(triggers) == 0 {
		return nil
	}
	out := make([]*pb.StepBackTrigger, 0, len(triggers))
	for _, trigger := range triggers {
		out = append(out, &pb.StepBackTrigger{Id: trigger.ID, Mechanism: trigger.Mechanism, Detail: trigger.Detail, Remedy: trigger.Remedy})
	}
	return out
}
