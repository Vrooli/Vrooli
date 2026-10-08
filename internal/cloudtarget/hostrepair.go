package cloudtarget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/shell"
)

// BrokerClient is the privilege broker seam.
type BrokerClient interface {
	Available() bool
	Do(ctx context.Context, req privilegebroker.Request) (privilegebroker.Result, error)
}

// HostRepairRequest names one broker action and its typed subject. Subject is
// the JSON object of subject fields exactly as the broker request carries
// them (for example {"apt":{"packages":["jq"]}}); no other field is accepted.
type HostRepairRequest struct {
	Action    string
	Subject   []byte
	RequestID string
	// Effect is optional: when OperationID is set the repair is fenced and
	// receipted like every other target effect.
	Effect EffectRequest
}

// HostRepairResult is the broker's typed result plus how it was executed.
type HostRepairResult struct {
	Action    string                 `json:"action"`
	Executor  string                 `json:"executor"`
	Result    privilegebroker.Result `json:"result"`
	Changed   bool                   `json:"changed"`
	RequestID string                 `json:"request_id"`
}

// BuildBrokerRequest assembles and validates the broker request. It is the
// only path from cloud input to a privileged action, and it fails closed on
// any unknown field or unlisted action.
func BuildBrokerRequest(req HostRepairRequest) (privilegebroker.Request, error) {
	action := strings.TrimSpace(req.Action)
	if action == "" {
		return privilegebroker.Request{}, refuse(CodeActionNotAllowed, "action is required")
	}
	var broker privilegebroker.Request
	subject := bytes.TrimSpace(req.Subject)
	if len(subject) == 0 {
		subject = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(subject))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&broker); err != nil {
		return privilegebroker.Request{}, refuse(CodeActionNotAllowed, "subject is not a typed broker subject: %v", err)
	}
	if broker.Version != "" || broker.RequestID != "" || broker.Action != "" {
		return privilegebroker.Request{}, refuse(CodeActionNotAllowed, "subject must not carry version, request_id or action")
	}
	broker.Version = privilegebroker.ProtocolVersion
	broker.Action = action
	broker.RequestID = strings.TrimSpace(req.RequestID)
	if broker.RequestID == "" {
		broker.RequestID = "cloud-target-" + time.Now().UTC().Format("20060102T150405") + "-" + randomSuffix()
	}
	if err := privilegebroker.Validate(broker); err != nil {
		return privilegebroker.Request{}, refuse(CodeActionNotAllowed, "%s", err.Error()).withDetails(map[string]any{"action": action, "reason": err.Error()})
	}
	return broker, nil
}

// HostRepair validates the action, then either delegates a scoped process
// stop to the lifecycle owner as the workload's own user, or sends the
// request to the privilege broker. No other execution path exists.
func (s *Store) HostRepair(ctx context.Context, req HostRepairRequest, client BrokerClient, runner shell.Runner) (HostRepairResult, *EffectResult, error) {
	broker, err := BuildBrokerRequest(req)
	if err != nil {
		return HostRepairResult{}, nil, err
	}
	run := func(ctx context.Context) (HostRepairResult, error) {
		if broker.Action == privilegebroker.ActionProcessStopScoped {
			return runScopedStop(ctx, broker, runner)
		}
		if client == nil || !client.Available() {
			return HostRepairResult{}, fail(CodeBrokerUnavailable, "privilege broker is not available; run vrooli setup to install it").withDetails(map[string]any{"action": broker.Action})
		}
		result, err := client.Do(ctx, broker)
		if err != nil {
			return HostRepairResult{}, fail(CodeHostActionFailed, "%v", err).withDetails(map[string]any{"action": broker.Action})
		}
		out := HostRepairResult{Action: broker.Action, Executor: "privilege-broker", Result: result, Changed: result.Changed, RequestID: broker.RequestID}
		if result.Status == "failed" || result.Status == "unavailable" {
			return out, fail(CodeHostActionFailed, "%s: %s", broker.Action, result.Code).withDetails(map[string]any{"action": broker.Action, "broker_code": result.Code, "broker_status": result.Status})
		}
		return out, nil
	}
	if strings.TrimSpace(req.Effect.OperationID) == "" {
		result, err := run(ctx)
		return result, nil, err
	}
	effect := req.Effect
	effect.Verb = "host.repair"
	effect.Input = map[string]any{"action": broker.Action, "subject": json.RawMessage(bytes.TrimSpace(req.Subject))}
	var result HostRepairResult
	effectResult, err := s.RunEffect(ctx, effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		var runErr error
		result, runErr = run(ctx)
		details := map[string]any{"action": broker.Action, "executor": result.Executor, "broker_status": result.Result.Status, "broker_code": result.Result.Code, "request_id": broker.RequestID}
		if runErr != nil {
			return details, OutcomeFailed, runErr
		}
		if result.Changed {
			return details, OutcomeSucceeded, nil
		}
		return details, OutcomeUnchanged, nil
	})
	return result, &effectResult, err
}

func runScopedStop(ctx context.Context, broker privilegebroker.Request, runner shell.Runner) (HostRepairResult, error) {
	name, args, err := privilegebroker.ProcessStopArgs(broker)
	if err != nil {
		return HostRepairResult{}, refuse(CodeActionNotAllowed, "%s", err.Error())
	}
	if runner == nil {
		runner = shell.OSRunner{}
	}
	out := HostRepairResult{Action: broker.Action, Executor: "lifecycle-owner", RequestID: broker.RequestID}
	if output, err := runner.Run(ctx, name, args...); err != nil {
		out.Result = privilegebroker.NewFailure(broker.RequestID, broker.Action, "scenario_stop_failed")
		return out, fail(CodeHostActionFailed, "%s: %v: %s", broker.Action, err, strings.TrimSpace(string(output))).withDetails(map[string]any{"action": broker.Action})
	}
	out.Result = privilegebroker.Result{Version: privilegebroker.ProtocolVersion, RequestID: broker.RequestID, Action: broker.Action, Status: "completed", Changed: true, Evidence: privilegebroker.Evidence{Detail: fmt.Sprintf("%s %s", name, strings.Join(args, " "))}}
	out.Changed = true
	return out, nil
}
