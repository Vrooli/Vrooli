package control

import (
	"context"
	"net/http"
	"time"

	connectrpc "connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices"
	devicesv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices/devices_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func inspectionCommands(core *cliapp.ScenarioApp) []cliapp.Command {
	return []cliapp.Command{
		command("inspect", "Read hardware, active WebView, storage, and default Home without device writes", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "transport", Description: "exact observed transport profile"}}}, func(ctx cliapp.RunContext) error {
			transport, base := cliapp.NewConnectHTTPClient(core)
			client := devicesv1connect.NewDeviceServiceClient(transport, base)
			deadline, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			response, err := client.InspectDevice(deadline, connectrpc.NewRequest(&devicesv1.InspectDeviceRequest{DeviceId: ctx.Positional("id"), Transport: ctx.Flag("transport")}))
			if err != nil {
				return err
			}
			data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
			if err != nil {
				return err
			}
			return emit(ctx, data, "Device runtime inspection")
		}),
		command("observe", "Capture a redacted still frame using a held lease and the existing flow owner", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "lease", Required: true}, {Name: "actor", Required: true}, {Name: "transport"}}}, func(ctx cliapp.RunContext) error {
			flow := map[string]any{"id": "operator-observation", "transport": ctx.Flag("transport"), "require_unlocked": true, "max_duration_ms": 15000, "steps": []map[string]string{{"id": "screen", "kind": "observe"}}}
			data, err := core.Request(http.MethodPost, "/flows/run", nil, map[string]any{"device_id": ctx.Positional("id"), "actor": ctx.Flag("actor"), "lease_token": ctx.Flag("lease"), "flow": flow})
			if err != nil {
				return err
			}
			return emit(ctx, data, "Device frame evidence")
		}),
	}
}
