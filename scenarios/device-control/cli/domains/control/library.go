package control

import (
	"context"
	"fmt"
	"os"
	"strconv"

	connectrpc "connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows/flows_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func savedFlowCommands(core *cliapp.ScenarioApp) []cliapp.Command {
	out := []cliapp.Command{}
	for _, name := range []string{"list", "get", "save", "replay"} {
		fields := map[string][]string{"list": {"device-id", "context-key"}, "get": {"id", "version"}, "save": {"run-id", "device-id", "context-key", "id", "expected-version"}, "replay": {"id", "version", "device-id", "context-key", "actor"}}[name]
		flags := []cliapp.Flag{}
		for _, field := range fields {
			required := true
			if name == "save" && (field == "id" || field == "expected-version") || name == "get" && field == "version" {
				required = false
			}
			flags = append(flags, cliapp.Flag{Name: field, Required: required, Description: field})
		}
		operation := name
		out = append(out, command(name, "Use the durable device-scoped flow library", cliapp.ArgSchema{Flags: flags}, func(ctx cliapp.RunContext) error {
			transport, base := cliapp.NewConnectHTTPClient(core)
			client := rpc.NewFlowServiceClient(transport, base)
			version := int32(0)
			field := "version"
			if operation == "save" {
				field = "expected-version"
			}
			if raw := ctx.Flag(field); raw != "" {
				n, err := strconv.ParseInt(raw, 10, 32)
				if err != nil || n < 0 {
					return fmt.Errorf("invalid %s", field)
				}
				version = int32(n)
			}
			var result proto.Message
			switch operation {
			case "list":
				r, e := client.ListSavedFlows(context.Background(), connectrpc.NewRequest(&pb.ListSavedFlowsRequest{DeviceId: ctx.Flag("device-id"), ContextKey: ctx.Flag("context-key")}))
				if e != nil {
					return e
				}
				result = r.Msg
			case "get":
				r, e := client.GetSavedFlow(context.Background(), connectrpc.NewRequest(&pb.GetSavedFlowRequest{Id: ctx.Flag("id"), Version: version}))
				if e != nil {
					return e
				}
				result = r.Msg
			case "save":
				r, e := client.SaveValidatedFlow(context.Background(), connectrpc.NewRequest(&pb.SaveValidatedFlowRequest{Id: ctx.Flag("id"), ExpectedVersion: version, RunId: ctx.Flag("run-id"), DeviceId: ctx.Flag("device-id"), ContextKey: ctx.Flag("context-key")}))
				if e != nil {
					return e
				}
				result = r.Msg
			case "replay":
				r, e := client.RunSavedFlow(context.Background(), connectrpc.NewRequest(&pb.RunSavedFlowRequest{Id: ctx.Flag("id"), Version: version, DeviceId: ctx.Flag("device-id"), ContextKey: ctx.Flag("context-key"), Actor: ctx.Flag("actor")}))
				if e != nil {
					return e
				}
				result = r.Msg
			}
			body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(result)
			if err != nil {
				return err
			}
			return emit(ctx, body, "Saved flow "+operation)
		}))
	}
	return out
}

func validateLibraryCandidate(ctx cliapp.RunContext, core *cliapp.ScenarioApp) error {
	body, err := os.ReadFile(ctx.Flag("file"))
	if err != nil {
		return err
	}
	flow := &pb.Flow{}
	if err := protojson.Unmarshal(body, flow); err != nil {
		return err
	}
	n := int64(0)
	if raw := ctx.Flag("expected-version"); raw != "" {
		n, err = strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid expected-version")
		}
	}
	transport, base := cliapp.NewConnectHTTPClient(core)
	result, err := rpc.NewFlowServiceClient(transport, base).ValidateFlow(context.Background(), connectrpc.NewRequest(&pb.ValidateFlowRequest{Flow: flow, StrategyId: ctx.Flag("strategy"), BaselineId: ctx.Flag("baseline-id"), ExpectedVersion: int32(n), RequireAssertion: ctx.Flag("require-assertion") == "true"}))
	if err != nil {
		return err
	}
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(result.Msg)
	if err != nil {
		return err
	}
	return emit(ctx, data, "Flow validation")
}
