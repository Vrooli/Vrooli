package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/binaryfetch"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
	capabilities "web-console/internal/capabilities"
)

// installCapabilityRemote uses the existing Bridge relay. The relay performs
// owner authentication, node-write-scope admission, durable audit, and typed
// argv delivery before the node runs the resource installer.
func (s *Server) installCapabilityRemote(ctx context.Context, targetID, capabilityID string) (capabilities.LifecycleActionResult, error) {
	def, ok := capabilityDefinition(capabilityID)
	if !ok || def.DependencyKind != capabilities.DependencyResource || def.DependencySlug == "" {
		return capabilities.LifecycleActionResult{}, fmt.Errorf("capability %q has no governed resource installer", capabilityID)
	}
	target, ok := s.targetByID(strings.TrimSpace(targetID))
	if !ok || target.NodeID == "" {
		return capabilities.LifecycleActionResult{}, fmt.Errorf("remote target %q was not found", targetID)
	}
	if acquisition, found := resourceAcquisition(def.DependencySlug); found {
		if _, err := acquisition.Resolve(binaryfetch.Facts{"os": target.OS, "arch": target.Architecture}); err != nil {
			var unsupported *binaryfetch.UnsupportedError
			if errors.As(err, &unsupported) {
				return capabilities.LifecycleActionResult{CapabilityID: capabilityID, ActionKind: capabilities.ActionKindOperatorCommand, Status: "not_applicable", Message: unsupported.Error()}, nil
			}
		}
	}
	for _, fact := range target.Readiness {
		if fact.Identity != "capability:"+capabilityID {
			continue
		}
		if fact.State == "not_applicable" {
			return capabilities.LifecycleActionResult{CapabilityID: capabilityID, ActionKind: capabilities.ActionKindOperatorCommand, Status: "not_applicable", Message: fact.Detail}, nil
		}
		break
	}
	base := strings.TrimRight(strings.TrimSpace(target.BaseURL), "/")
	if base == "" {
		return capabilities.LifecycleActionResult{}, fmt.Errorf("Bridge URL is not configured")
	}
	client := relayconnect.NewRelayServiceClient(authenticatedHTTPClient(target.OwnerToken, target.ReauthToken), base)
	response, err := client.Call(ctx, connect.NewRequest(&relayv1.RelayCallRequest{
		NodeId: target.NodeID, Scenario: "vrooli", Command: "resource install",
		Args: []string{def.DependencySlug, "--json"}, TimeoutSeconds: 180, MaxResponseBytes: 512 * 1024,
	}))
	if err != nil {
		return capabilities.LifecycleActionResult{}, err
	}
	if response == nil || response.Msg == nil {
		return capabilities.LifecycleActionResult{}, fmt.Errorf("Bridge returned no install result")
	}
	message := strings.TrimSpace(string(response.Msg.Data))
	if message == "" {
		message = strings.TrimSpace(response.Msg.Reason)
	}
	result := capabilities.LifecycleActionResult{CapabilityID: capabilityID, ActionKind: capabilities.ActionKindOperatorCommand, OperationID: response.Msg.GetCorrelationId(), Status: response.Msg.Outcome.String(), Success: response.Msg.Outcome == relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED && response.Msg.ExitCode == 0, Message: message}
	if result.Message == "" {
		result.Message = "remote capability installation completed"
	}
	return result, nil
}

func resourceAcquisition(name string) (*binaryfetch.Acquisition, bool) {
	for _, root := range []string{os.Getenv("VROOLI_CLI_SOURCE_ROOT"), os.Getenv("VROOLI_RESOURCE_ROOT"), os.Getenv("VROOLI_ROOT"), "."} {
		if strings.TrimSpace(root) == "" {
			continue
		}
		for _, path := range []string{
			filepath.Join(root, "resource.json"),
			filepath.Join(root, name, "resource.json"),
			filepath.Join(root, "resources", name, "resource.json"),
		} {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var item struct {
				Name        string                   `json:"name"`
				Acquisition *binaryfetch.Acquisition `json:"acquisition"`
			}
			if json.Unmarshal(data, &item) == nil && item.Name == name && item.Acquisition != nil {
				return item.Acquisition, true
			}
		}
	}
	return nil, false
}

func capabilityDefinition(id string) (capabilities.Def, bool) {
	for _, def := range capabilities.Known {
		if def.ID == id {
			return def, true
		}
	}
	return capabilities.Def{}, false
}

func authenticatedHTTPClient(ownerToken, reauthToken string) *http.Client {
	return &http.Client{Transport: authTransport{base: http.DefaultTransport, owner: ownerToken, reauth: reauthToken}}
}

type authTransport struct {
	base          http.RoundTripper
	owner, reauth string
}

func (t authTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	if strings.TrimSpace(t.owner) != "" {
		clone.Header.Set("Authorization", t.owner)
	}
	if strings.TrimSpace(t.reauth) != "" {
		clone.Header.Set("X-Bridge-Owner-Reauth", t.reauth)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}
