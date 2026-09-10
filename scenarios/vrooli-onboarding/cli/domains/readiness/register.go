package readiness

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
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

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Readiness", Commands: []cliapp.Command{
		{Name: "readiness", Description: "Validate required onboarding items", NeedsAPI: true, Run: func(args []string) error {
			if len(args) > 0 && args[0] == "acknowledge-degraded" {
				return acknowledgeDegraded(core, args[1:])
			}
			return run(core, args)
		}},
	}}
}

// ManifestHandlers keeps the readiness exit-code contract on the production
// command path. The manifest still declares the typed Connect binding and its
// governance; this small adapter adds the domain rule that a generated
// response-only primitive cannot express: required blockers and unacknowledged
// degradation must produce a distinct non-zero process result.
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
		"status": cliapp.ProtoOperationalWithExit(
			readinessResponse,
			func(_ cliapp.OperationContext, response *readinessv1.GetReadinessResponse) cliapp.OperationalReport {
				return operationalReport(response)
			},
			func(_ cliapp.OperationContext, response *readinessv1.GetReadinessResponse) error {
				return readinessExit(response)
			},
		),
	}
}

func readinessResponse(ctx cliapp.OperationContext) (*readinessv1.GetReadinessResponse, error) {
	target := "local"
	if ctx.FlagDeclared("target") && strings.TrimSpace(ctx.Flag("target")) != "" {
		target = strings.TrimSpace(ctx.Flag("target"))
	}
	request, err := protojson.Marshal(&readinessv1.GetReadinessRequest{Target: target})
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

// blocker mirrors the API's metadata-only completion blocker. It never carries
// a credential value.
type blocker struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation"`
}

// acknowledgeDegraded records the operator's acceptance of the exact set of
// degraded optional items readiness reports now. The digest identifies that
// set, so an acknowledgement cannot carry over to a different gap.
func acknowledgeDegraded(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("readiness acknowledge-degraded")
	digest := fs.String("digest", "", "Degraded-set digest from `readiness` output; omit to read the current one")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	value := strings.TrimSpace(*digest)
	if value == "" {
		body, err := core.RequestRoot("POST", readinessProcedure, nil, []byte(`{}`))
		if err != nil {
			return err
		}
		current, err := decodeReadinessStatus(body)
		if err != nil {
			return err
		}
		value = strings.TrimSpace(current.DegradedDigest)
		if value == "" {
			return &ExitError{Code: 2, Text: "there is no degraded set to acknowledge"}
		}
	}
	payload, err := json.Marshal(map[string]string{"readinessDigest": value})
	if err != nil {
		return err
	}
	response, err := core.RequestRoot("POST", acknowledgeProcedure, nil, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err := os.Stdout.Write(append(response, '\n'))
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Recorded the degraded acknowledgement for digest", value)
	return err
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("readiness")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	return runWithOutput(core, *jsonOutput, os.Stdout, os.Stderr)
}

func runWithOutput(core *cliapp.ScenarioApp, jsonOutput bool, stdout, stderr io.Writer) error {
	body, err := core.RequestRoot("POST", readinessProcedure, nil, []byte(`{}`))
	if err != nil {
		return err
	}
	if jsonOutput {
		if _, err := stdout.Write(append(body, '\n')); err != nil {
			return err
		}
	} else {
		var value any
		if err := json.Unmarshal(body, &value); err != nil {
			return fmt.Errorf("decode readiness response: %w", err)
		}
		pretty, _ := json.MarshalIndent(value, "", "  ")
		if _, err := fmt.Fprintln(stdout, string(pretty)); err != nil {
			return err
		}
	}
	// The verdict comes from the API's typed blockers rather than from a second
	// derivation here. Two implementations of the same rule are how the wizard
	// and the completion marker came to disagree in the first place.
	response, err := decodeReadinessStatus(body)
	if err != nil {
		return err
	}
	messageOut := stdout
	if jsonOutput {
		// stdout must remain one parseable JSON document. Human diagnostics and
		// progress belong on stderr when automation requested JSON.
		messageOut = stderr
	}
	for _, item := range response.Blockers {
		if _, err := fmt.Fprintf(messageOut, "Blocked: %s %s — %s. Next: %s\n", item.Kind, item.Name, item.Reason, item.Remediation); err != nil {
			return err
		}
	}
	if len(response.Blockers) > 0 {
		return &ExitError{Code: 2, Text: fmt.Sprintf("configuration is not complete: %d blocking item(s) remain; blockers: %s", len(response.Blockers), blockerDetails(response.Blockers))}
	}
	if len(response.Degraded) > 0 && !response.DegradedAcknowledged {
		for _, item := range response.Degraded {
			if _, err := fmt.Fprintf(messageOut, "Degraded: %s %s — %s. Next: %s\n", item.Kind, item.Name, item.Reason, item.Remediation); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(messageOut, "Accept them with: vrooli-onboarding readiness acknowledge-degraded --digest %s\n", response.DegradedDigest); err != nil {
			return err
		}
		return &ExitError{Code: 2, Text: fmt.Sprintf("configuration is not complete: %d optional item(s) need an explicit acknowledgement; degraded: %s", len(response.Degraded), blockerDetails(response.Degraded))}
	}
	return nil
}

type readinessStatus struct {
	Blockers             []blocker
	Degraded             []blocker
	DegradedDigest       string
	DegradedAcknowledged bool
}

func decodeReadinessStatus(body []byte) (readinessStatus, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return readinessStatus{}, fmt.Errorf("decode readiness status: %w", err)
	}
	read := func(names ...string) json.RawMessage {
		for _, name := range names {
			if value, ok := raw[name]; ok {
				return value
			}
		}
		return nil
	}
	var result readinessStatus
	if value := read("blockers"); len(value) > 0 {
		if err := json.Unmarshal(value, &result.Blockers); err != nil {
			return readinessStatus{}, fmt.Errorf("decode readiness blockers: %w", err)
		}
	}
	if value := read("degraded"); len(value) > 0 {
		if err := json.Unmarshal(value, &result.Degraded); err != nil {
			return readinessStatus{}, fmt.Errorf("decode readiness degraded items: %w", err)
		}
	}
	if value := read("degraded_digest", "degradedDigest"); len(value) > 0 {
		if err := json.Unmarshal(value, &result.DegradedDigest); err != nil {
			return readinessStatus{}, fmt.Errorf("decode readiness digest: %w", err)
		}
	}
	if value := read("degraded_acknowledged", "degradedAcknowledged"); len(value) > 0 {
		if err := json.Unmarshal(value, &result.DegradedAcknowledged); err != nil {
			return readinessStatus{}, fmt.Errorf("decode readiness acknowledgement: %w", err)
		}
	}
	return result, nil
}

// blockerDetails keeps the process error useful to non-interactive callers
// that do not retain the CLI's stdout. It contains only the metadata already
// exposed by the readiness API; credentials and other secret values never
// enter this string.
func blockerDetails(items []blocker) string {
	details := make([]string, 0, len(items))
	for _, item := range items {
		details = append(details, fmt.Sprintf("%s %s — %s; next: %s", item.Kind, item.Name, item.Reason, item.Remediation))
	}
	return strings.Join(details, " | ")
}
