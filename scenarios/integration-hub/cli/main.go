package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

func main() {
	base := os.Getenv("INTEGRATION_HUB_API_BASE")
	if base == "" {
		base = "http://localhost:" + os.Getenv("API_PORT")
	}
	if strings.HasSuffix(base, ":") {
		base += "15000"
	}
	identity := os.Getenv("VROOLI_IDENTITY")
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	client := commonv1connect.NewConnectionServiceClient(http.DefaultClient, base)
	ctx := context.Background()
	var result any
	var err error
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		connector := fs.String("connector", "", "connector id")
		_ = fs.Parse(args[1:])
		req := connect.NewRequest(&commonv1.ListConnectionsRequest{ConnectorId: *connector})
		withIdentity(req, identity)
		response, callErr := client.ListConnections(ctx, req)
		if callErr == nil {
			result = response.Msg
		}
		err = callErr
	case "get", "probe", "refresh", "rotate", "revoke", "delete":
		fs := flag.NewFlagSet(args[0], flag.ExitOnError)
		id := fs.String("id", "", "connection id")
		credentialStdin := fs.Bool("credential-stdin", false, "read the write-only credential value from stdin")
		_ = fs.Parse(args[1:])
		value, readErr := credentialValue(*credentialStdin)
		if readErr != nil {
			fmt.Fprintln(os.Stderr, readErr)
			os.Exit(1)
		}
		request := connect.NewRequest(&commonv1.ConnectionMutationRequest{ConnectionId: *id, CredentialValue: value})
		withIdentity(request, identity)
		result, err = mutation(ctx, client, args[0], request)
	case "bind", "unbind":
		fs := flag.NewFlagSet(args[0], flag.ExitOnError)
		id := fs.String("id", "", "connection id")
		scenario := fs.String("scenario", "", "scenario slug")
		bindingContext := fs.String("context", "", "binding context")
		var requiredScopes stringList
		fs.Var(&requiredScopes, "required-scope", "scope required by the target scenario; may be repeated")
		_ = fs.Parse(args[1:])
		request := connect.NewRequest(&commonv1.ConnectionMutationRequest{ConnectionId: *id, BindingScenarioSlug: *scenario, BindingContext: *bindingContext, RequiredScopes: requiredScopes})
		withIdentity(request, identity)
		result, err = mutation(ctx, client, args[0], request)
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		id := fs.String("id", "", "connection id")
		display := fs.String("display-name", "", "display name")
		credentialStdin := fs.Bool("credential-stdin", false, "read the write-only credential value from stdin")
		_ = fs.Parse(args[1:])
		value, readErr := credentialValue(*credentialStdin)
		if readErr != nil {
			fmt.Fprintln(os.Stderr, readErr)
			os.Exit(1)
		}
		request := connect.NewRequest(&commonv1.ConnectionMutationRequest{ConnectionId: *id, ConnectorId: "openrouter", DisplayName: *display, CredentialValue: value})
		withIdentity(request, identity)
		response, callErr := client.CreateConnection(ctx, request)
		if callErr == nil {
			result = response.Msg
		}
		err = callErr
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("scope cannot be empty")
	}
	*s = append(*s, value)
	return nil
}

func credentialValue(fromStdin bool) (string, error) {
	if !fromStdin {
		return "", nil
	}
	value, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read credential from stdin: %w", err)
	}
	return strings.TrimSpace(string(value)), nil
}

func withIdentity[T any](request *connect.Request[T], identity string) {
	if identity != "" {
		request.Header().Set("X-Vrooli-Identity", identity)
	}
}
func mutation(ctx context.Context, client commonv1connect.ConnectionServiceClient, operation string, request *connect.Request[commonv1.ConnectionMutationRequest]) (any, error) {
	switch operation {
	case "probe":
		response, err := client.ProbeConnection(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "refresh":
		response, err := client.RefreshConnection(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "rotate":
		response, err := client.RotateConnection(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "bind":
		response, err := client.BindConnection(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "unbind":
		response, err := client.UnbindConnection(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "revoke":
		response, err := client.RevokeConnection(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "delete":
		response, err := client.DeleteConnection(ctx, request)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	case "get":
		getRequest := connect.NewRequest(&commonv1.GetConnectionRequest{ConnectionId: request.Msg.GetConnectionId()})
		withIdentity(getRequest, request.Header().Get("X-Vrooli-Identity"))
		response, err := client.GetConnection(ctx, getRequest)
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	default:
		return nil, fmt.Errorf("unknown operation %s", operation)
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "integration-hub list|get|create|probe|refresh|rotate|bind|unbind|revoke|delete")
}
