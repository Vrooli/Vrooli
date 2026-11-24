package readiness

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	readinessProcedure   = "/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/GetReadiness"
	acknowledgeProcedure = "/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/AcknowledgeDegradedReadiness"
)

// ExitError preserves the machine-readable readiness contract at the process
// boundary. A required item that is not ready is deliberately distinct from a
// transport or command error.
type ExitError struct {
	Code int
	Text string
}

func (e *ExitError) Error() string { return e.Text }
func (e *ExitError) ExitCode() int { return e.Code }

// ManifestHandlers applies readiness's domain exit contract to the one active
// manifest-driven command path. Acknowledge uses the same primitive as the
// manifest and resolves an omitted digest from the authoritative read first.
func ManifestHandlers(core *cliapp.ScenarioApp) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"ReadinessService.GetReadiness": cliapp.ProtoOperationalWithExit(
			readinessResponse,
			func(_ cliapp.OperationContext, response *readinessv1.GetReadinessResponse) cliapp.OperationalReport {
				return operationalReport(response)
			},
			func(_ cliapp.OperationContext, response *readinessv1.GetReadinessResponse) error {
				return readinessExit(response)
			},
		),
		"ReadinessService.AcknowledgeDegradedReadiness": cliapp.ProtoMutation(
			acknowledgeResponse,
			acknowledgementReport,
		),
	}
}

func readinessTarget(ctx cliapp.OperationContext) string {
	target := "local"
	if ctx.FlagDeclared("target") && strings.TrimSpace(ctx.Flag("target")) != "" {
		target = strings.TrimSpace(ctx.Flag("target"))
	}
	return target
}

func readinessResponse(ctx cliapp.OperationContext) (*readinessv1.GetReadinessResponse, error) {
	request, err := protojson.Marshal(&readinessv1.GetReadinessRequest{Target: readinessTarget(ctx)})
	if err != nil {
		return nil, fmt.Errorf("encode readiness request: %w", err)
	}
	body, err := ctx.Core().RequestRoot("POST", readinessProcedure, nil, request)
	if err != nil {
		return nil, err
	}
	response := new(readinessv1.GetReadinessResponse)
	if err := protojson.Unmarshal(body, response); err != nil {
		return nil, fmt.Errorf("decode readiness response: %w", err)
	}
	return response, nil
}

func acknowledgeResponse(ctx cliapp.OperationContext) (*readinessv1.AcknowledgeDegradedReadinessResponse, error) {
	target := readinessTarget(ctx)
	digest := strings.TrimSpace(ctx.Flag("digest"))
	if digest == "" {
		current, err := readinessResponse(ctx)
		if err != nil {
			return nil, err
		}
		digest = strings.TrimSpace(current.GetDegradedDigest())
		if digest == "" {
			return nil, &ExitError{Code: 2, Text: "there is no degraded set to acknowledge"}
		}
	}
	request := &readinessv1.AcknowledgeDegradedReadinessRequest{Target: target, ReadinessDigest: digest}
	body, err := protojson.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode readiness acknowledgement: %w", err)
	}
	responseBody, err := ctx.Core().RequestRoot("POST", acknowledgeProcedure, nil, body)
	if err != nil {
		return nil, err
	}
	response := new(readinessv1.AcknowledgeDegradedReadinessResponse)
	if err := protojson.Unmarshal(responseBody, response); err != nil {
		return nil, fmt.Errorf("decode readiness acknowledgement: %w", err)
	}
	return response, nil
}

func acknowledgementReport(_ cliapp.OperationContext, response *readinessv1.AcknowledgeDegradedReadinessResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{"Recorded degraded acknowledgement for digest " + response.GetReadinessDigest()},
		Changes: []string{"status: " + response.GetStatus()},
	}
}

func operationalReport(response *readinessv1.GetReadinessResponse) cliapp.OperationalReport {
	report := cliapp.OperationalReport{Status: []string{response.GetStatus().String()}}
	if target := strings.TrimSpace(response.GetTarget()); target != "" {
		report.Status = append(report.Status, "Target: "+target)
	}
	if len(response.GetBlockers()) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Required blockers", Items: protoBlockerDetails(response.GetBlockers())})
	}
	if len(response.GetDegraded()) > 0 && !response.GetDegradedAcknowledged() {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Unacknowledged degradation", Items: protoBlockerDetails(response.GetDegraded())})
		report.NextSteps = append(report.NextSteps, "Acknowledge the exact degraded digest before treating configuration as complete: vrooli-onboarding readiness acknowledge-degraded --digest "+response.GetDegradedDigest())
	}
	if len(report.NextSteps) == 0 {
		report.NextSteps = []string{"Use the apply and verification commands to inspect the authoritative configuration state."}
	}
	return report
}

func protoBlockerDetails(items []*sharedv1.CompletionBlocker) []string {
	details := make([]string, 0, len(items))
	for _, item := range items {
		details = append(details, fmt.Sprintf("%s %s — %s. Next: %s", item.GetKind(), item.GetName(), item.GetReason(), item.GetRemediation()))
	}
	return details
}

func readinessExit(response *readinessv1.GetReadinessResponse) error {
	if len(response.GetBlockers()) > 0 {
		return &ExitError{Code: 2, Text: fmt.Sprintf("configuration is not complete: %d blocking item(s) remain; blockers: %s", len(response.GetBlockers()), strings.Join(protoBlockerDetails(response.GetBlockers()), " | "))}
	}
	if len(response.GetDegraded()) > 0 && !response.GetDegradedAcknowledged() {
		return &ExitError{Code: 2, Text: fmt.Sprintf("configuration is not complete: %d optional item(s) need an explicit acknowledgement; degraded: %s", len(response.GetDegraded()), strings.Join(protoBlockerDetails(response.GetDegraded()), " | "))}
	}
	return nil
}
