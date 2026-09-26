package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

const (
	openRouterConnector = "openrouter"
	openRouterName      = "OpenRouter"
	credentialField     = "api-key"
	identityHeader      = "X-Vrooli-Identity"
)

var (
	errUnauthenticated = errors.New("authenticated identity is required")
	errNotFound        = errors.New("connection not found")
)

// CredentialStore is the narrow secret-authority seam. The hub can test its
// lifecycle without importing or persisting provider values, and production
// wiring uses credentialclient-go over the canonical credential authority.
type CredentialStore interface {
	Provision(context.Context, CredentialProvisionRequest) error
	Status(context.Context, string, string) (CredentialStatus, error)
	Delete(context.Context, string, string) error
}

type CredentialProvisionRequest struct {
	Identity string
	Field    string
	Value    string
}

type CredentialStatus struct {
	Configured    bool   `json:"configured"`
	ProviderState string `json:"provider_state"`
}

// ConnectionRecord is the durable metadata projection. Secret values are
// deliberately absent; CredentialAuthorityRef is only an opaque address.
type ConnectionRecord struct {
	ID                     string                        `json:"id"`
	Owner                  string                        `json:"owner"`
	ConnectorID            string                        `json:"connector_id"`
	DisplayName            string                        `json:"display_name"`
	AccountLabel           string                        `json:"account_label"`
	AccountIdentity        string                        `json:"account_identity"`
	Status                 commonv1.ConnectionStatus     `json:"status"`
	Scopes                 []*commonv1.ConnectionScope   `json:"scopes"`
	Bindings               []*commonv1.ConnectionBinding `json:"bindings"`
	LastVerifiedAt         string                        `json:"last_verified_at,omitempty"`
	ReasonCode             string                        `json:"reason_code,omitempty"`
	NextAction             string                        `json:"next_action,omitempty"`
	CredentialAuthorityRef string                        `json:"credential_authority_ref"`
}

type durableState struct {
	Connections map[string]ConnectionRecord `json:"connections"`
	Requests    map[string]string           `json:"requests"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data durableState
}

func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("connection state path is required")
	}
	s := &Store{path: path, data: durableState{Connections: map[string]ConnectionRecord{}, Requests: map[string]string{}}}
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read connection state: %w", err)
	}
	if err := json.Unmarshal(contents, &s.data); err != nil {
		return nil, fmt.Errorf("decode connection state: %w", err)
	}
	if s.data.Connections == nil {
		s.data.Connections = map[string]ConnectionRecord{}
	}
	if s.data.Requests == nil {
		s.data.Requests = map[string]string{}
	}
	return s, nil
}

func (s *Store) snapshot() durableState {
	return durableState{Connections: s.data.Connections, Requests: s.data.Requests}
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create connection state directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".connections-*.tmp")
	if err != nil {
		return fmt.Errorf("create connection state temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("protect connection state temporary file: %w", err)
	}
	encoded, err := json.MarshalIndent(s.snapshot(), "", "  ")
	if err != nil {
		tmp.Close()
		return fmt.Errorf("encode connection state: %w", err)
	}
	if _, err := tmp.Write(append(encoded, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write connection state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close connection state: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("commit connection state: %w", err)
	}
	return nil
}

type Hub struct {
	commonv1connect.UnimplementedConnectionServiceHandler
	store       *Store
	credentials CredentialStore
	now         func() time.Time
}

func NewHub(store *Store, credentials CredentialStore) *Hub {
	return &Hub{store: store, credentials: credentials, now: time.Now}
}

func (h *Hub) identity(req interface{ Header() http.Header }) (string, error) {
	identity := strings.TrimSpace(req.Header().Get(identityHeader))
	if identity == "" {
		auth := strings.TrimSpace(req.Header().Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token := strings.TrimSpace(auth[len("Bearer "):])
			digest := sha256.Sum256([]byte(token))
			identity = "bearer-" + hex.EncodeToString(digest[:])
		}
	}
	if identity == "" {
		return "", errUnauthenticated
	}
	return identity, nil
}

func accountIdentity(owner string) string {
	if strings.HasPrefix(owner, "bearer-") {
		return "authenticated-user"
	}
	return owner
}

func connector(id string) bool { return strings.TrimSpace(id) == openRouterConnector }

func authorityRef(id string) string { return "vrooli/integration-hub/" + id }

func (h *Hub) toProto(record ConnectionRecord) *commonv1.Connection {
	actions := []commonv1.ConnectionActionKind{
		commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_TEST,
		commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_REFRESH,
		commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_ROTATE,
		commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_BIND,
		commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_UNBIND,
		commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_REVOKE,
		commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_DELETE,
	}
	if record.Status == commonv1.ConnectionStatus_CONNECTION_STATUS_REVOKED || record.Status == commonv1.ConnectionStatus_CONNECTION_STATUS_DISCONNECTED {
		actions = []commonv1.ConnectionActionKind{commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_CONNECT, commonv1.ConnectionActionKind_CONNECTION_ACTION_KIND_DELETE}
	}
	return &commonv1.Connection{
		Id: record.ID, ConnectorId: record.ConnectorID, ConnectorName: openRouterName,
		DisplayName: record.DisplayName, AccountLabel: record.AccountLabel, AccountIdentity: accountIdentity(record.AccountIdentity),
		Status: record.Status, Scopes: record.Scopes, Bindings: record.Bindings,
		LastVerifiedAt: record.LastVerifiedAt, Freshness: freshness(record.LastVerifiedAt, h.now()),
		ReasonCode: record.ReasonCode, NextAction: record.NextAction, SupportedActions: actions,
		CredentialAuthorityRef: record.CredentialAuthorityRef,
	}
}

func freshness(verified string, now time.Time) string {
	if verified == "" {
		return "never"
	}
	t, err := time.Parse(time.RFC3339, verified)
	if err != nil || now.Sub(t) > 24*time.Hour {
		return "stale"
	}
	return "fresh"
}

func (h *Hub) authorizedRecord(identity, id string) (ConnectionRecord, error) {
	record, ok := h.store.data.Connections[id]
	if !ok || record.Owner != identity {
		return ConnectionRecord{}, errNotFound
	}
	return record, nil
}

func requestID(req *commonv1.ConnectionMutationRequest) string {
	return strings.TrimSpace(req.GetRequestId())
}

func (h *Hub) idempotentLocked(req *commonv1.ConnectionMutationRequest, identity string) (*connect.Response[commonv1.ConnectionMutationResponse], bool) {
	id := requestID(req)
	if id == "" {
		return nil, false
	}
	connectionID, ok := h.store.data.Requests[identity+":"+id]
	if !ok {
		return nil, false
	}
	record, ok := h.store.data.Connections[connectionID]
	if !ok {
		// A successful delete removes the metadata, but its request key remains
		// as a replay tombstone so a client retry cannot repeat the authority
		// mutation or turn a successful delete into a misleading 404.
		return connect.NewResponse(&commonv1.ConnectionMutationResponse{
			Connection: &commonv1.Connection{Id: connectionID}, RequestId: id,
		}), true
	}
	if record.Owner != identity {
		return nil, false
	}
	return connect.NewResponse(&commonv1.ConnectionMutationResponse{Connection: h.toProto(record), RequestId: id}), true
}

func (h *Hub) rememberLocked(req *commonv1.ConnectionMutationRequest, identity, connectionID string) {
	if id := requestID(req); id != "" {
		h.store.data.Requests[identity+":"+id] = connectionID
	}
}

func mutationResponse(record ConnectionRecord, request string) *connect.Response[commonv1.ConnectionMutationResponse] {
	return connect.NewResponse(&commonv1.ConnectionMutationResponse{Connection: &commonv1.Connection{Id: record.ID}, RequestId: request})
}

func (h *Hub) ListConnections(ctx context.Context, req *connect.Request[commonv1.ListConnectionsRequest]) (*connect.Response[commonv1.ListConnectionsResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	response := &commonv1.ListConnectionsResponse{Connections: []*commonv1.Connection{}, GeneratedAt: h.now().UTC().Format(time.RFC3339)}
	for _, record := range h.store.data.Connections {
		if record.Owner == identity && (req.Msg.GetConnectorId() == "" || record.ConnectorID == req.Msg.GetConnectorId()) {
			response.Connections = append(response.Connections, h.toProto(record))
		}
	}
	return connect.NewResponse(response), nil
}

func (h *Hub) GetConnection(ctx context.Context, req *connect.Request[commonv1.GetConnectionRequest]) (*connect.Response[commonv1.GetConnectionResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	record, err := h.authorizedRecord(identity, req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&commonv1.GetConnectionResponse{Connection: h.toProto(record)}), nil
}

func (h *Hub) CreateConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if !connector(req.Msg.GetConnectorId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported connector"))
	}
	if strings.TrimSpace(req.Msg.GetCredentialValue()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("write-only credential value is required"))
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if response, ok := h.idempotentLocked(req.Msg, identity); ok {
		return response, nil
	}
	id := strings.TrimSpace(req.Msg.GetConnectionId())
	if id == "" {
		id = fmt.Sprintf("%s-%d", openRouterConnector, h.now().UnixNano())
	}
	if _, exists := h.store.data.Connections[id]; exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("connection id already exists"))
	}
	ref := authorityRef(id)
	if err := h.credentials.Provision(ctx, CredentialProvisionRequest{Identity: ref, Field: credentialField, Value: req.Msg.GetCredentialValue()}); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential authority unavailable"))
	}
	record := ConnectionRecord{ID: id, Owner: identity, ConnectorID: openRouterConnector, DisplayName: strings.TrimSpace(req.Msg.GetDisplayName()), AccountLabel: openRouterName, AccountIdentity: identity, Status: commonv1.ConnectionStatus_CONNECTION_STATUS_CONNECTED, Scopes: []*commonv1.ConnectionScope{{Name: "models", Purpose: "Use configured OpenRouter models", Granted: true}}, Bindings: []*commonv1.ConnectionBinding{}, LastVerifiedAt: h.now().UTC().Format(time.RFC3339), NextAction: "Test connection", CredentialAuthorityRef: ref}
	if record.DisplayName == "" {
		record.DisplayName = "OpenRouter connection"
	}
	h.store.data.Connections[id] = record
	h.rememberLocked(req.Msg, identity, id)
	if err := h.store.saveLocked(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist connection metadata"))
	}
	return connect.NewResponse(&commonv1.ConnectionMutationResponse{Connection: h.toProto(record), RequestId: requestID(req.Msg)}), nil
}

func (h *Hub) ProbeConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	return h.checkConnection(ctx, req, false)
}

func (h *Hub) RefreshConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	return h.checkConnection(ctx, req, true)
}

func (h *Hub) RotateConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	secret := strings.TrimSpace(req.Msg.GetCredentialValue())
	if secret == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("write-only credential value is required"))
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if response, ok := h.idempotentLocked(req.Msg, identity); ok {
		return response, nil
	}
	record, err := h.authorizedRecord(identity, req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := h.credentials.Provision(ctx, CredentialProvisionRequest{Identity: record.CredentialAuthorityRef, Field: credentialField, Value: secret}); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential authority unavailable"))
	}
	record.Status = commonv1.ConnectionStatus_CONNECTION_STATUS_CONNECTED
	record.ReasonCode = ""
	record.NextAction = "Test connection"
	record.LastVerifiedAt = h.now().UTC().Format(time.RFC3339)
	h.store.data.Connections[record.ID] = record
	h.rememberLocked(req.Msg, identity, record.ID)
	if err := h.store.saveLocked(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist connection metadata"))
	}
	return connect.NewResponse(&commonv1.ConnectionMutationResponse{Connection: h.toProto(record), RequestId: requestID(req.Msg)}), nil
}

func (h *Hub) checkConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest], refresh bool) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if response, ok := h.idempotentLocked(req.Msg, identity); ok {
		return response, nil
	}
	record, err := h.authorizedRecord(identity, req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	status, statusErr := h.credentials.Status(ctx, record.CredentialAuthorityRef, credentialField)
	if statusErr != nil || status.ProviderState == "unavailable" || status.ProviderState == "absent" {
		record.Status = commonv1.ConnectionStatus_CONNECTION_STATUS_PROVIDER_UNAVAILABLE
		record.ReasonCode = "credential_authority_unavailable"
		record.NextAction = "Retry connection check"
	} else if !status.Configured {
		record.Status = commonv1.ConnectionStatus_CONNECTION_STATUS_DISCONNECTED
		record.ReasonCode = "credential_missing"
		record.NextAction = "Reconnect"
	} else {
		record.Status = commonv1.ConnectionStatus_CONNECTION_STATUS_CONNECTED
		record.ReasonCode = ""
		record.NextAction = "Test connection"
		if refresh {
			record.LastVerifiedAt = h.now().UTC().Format(time.RFC3339)
		}
	}
	h.store.data.Connections[record.ID] = record
	if requestID(req.Msg) != "" {
		h.rememberLocked(req.Msg, identity, record.ID)
	}
	if err := h.store.saveLocked(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist connection metadata"))
	}
	return connect.NewResponse(&commonv1.ConnectionMutationResponse{Connection: h.toProto(record), RequestId: requestID(req.Msg)}), nil
}

func (h *Hub) BindConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	return h.bind(ctx, req, true)
}
func (h *Hub) UnbindConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	return h.bind(ctx, req, false)
}

func (h *Hub) bind(reqCtx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest], add bool) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	slug := strings.Trim(strings.TrimSpace(req.Msg.GetBindingScenarioSlug()), "/")
	if slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("binding scenario slug is required"))
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if response, ok := h.idempotentLocked(req.Msg, identity); ok {
		return response, nil
	}
	record, err := h.authorizedRecord(identity, req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if add {
		granted := make(map[string]bool, len(record.Scopes))
		for _, scope := range record.Scopes {
			if scope.GetGranted() {
				granted[scope.GetName()] = true
			}
		}
		for _, required := range req.Msg.GetRequiredScopes() {
			if !granted[strings.TrimSpace(required)] {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("connection lacks required scope"))
			}
		}
	}
	binding := &commonv1.ConnectionBinding{ScenarioSlug: slug, ScenarioName: slug, Context: strings.TrimSpace(req.Msg.GetBindingContext())}
	found := false
	filtered := make([]*commonv1.ConnectionBinding, 0, len(record.Bindings)+1)
	for _, existing := range record.Bindings {
		if existing.GetScenarioSlug() == slug && existing.GetContext() == binding.GetContext() {
			found = true
			if add {
				filtered = append(filtered, existing)
			}
			continue
		}
		filtered = append(filtered, existing)
	}
	if add && !found {
		filtered = append(filtered, binding)
	}
	record.Bindings = filtered
	if err := h.store.saveLocked(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist binding metadata"))
	}
	h.rememberLocked(req.Msg, identity, record.ID)
	if err := h.store.saveLocked(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist binding request"))
	}
	return connect.NewResponse(&commonv1.ConnectionMutationResponse{Connection: h.toProto(record), RequestId: requestID(req.Msg)}), nil
}

func (h *Hub) RevokeConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if response, ok := h.idempotentLocked(req.Msg, identity); ok {
		return response, nil
	}
	record, err := h.authorizedRecord(identity, req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := h.credentials.Delete(ctx, record.CredentialAuthorityRef, credentialField); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential authority unavailable"))
	}
	record.Status = commonv1.ConnectionStatus_CONNECTION_STATUS_REVOKED
	record.ReasonCode = "revoked_by_user"
	record.NextAction = "Reconnect or delete"
	h.store.data.Connections[record.ID] = record
	h.rememberLocked(req.Msg, identity, record.ID)
	if err := h.store.saveLocked(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist connection metadata"))
	}
	return connect.NewResponse(&commonv1.ConnectionMutationResponse{Connection: h.toProto(record), RequestId: requestID(req.Msg)}), nil
}

func (h *Hub) DeleteConnection(ctx context.Context, req *connect.Request[commonv1.ConnectionMutationRequest]) (*connect.Response[commonv1.ConnectionMutationResponse], error) {
	identity, err := h.identity(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if response, ok := h.idempotentLocked(req.Msg, identity); ok {
		return response, nil
	}
	record, err := h.authorizedRecord(identity, req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := h.credentials.Delete(ctx, record.CredentialAuthorityRef, credentialField); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential authority unavailable"))
	}
	h.rememberLocked(req.Msg, identity, record.ID)
	delete(h.store.data.Connections, record.ID)
	if err := h.store.saveLocked(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist connection metadata"))
	}
	return mutationResponse(record, requestID(req.Msg)), nil
}
