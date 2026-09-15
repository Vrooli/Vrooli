package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

func (s *Server) hubClient() (commonv1connect.ConnectionServiceClient, error) {
	if s == nil || strings.TrimSpace(s.integrationHubURL) == "" {
		return nil, errors.New("integration hub unavailable")
	}
	return commonv1connect.NewConnectionServiceClient(http.DefaultClient, s.integrationHubURL), nil
}

func hubRequest[T any](ctx context.Context, request *connect.Request[T], source *http.Request) {
	if source == nil {
		return
	}
	for _, name := range []string{"Authorization", "X-Vrooli-Identity"} {
		if value := strings.TrimSpace(source.Header.Get(name)); value != "" {
			request.Header().Set(name, value)
		}
	}
}

func (s *Server) listHubConnections(ctx context.Context, source *http.Request) ([]*commonv1.Connection, error) {
	client, err := s.hubClient()
	if err != nil {
		return nil, err
	}
	request := connect.NewRequest(&commonv1.ListConnectionsRequest{ConnectorId: webConsoleOpenRouterConnector})
	hubRequest(ctx, request, source)
	response, err := client.ListConnections(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Msg.GetConnections(), nil
}

func (s *Server) probeHubOpenRouter(ctx context.Context, source *http.Request) (*commonv1.Connection, error) {
	client, err := s.hubClient()
	if err != nil {
		return nil, err
	}
	connections, err := s.listHubConnections(ctx, source)
	if err != nil {
		return nil, err
	}
	for _, connection := range connections {
		if connection.GetConnectorId() != webConsoleOpenRouterConnector {
			continue
		}
		request := connect.NewRequest(&commonv1.ConnectionMutationRequest{ConnectionId: connection.GetId(), RequestId: "web-console-openrouter-probe"})
		hubRequest(ctx, request, source)
		response, probeErr := client.ProbeConnection(ctx, request)
		if probeErr != nil {
			return nil, probeErr
		}
		return response.Msg.GetConnection(), nil
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.New("OpenRouter connection not found"))
}

func (s *Server) createHubConnection(ctx context.Context, source *http.Request, value string) error {
	client, err := s.hubClient()
	if err != nil {
		return err
	}
	request := connect.NewRequest(&commonv1.ConnectionMutationRequest{
		ConnectionId:    "web-console-openrouter",
		ConnectorId:     webConsoleOpenRouterConnector,
		DisplayName:     "OpenRouter API credential",
		CredentialValue: value,
		RequestId:       "web-console-openrouter-provision",
	})
	hubRequest(ctx, request, source)
	if _, err = client.CreateConnection(ctx, request); err == nil {
		return nil
	}
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		return err
	}
	// Provisioning the same declared Web Console credential is a rotation,
	// not an attempt to create a second connection with a fixed id.
	rotate := connect.NewRequest(&commonv1.ConnectionMutationRequest{
		ConnectionId:    request.Msg.GetConnectionId(),
		CredentialValue: value,
		RequestId:       "web-console-openrouter-rotate",
	})
	hubRequest(ctx, rotate, source)
	_, rotateErr := client.RotateConnection(ctx, rotate)
	if rotateErr == nil {
		return nil
	}
	return err
}

func (s *Server) deleteHubConnection(ctx context.Context, source *http.Request, id string) error {
	client, err := s.hubClient()
	if err != nil {
		return err
	}
	request := connect.NewRequest(&commonv1.ConnectionMutationRequest{ConnectionId: id, RequestId: "web-console-openrouter-delete"})
	hubRequest(ctx, request, source)
	_, err = client.DeleteConnection(ctx, request)
	return err
}
