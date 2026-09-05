package channels

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/owneridentity"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/switchboard/v1/channels"
	channelconnect "github.com/vrooli/vrooli/packages/proto/gen/go/switchboard/v1/channels/channels_v1connect"

	agents "switchboard/internal/agents"
	channelcore "switchboard/internal/channels"
	"switchboard/internal/contacts"
	"switchboard/internal/dispatch"
	"switchboard/internal/egress"
	"switchboard/internal/gates"
	"switchboard/internal/identity"
	"switchboard/internal/module"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
)

type ModuleDeps struct {
	Registry  *channelcore.Registry
	Facts     channelcore.HostFacts
	DB        *sql.DB
	Egress    *egress.Router
	Processor *dispatch.Processor
	Gates     *gates.Store
	Identity  owneridentity.Validator
	Contacts  *contacts.Store
	Threads   *threads.Store
}

type service struct{ deps ModuleDeps }

// Start launches receive loops for configured external adapters. The HTTP
// module binds the in-app adapter separately, so the registry excludes it.
func Start(ctx context.Context, d ModuleDeps, logf func(string, ...any)) func() {
	if d.Registry == nil {
		return func() {}
	}
	svc := &service{deps: d}
	return d.Registry.Start(ctx, d.Facts, func(envelope channelcore.Envelope) error {
		_, err := svc.ingest(ctx, envelope)
		return err
	}, logf)
}

type gateRequest struct {
	ThreadID   string `json:"thread_id"`
	OwnerID    string `json:"owner_id"`
	Scope      string `json:"scope"`
	Withheld   string `json:"withheld"`
	Unblock    string `json:"unblock"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

type gateAnswer struct {
	ActorID string `json:"actor_id"`
	Grant   bool   `json:"grant"`
}

var errUnbound = errors.New("sender is not bound to an agent")

func (s *service) ListChannels(ctx context.Context, _ *connect.Request[channelv1.ListChannelsRequest]) (*connect.Response[channelv1.ListChannelsResponse], error) {
	list := s.deps.Registry.List(ctx, s.deps.Facts)
	out := &channelv1.ListChannelsResponse{}
	for _, item := range list {
		out.Channels = append(out.Channels, &channelv1.Channel{Id: item.Descriptor.ID, DisplayName: item.Descriptor.DisplayName, Availability: string(item.Availability), Reason: item.Reason, Friction: int32(item.Descriptor.Setup.Friction)})
	}
	return connect.NewResponse(out), nil
}

func (s *service) GetChannel(ctx context.Context, req *connect.Request[channelv1.GetChannelRequest]) (*connect.Response[channelv1.GetChannelResponse], error) {
	d, ok := s.deps.Registry.Get(req.Msg.Id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("channel %q not found", req.Msg.Id))
	}
	state, reason := channelcore.Evaluate(d, s.deps.Facts)
	return connect.NewResponse(&channelv1.GetChannelResponse{Channel: &channelv1.Channel{Id: d.ID, DisplayName: d.DisplayName, Availability: string(state), Reason: reason, Friction: int32(d.Setup.Friction)}}), nil
}

func (s *service) ListBindings(ctx context.Context, req *connect.Request[channelv1.ListBindingsRequest]) (*connect.Response[channelv1.ListBindingsResponse], error) {
	if s.deps.DB == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("binding storage unavailable"))
	}
	records, err := agents.NewStore(s.deps.DB).List(ctx, req.Msg.AgentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := connect.NewResponse(&channelv1.ListBindingsResponse{})
	for _, record := range records {
		out.Msg.Bindings = append(out.Msg.Bindings, &channelv1.Binding{Id: record.ID, AgentId: record.AgentID, ChannelId: record.ChannelID, Address: record.Address, ThreadKey: record.ThreadKey})
	}
	return out, nil
}

func (s *service) CreateBinding(ctx context.Context, req *connect.Request[channelv1.CreateBindingRequest]) (*connect.Response[channelv1.CreateBindingResponse], error) {
	if s.deps.DB == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("binding storage unavailable"))
	}
	if _, err := s.authenticatedSubjectFromHeaders(ctx, req.Header()); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	record, err := agents.NewStore(s.deps.DB).Create(ctx, agents.Binding{AgentID: req.Msg.AgentId, ChannelID: req.Msg.ChannelId, Address: req.Msg.Address, ThreadKey: req.Msg.ThreadKey})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&channelv1.CreateBindingResponse{Binding: &channelv1.Binding{Id: record.ID, AgentId: record.AgentID, ChannelId: record.ChannelID, Address: record.Address, ThreadKey: record.ThreadKey}}), nil
}

func (s *service) send(w http.ResponseWriter, req *http.Request) {
	if s.deps.Egress == nil {
		http.Error(w, "channel egress unavailable", http.StatusServiceUnavailable)
		return
	}
	var out channelcore.Outbound
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 8<<20)).Decode(&out); err != nil {
		http.Error(w, "invalid outbound message", http.StatusBadRequest)
		return
	}
	if err := s.deps.Egress.Send(req.Context(), out); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *service) receive(w http.ResponseWriter, req *http.Request) {
	if s.deps.Processor == nil || s.deps.DB == nil {
		http.Error(w, "inbound processing unavailable", http.StatusServiceUnavailable)
		return
	}
	var envelope channelcore.Envelope
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 8<<20)).Decode(&envelope); err != nil {
		http.Error(w, "invalid inbound envelope", http.StatusBadRequest)
		return
	}
	result, err := s.ingest(dispatch.WithAuthorization(req.Context(), req.Header.Get("Authorization")), envelope)
	if err != nil {
		if errors.Is(err, errUnbound) {
			http.Error(w, errUnbound.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *service) raiseGate(w http.ResponseWriter, req *http.Request) {
	if s.deps.Gates == nil {
		http.Error(w, "gate storage unavailable", http.StatusServiceUnavailable)
		return
	}
	var input gateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, "invalid gate request", http.StatusBadRequest)
		return
	}
	ownerID, err := s.authenticatedSubject(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if input.OwnerID != "" && input.OwnerID != ownerID {
		http.Error(w, "owner_id does not match authenticated owner", http.StatusForbidden)
		return
	}
	g, err := s.deps.Gates.Raise(req.Context(), input.ThreadID, ownerID, input.Scope, input.Withheld, input.Unblock, time.Duration(input.TTLSeconds)*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGate(w, g)
}

func (s *service) getGate(w http.ResponseWriter, req *http.Request) {
	if s.deps.Gates == nil {
		http.Error(w, "gate storage unavailable", http.StatusServiceUnavailable)
		return
	}
	g, ok, err := s.deps.Gates.Get(req.Context(), mux.Vars(req)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, req)
		return
	}
	writeGate(w, g)
}

func (s *service) answerGate(w http.ResponseWriter, req *http.Request) {
	if s.deps.Gates == nil {
		http.Error(w, "gate storage unavailable", http.StatusServiceUnavailable)
		return
	}
	var input gateAnswer
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, "invalid gate answer", http.StatusBadRequest)
		return
	}
	actorID, err := s.authenticatedSubject(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if input.ActorID != "" && input.ActorID != actorID {
		http.Error(w, "actor_id does not match authenticated owner", http.StatusForbidden)
		return
	}
	g, err := s.deps.Gates.Answer(req.Context(), mux.Vars(req)["id"], actorID, input.Grant)
	if errors.Is(err, gates.ErrNotOwner) {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if errors.Is(err, gates.ErrNotPending) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeGate(w, g)
}

func (s *service) authenticatedSubject(req *http.Request) (string, error) {
	return s.authenticatedSubjectFromHeaders(req.Context(), req.Header)
}

func (s *service) authenticatedSubjectFromHeaders(ctx context.Context, headers http.Header) (string, error) {
	return identity.Subject(ctx, headers, s.deps.Identity)
}

func writeGate(w http.ResponseWriter, g gates.Gate) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(g)
}

// ingest is the single inbound path for every adapter. It resolves the
// binding, records the sender as a contact at the tier the channel descriptor
// declares for a first appearance, joins the thread roster, and hands the
// REAL sender tier and room ceiling to the processor. Nothing here reads a
// channel identifier: the default tier is descriptor data.
func (s *service) ingest(ctx context.Context, envelope channelcore.Envelope) (dispatch.Result, error) {
	records, err := agents.NewStore(s.deps.DB).List(ctx, "")
	if err != nil {
		return dispatch.Result{}, err
	}
	bindings := make([]agents.Binding, 0, len(records))
	for _, record := range records {
		bindings = append(bindings, record.Binding)
	}
	binding, err := agents.Resolve(bindings, envelope.ChannelID, envelope.ThreadKey, envelope.SenderAddress)
	if err != nil {
		// A group room is bound by its thread key alone: any human in the room
		// may address the agent, and the trust guard decides what they get.
		binding, err = agents.Resolve(bindings, envelope.ChannelID, envelope.ThreadKey, "")
		if err != nil {
			return dispatch.Result{}, errUnbound
		}
	}
	sender, ceiling := trust.Stranger, trust.Stranger
	if envelope.AuthorKind != channelcore.AuthorAgent && s.deps.Contacts != nil && s.deps.Threads != nil {
		defaultTier := "stranger"
		if d, ok := s.deps.Registry.Get(envelope.ChannelID); ok {
			defaultTier = d.DefaultTier()
		}
		contact, err := s.deps.Contacts.Seen(ctx, envelope.ChannelID, envelope.SenderAddress, defaultTier)
		if err != nil {
			return dispatch.Result{}, err
		}
		thread, err := s.deps.Threads.Upsert(ctx, envelope, envelope.Group)
		if err != nil {
			return dispatch.Result{}, err
		}
		if err := s.deps.Contacts.Join(ctx, thread.ID, contact.ID); err != nil {
			return dispatch.Result{}, err
		}
		if sender, err = trust.ParseTier(contact.Tier); err != nil {
			return dispatch.Result{}, err
		}
		if ceiling, err = s.deps.Contacts.Ceiling(ctx, thread.ID); err != nil {
			return dispatch.Result{}, err
		}
		if !thread.IsGroup && !envelope.Group {
			ceiling = sender
		}
	}
	result, err := s.deps.Processor.Process(ctx, envelope, sender, ceiling, binding.AgentID, envelope.Group, len(envelope.Mentions) > 0 || envelope.ReplyToRemoteID != "")
	if err != nil {
		return dispatch.Result{}, err
	}
	return result, nil
}

// startThread is injected into every adapter that can open a conversation on
// demand. It creates the binding and the durable thread in one step.
func (s *service) startThread(ctx context.Context, agentID string, started channelcore.Started) (string, error) {
	if s.deps.DB == nil || s.deps.Threads == nil {
		return "", fmt.Errorf("binding storage unavailable")
	}
	if _, err := agents.NewStore(s.deps.DB).Create(ctx, agents.Binding{AgentID: agentID, ChannelID: started.ChannelID, Address: started.Address, ThreadKey: started.ThreadKey}); err != nil {
		return "", err
	}
	thread, err := s.deps.Threads.Upsert(ctx, channelcore.Envelope{ChannelID: started.ChannelID, ThreadKey: started.ThreadKey}, false)
	if err != nil {
		return "", err
	}
	return thread.ID, nil
}

func (s *service) SendMessage(ctx context.Context, req *connect.Request[channelv1.SendMessageRequest]) (*connect.Response[channelv1.SendMessageResponse], error) {
	if s.deps.Egress == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("channel egress unavailable"))
	}
	media := make([]channelcore.Media, 0, len(req.Msg.Media))
	for _, item := range req.Msg.Media {
		media = append(media, channelcore.Media{Name: item.Name, MIME: item.Mime, URL: item.Url, Size: item.Size})
	}
	err := s.deps.Egress.Send(ctx, channelcore.Outbound{ChannelID: req.Msg.ChannelId, ThreadKey: req.Msg.ThreadKey, Text: req.Msg.Text, ReplyToRemoteID: req.Msg.ReplyToRemoteId, Media: media})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&channelv1.SendMessageResponse{Accepted: true}), nil
}

func Module(d ModuleDeps) module.Module {
	return module.Module{Name: "channels", Mount: func(r *mux.Router) {
		svc := &service{deps: d}
		for _, adapter := range d.Registry.HTTPAdapters() {
			adapter.BindReceive(func(e channelcore.Envelope) error { _, err := svc.ingest(context.Background(), e); return err })
			r.Handle(adapter.HTTPPath(), adapter.Handler()).Methods(http.MethodGet)
		}
		for _, adapter := range d.Registry.ThreadStarters() {
			adapter.BindStart(svc.startThread)
			r.Handle(adapter.StartPath(), adapter.StartHandler()).Methods(http.MethodPost)
		}
		path, handler := channelconnect.NewChannelServiceHandler(&service{deps: d})
		r.PathPrefix(path).Handler(handler)
		r.HandleFunc("/api/v1/channels", func(w http.ResponseWriter, req *http.Request) {
			if d.Registry == nil {
				http.Error(w, "channel registry unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(d.Registry.List(req.Context(), d.Facts))
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/channels/send", (&service{deps: d}).send).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channels/receive", (&service{deps: d}).receive).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/gates", (&service{deps: d}).raiseGate).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/gates/{id}", (&service{deps: d}).getGate).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/gates/{id}/answer", (&service{deps: d}).answerGate).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "channels_list", Path: "/api/v1/channels", Method: http.MethodGet, Summary: "List channel availability", Description: "Lists descriptor-backed channels ordered by setup friction.", Category: "channels", RESTException: jsonRESTException(false)},
	{ID: "channels_start_thread", Path: "/api/v1/channels/in-app/threads", Method: http.MethodPost, Summary: "Start an in-app conversation", Description: "Opens a new in-app thread with one agent: creates the binding and the durable thread, and returns the thread key the console connects with.", Category: "channels", RESTException: jsonRESTException(true)},
	{ID: "channels_socket", Path: "/api/v1/channels/socket", Method: http.MethodGet, Summary: "Connect an in-app conversation", Description: "Connects an in-app conversation over the registered WebSocket adapter.", Category: "channels", RESTException: websocketRESTException()},
	{ID: "channels_rpc_list", Path: channelconnect.ChannelServiceListChannelsProcedure, Method: http.MethodPost, Summary: "List channels", Category: "channels"},
	{ID: "channels_rpc_get", Path: channelconnect.ChannelServiceGetChannelProcedure, Method: http.MethodPost, Summary: "Get channel", Category: "channels"},
	{ID: "channels_rpc_bindings", Path: channelconnect.ChannelServiceListBindingsProcedure, Method: http.MethodPost, Summary: "List bindings", Category: "channels"},
	{ID: "channels_rpc_create_binding", Path: channelconnect.ChannelServiceCreateBindingProcedure, Method: http.MethodPost, Summary: "Create binding", Category: "channels"},
	{ID: "channels_rpc_send_message", Path: channelconnect.ChannelServiceSendMessageProcedure, Method: http.MethodPost, Summary: "Send a message through a registered channel adapter", Category: "channels"},
	{ID: "gates_create", Path: "/api/v1/gates", Method: http.MethodPost, Summary: "Raise a capability gate", Description: "Creates an owner-only, expiring capability approval request.", Category: "gates", RESTException: jsonRESTException(true)},
	{ID: "gates_get", Path: "/api/v1/gates/{id}", Method: http.MethodGet, Summary: "Read a capability gate", Category: "gates", RESTException: jsonRESTException(false)},
	{ID: "gates_answer", Path: "/api/v1/gates/{id}/answer", Method: http.MethodPost, Summary: "Answer a capability gate", Description: "Answers a pending gate with owner authorization enforced by the gate store.", Category: "gates", RESTException: jsonRESTException(true)},
}

func jsonRESTException(hasRequest bool) *module.RESTException {
	request := module.RESTPayload{Transport: "none", Conformance: "none"}
	if hasRequest {
		request = module.RESTPayload{Transport: "json", Conformance: "external_shape"}
	}
	return &module.RESTException{Reason: module.RESTReasonOpsProbe, ProtoPayloads: &module.RESTProtoPayloads{
		Request: request, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
	}}
}

func websocketRESTException() *module.RESTException {
	return &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Bidirectional JSON WebSocket transport; messages use the normalized channel envelope.", ProtoPayloads: &module.RESTProtoPayloads{
		Request: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
	}}
}
