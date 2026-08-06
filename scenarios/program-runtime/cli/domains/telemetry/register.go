package telemetry

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	telemetryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry/telemetry_v1connect"
)

const GroupName = "telemetry"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := telemetryconnect.NewTelemetryServiceClient(httpClient, baseURL)
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"TelemetryService.ListEvents": cliapp.ProtoList(func(ctx cliapp.OperationContext) (*telemetryv1.ListEventsResponse, error) {
		r, e := client.ListEvents(context.Background(), connect.NewRequest(&telemetryv1.ListEventsRequest{SessionId: ctx.Flag("session-id"), Kind: parseKind(ctx.Flag("kind"))}))
		if e != nil {
			return nil, cliapp.WrapAPIError("list telemetry events", e, nil)
		}
		return r.Msg, nil
	}, func(_ cliapp.OperationContext, r *telemetryv1.ListEventsResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d typed event(s).", len(r.Events))}}
	})})
}

func parseKind(value string) telemetryv1.EventKind {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value != "" && !strings.HasPrefix(value, "EVENT_KIND_") {
		value = "EVENT_KIND_" + value
	}
	values := telemetryv1.EventKind(0).Descriptor().Values()
	for i := 0; i < values.Len(); i++ {
		kind := values.Get(i)
		if string(kind.Name()) == value {
			return telemetryv1.EventKind(kind.Number())
		}
	}
	return telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED
}
