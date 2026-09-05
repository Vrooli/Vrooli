package targets

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets/targets_v1connect"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/vrooli/cli-core/cliapp"
	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client targetsconnect.TargetCatalogServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: targetsconnect.NewTargetCatalogServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	response, err := h.client.List(context.Background(), connect.NewRequest(&targetsv1.ListRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("target list", err, nil)
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Target catalog: %s (%d locations)", response.Msg.GetState(), len(response.Msg.GetTargets()))},
		ResultsHeading: "Locations", Results: targetRows(response.Msg.GetTargets()),
		RetrievalHints: []string{fmt.Sprintf("%s target doctor <target-id>", support.CLIName)},
	}
	return cliapp.RenderProtoList(ctx, response.Msg, report)
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id, err := targetID(ctx, "get")
	if err != nil {
		return err
	}
	response, err := h.client.Get(context.Background(), connect.NewRequest(&targetsv1.GetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("target get", err, nil)
	}
	if response.Msg.GetTarget() == nil {
		return fmt.Errorf("target get returned no target for %q", id)
	}
	target := response.Msg.GetTarget()
	report := cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Target: %s", target.GetLabel())}, ResultsHeading: "Details",
		Results: targetDetails(target), RetrievalHints: []string{fmt.Sprintf("%s target doctor %s", support.CLIName, target.GetId())},
	}
	return cliapp.RenderProtoList(ctx, response.Msg, report)
}

func (h *handlers) doctor(ctx cliapp.RunContext) error {
	id, err := targetID(ctx, "doctor")
	if err != nil {
		return err
	}
	response, err := h.client.Doctor(context.Background(), connect.NewRequest(&targetsv1.DoctorRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("target doctor", err, nil)
	}
	if response.Msg.GetTarget() == nil {
		return fmt.Errorf("target doctor returned no target for %q", id)
	}
	target := response.Msg.GetTarget()
	next := targetText(response.Msg.GetTarget(), "recovery_action")
	if next == "" {
		next = fmt.Sprintf("%s session create --target %s", support.CLIName, target.GetId())
	}
	return cliapp.RenderProtoOperational(ctx, response.Msg, cliapp.OperationalReport{
		Status:    []string{response.Msg.GetSummary()},
		Triage:    []cliapp.TriageGroup{{Heading: "Readiness", Items: readinessRows(target)}},
		NextSteps: []string{next},
	})
}

func targetID(ctx cliapp.RunContext, command string) (string, error) {
	id := strings.TrimSpace(ctx.Positional("target-id"))
	if id == "" {
		return "", fmt.Errorf("usage: target %s <target-id>", command)
	}
	return id, nil
}

func targetRows(targets []*sharedv1.Target) []string {
	if len(targets) == 0 {
		return []string{"(no locations)"}
	}
	rows := make([]string, 0, len(targets))
	for _, target := range targets {
		platform := strings.Trim(strings.Join([]string{target.GetOs(), target.GetArch()}, "/"), "/")
		if platform == "" {
			platform = "platform unknown"
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s | state=%s | dispatchable=%t | %s", target.GetId(), target.GetKind(), target.GetLabel(), target.GetState(), target.GetDispatchable(), platform))
	}
	return rows
}

func targetDetails(target *sharedv1.Target) []string {
	return []string{
		fmt.Sprintf("ID: %s", target.GetId()), fmt.Sprintf("Kind: %s", target.GetKind()), fmt.Sprintf("Label: %s", target.GetLabel()),
		fmt.Sprintf("State: %s", target.GetState()), fmt.Sprintf("Dispatchable: %t", target.GetDispatchable()), fmt.Sprintf("Platform: %s/%s", target.GetOs(), target.GetArch()),
		fmt.Sprintf("Status: %s", target.GetStatus()), fmt.Sprintf("Survives restart: %t", target.GetSurvivesRestart()),
		fmt.Sprintf("Dispatch reason: %s", defaultText(targetText(target, "failure_rung"), "none")), fmt.Sprintf("Operator action: %s", defaultText(targetText(target, "recovery_action"), "none")),
	}
}

func targetText(target *sharedv1.Target, name string) string {
	if target == nil {
		return ""
	}
	field := target.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
	if field == nil || field.Kind() != protoreflect.StringKind {
		return ""
	}
	return target.ProtoReflect().Get(field).String()
}

func readinessRows(target *sharedv1.Target) []string {
	if len(target.GetReadiness()) == 0 {
		return []string{"No readiness facts were returned"}
	}
	rows := make([]string, 0, len(target.GetReadiness()))
	for _, fact := range target.GetReadiness() {
		rows = append(rows, fmt.Sprintf("%s: passed=%t (%s)", fact.GetLabel(), fact.GetPassed(), fact.GetDetail()))
	}
	return rows
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
