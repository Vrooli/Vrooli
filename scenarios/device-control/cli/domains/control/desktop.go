package control

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	connectrpc "connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// DesktopGroup uses the local owner transport directly. API discovery must not
// replace an explicitly selected owner socket with the scenario API host.
func DesktopGroup() cliapp.SubcommandGroup {
	group := cliapp.SubcommandGroup{Name: "desktop", Description: "Use exact desktop sessions through local owner IPC"}
	for _, operation := range []string{"open", "applications", "observe", "resolve", "run-flow", "promote-flow", "get-saved-flow", "run-saved-flow", "act", "stop", "read-cleanup", "list-admissions", "reconcile-open", "capture-activation", "read-activation"} {
		op := operation
		flags := []cliapp.Flag{{Name: "socket", Required: true, Description: "absolute path to the destination owner Unix socket"}, {Name: "request", Required: true, Description: "file containing the typed owner request JSON"}}
		flags = append(flags, cliapp.Flag{Name: "access-token-file", Description: "private file containing an account access token; selects account authority"})
		if op == "observe" {
			flags = append(flags, cliapp.Flag{Name: "output", Required: true, Description: "new PNG file; existing files are never overwritten"})
		}
		group.Subcommands = append(group.Subcommands, cliapp.Command{Name: op, Description: "Desktop owner " + op, Args: cliapp.ArgSchema{Flags: flags}, RunCtx: func(ctx cliapp.RunContext) error {
			deadline, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			output := ""
			if op == "observe" {
				output = ctx.Flag("output")
			}
			result, err := desktopCallAuthorized(deadline, ctx.Flag("socket"), op, ctx.Flag("request"), output, ctx.Flag("access-token-file"))
			if err != nil {
				return err
			}
			data, err := protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: op == "read-cleanup" || op == "list-admissions"}.Marshal(result)
			if err != nil {
				return err
			}
			return emit(ctx, data, "Desktop "+op)
		}})
	}
	return group
}

func desktopCall(ctx context.Context, socket, op, requestPath, output string) (proto.Message, error) {
	return desktopCallAuthorized(ctx, socket, op, requestPath, output, "")
}

func desktopCallAuthorized(ctx context.Context, socket, op, requestPath, output, tokenPath string) (proto.Message, error) {
	if !filepath.IsAbs(socket) {
		return nil, fmt.Errorf("socket must be an explicit absolute Unix path")
	}
	requestInfo, err := os.Lstat(requestPath)
	if err != nil {
		return nil, err
	}
	if !requestInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("request must be a regular JSON file")
	}
	file, err := os.Open(requestPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("request must be a regular JSON file")
	}
	data, err := io.ReadAll(io.LimitReader(file, 128*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 128*1024 {
		return nil, fmt.Errorf("request exceeds 128 KiB")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	httpClient := &http.Client{Transport: transport, Timeout: 40 * time.Second}
	var client desktopv1connect.DesktopAccountServiceClient = desktopv1connect.NewDesktopOwnerServiceClient(httpClient, "http://desktop-owner", connectrpc.WithReadMaxBytes(33*1024*1024))
	if tokenPath != "" {
		token, err := readDesktopAccessToken(tokenPath)
		if err != nil {
			return nil, err
		}
		auth := connectrpc.UnaryInterceptorFunc(func(next connectrpc.UnaryFunc) connectrpc.UnaryFunc {
			return func(ctx context.Context, request connectrpc.AnyRequest) (connectrpc.AnyResponse, error) {
				request.Header().Set("Authorization", "Bearer "+token)
				response, err := next(ctx, request)
				if err != nil {
					return nil, connectrpc.NewError(connectrpc.CodeOf(err), fmt.Errorf("account desktop operation refused or incomplete; inspect state before retrying"))
				}
				return response, nil
			}
		})
		client = desktopv1connect.NewDesktopAccountServiceClient(httpClient, "http://desktop-owner", connectrpc.WithReadMaxBytes(33*1024*1024), connectrpc.WithInterceptors(auth))
	}
	switch op {
	case "capture-activation":
		request := &desktopv1.OwnerStopRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.CaptureActivation(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "read-activation":
		request := &desktopv1.OwnerReadActivationRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.ReadActivation(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil

	case "reconcile-open":
		request := &desktopv1.OwnerOpenRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.ReconcileOpen(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil

	case "list-admissions":
		request := &desktopv1.OwnerListAdmissionsRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.ListAdmissions(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil

	case "read-cleanup":
		request := &desktopv1.OwnerStopRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.ReadCleanup(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil

	case "promote-flow":
		request := &desktopv1.OwnerPromoteFlowRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.PromoteFlow(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "get-saved-flow":
		request := &desktopv1.OwnerGetSavedFlowRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.GetSavedFlow(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "run-saved-flow":
		request := &desktopv1.OwnerRunSavedFlowRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.RunSavedFlow(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "run-flow":
		request := &desktopv1.OwnerRunFlowRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.RunFlow(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil

	case "applications":
		request := &desktopv1.OwnerApplicationsRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.Applications(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "resolve":
		request := &desktopv1.OwnerResolveRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.Resolve(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "open":
		request := &desktopv1.OwnerOpenRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.Open(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "observe":
		request := &desktopv1.OwnerObserveRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		if output == "" {
			return nil, fmt.Errorf("output is required")
		}
		result, err := client.Observe(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		dimensions, err := png.DecodeConfig(bytes.NewReader(result.Msg.Png))
		if err != nil || dimensions.Width != int(result.Msg.Width) || dimensions.Height != int(result.Msg.Height) {
			return nil, fmt.Errorf("owner returned invalid PNG dimensions")
		}
		out, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, err
		}
		_, writeErr := out.Write(result.Msg.Png)
		closeErr := out.Close()
		if writeErr != nil {
			os.Remove(output)
			return nil, writeErr
		}
		if closeErr != nil {
			os.Remove(output)
			return nil, closeErr
		}
		result.Msg.Png = nil // Pixels belong in the explicit file, never terminal logs.
		return result.Msg, nil
	case "act":
		request := &desktopv1.OwnerActRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.Act(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	case "stop":
		request := &desktopv1.OwnerStopRequest{}
		if err = protojson.Unmarshal(data, request); err != nil {
			return nil, err
		}
		result, err := client.Stop(ctx, connectrpc.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return result.Msg, nil
	}
	return nil, fmt.Errorf("unknown desktop operation")
}

// Read credentials from a bounded private regular file, never command arguments
// or ambient HTTP configuration. Account failure never falls back to local IPC.
func readDesktopAccessToken(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("access token must be a private regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot read access token file")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Mode().Perm()&0o077 != 0 || !os.SameFile(info, opened) {
		return "", fmt.Errorf("access token file changed or is not private")
	}
	data, err := io.ReadAll(io.LimitReader(file, 16*1024+1))
	if err != nil || len(data) > 16*1024 {
		return "", fmt.Errorf("invalid access token file")
	}
	token := strings.TrimSpace(string(data))
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", fmt.Errorf("invalid access token file")
	}
	return token, nil
}
