package relay

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/dispatch"

	"github.com/google/uuid"
)

type Service interface {
	Call(ctx context.Context, request Request) (Response, error)
	Reconcile(ctx context.Context, request ReconcileRequest) (CommandRecord, error)
}

type service struct {
	nodes      NodeReader
	presence   Presence
	audit      audit.Sink
	pusher     Pusher
	broker     *Broker
	commands   CommandStore
	manifest   []string
	catalogErr error
}

type Option func(*service)

func WithCommandStore(store CommandStore) Option {
	return func(s *service) {
		if store != nil {
			s.commands = store
		}
	}
}

func WithManifest(manifest []string) Option {
	return func(s *service) {
		s.manifest = append([]string(nil), manifest...)
		s.catalogErr = nil
	}
}

// WithCatalogError forces relay admission into the same typed degraded state
// as dispatch. It is useful for startup wiring and negative-path tests.
func WithCatalogError(err error) Option {
	return func(s *service) {
		if err != nil {
			s.catalogErr = dispatch.ErrCatalogUnavailable{Cause: err}
		}
		s.manifest = nil
	}
}

func NewService(nodes NodeReader, presence Presence, sink audit.Sink, pusher Pusher, broker *Broker, opts ...Option) Service {
	if broker == nil {
		broker = NewBroker()
	}
	manifest, _, catalogErr := dispatch.BuildManifest()
	s := &service{nodes: nodes, presence: presence, audit: sink, pusher: pusher, broker: broker, commands: NewMemoryCommandStore(), manifest: manifest}
	if catalogErr != nil {
		s.catalogErr = dispatch.ErrCatalogUnavailable{Cause: catalogErr}
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

var _ Service = (*service)(nil)

func (s *service) Call(ctx context.Context, request Request) (Response, error) {
	request = normalizeRequest(request)
	if s.catalogErr != nil {
		s.auditTerminal(ctx, request, audit.OutcomeRejected, s.catalogErr.Error())
		return Response{}, s.catalogErr
	}
	if request.NodeID == "" || request.Scenario == "" || request.Command == "" {
		return Response{}, fmt.Errorf("%w: node_id, scenario, and command are required", ErrInvalidRequest)
	}
	if request.MaxResponseBytes == 0 {
		request.MaxResponseBytes = DefaultMaxResponseBytes
	}
	if request.MaxResponseBytes > MaxResponseBytes {
		request.MaxResponseBytes = MaxResponseBytes
	}
	if request.CorrelationID == "" {
		request.CorrelationID = uuid.NewString()
	}
	if request.CommandID == "" {
		request.CommandID = uuid.NewString()
	}

	node, err := s.nodes.GetTarget(ctx, request.NodeID)
	if err != nil {
		return Response{}, err
	}
	if err := Admit(request, node, s.manifest); err != nil {
		s.auditTerminal(ctx, request, audit.OutcomeRejected, err.Error())
		return Response{}, err
	}
	if !s.presence.IsOnline(request.NodeID) {
		err := dispatch.ErrNodeOffline{ID: request.NodeID}
		s.auditTerminal(ctx, request, audit.OutcomeRejected, err.Error())
		return Response{}, err
	}
	if !s.presence.Dispatchable(request.NodeID) {
		err := dispatch.ErrNodeNeedsUpdate{ID: request.NodeID}
		s.auditTerminal(ctx, request, audit.OutcomeRejected, err.Error())
		return Response{}, err
	}

	record, err := s.commands.Reserve(ctx, request)
	if err != nil {
		return Response{}, err
	}
	if record.State == StateCompleted || record.State == StateFailed || record.State == StateOutcomeUnknown {
		return responseForRecord(record), nil
	}
	if record.State == StateReserved || record.State == StateSubmitted {
		// A reserved/submitted command may have crossed the node boundary. A
		// reconnecting caller must ask Reconcile instead of issuing a second
		// side effect.
		if record.Request.CorrelationID != request.CorrelationID && record.State == StateSubmitted {
			return Response{CorrelationID: record.Request.CorrelationID, Kind: KindOutcomeUnknown, Reason: "command is submitted; reconcile the original receipt"}, ErrCommandInFlight
		}
		if record.State == StateSubmitted {
			return Response{CorrelationID: record.Request.CorrelationID, Kind: KindOutcomeUnknown, Reason: "command is submitted; reconcile the original receipt"}, ErrCommandInFlight
		}
	}

	responses, unregister, err := s.broker.Register(request.CorrelationID, request.NodeID)
	if err != nil {
		return Response{}, err
	}
	defer unregister()
	if err := s.auditAccepted(ctx, request); err != nil {
		return Response{}, err
	}
	routeName, routeCost := "relay", uint64(1)
	if describer, ok := s.pusher.(RouteDescriber); ok {
		if name, cost := describer.Route(ctx, request.NodeID, request); strings.TrimSpace(name) != "" {
			routeName = strings.TrimSpace(name)
			if cost > 0 {
				routeCost = cost
			}
		}
	}
	// Persist the boundary before writing to the node. A Bridge crash after the
	// frame leaves a submitted record that forces reconciliation; a reported
	// zero-delivery result below is the explicit proof that it never crossed.
	if err := s.commands.Update(ctx, request.CommandID, request.NodeID, StateSubmitted, Response{CorrelationID: request.CorrelationID, Kind: KindAccepted, Route: routeName, RouteCostUnits: routeCost}); err != nil {
		return Response{CorrelationID: request.CorrelationID, Kind: KindOutcomeUnknown, Reason: "command admission could not be journaled"}, err
	}
	routeStarted := time.Now()
	delivered, err := s.pusher.Push(ctx, request.NodeID, request)
	if err != nil || delivered == 0 {
		_ = s.commands.Update(context.Background(), request.CommandID, request.NodeID, StateNotAdmitted, Response{CorrelationID: request.CorrelationID, Kind: KindFailed, Reason: "command was not admitted by the node", Route: routeName, RouteCostUnits: routeCost, RouteLatencyMS: elapsedMillis(routeStarted)})
		if err == nil {
			err = dispatch.ErrDeliveryFailed{NodeID: request.NodeID}
		}
		s.auditTerminal(ctx, request, audit.OutcomeFailed, "relay delivery failed: "+err.Error())
		return Response{CorrelationID: request.CorrelationID, Kind: KindFailed, Route: routeName, RouteCostUnits: routeCost, RouteLatencyMS: elapsedMillis(routeStarted)}, err
	}

	callCtx := ctx
	if request.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, time.Duration(request.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	var data []byte
	for {
		select {
		case <-callCtx.Done():
			reason := callCtx.Err().Error()
			_, _ = s.pusher.Cancel(context.Background(), request.NodeID, request.CorrelationID, reason)
			unknown := Response{CorrelationID: request.CorrelationID, Kind: KindOutcomeUnknown, Reason: reason, TotalBytes: uint64(len(data)), Data: data, Route: routeName, RouteCostUnits: routeCost, RouteLatencyMS: elapsedMillis(routeStarted)}
			_ = s.commands.Update(context.Background(), request.CommandID, request.NodeID, StateOutcomeUnknown, unknown)
			s.auditTerminal(context.Background(), request, audit.OutcomeFailed, KindOutcomeUnknown+": "+reason)
			return unknown, callCtx.Err()
		case response := <-responses:
			if response.Kind == KindData {
				if uint64(len(data))+uint64(len(response.Data)) > request.MaxResponseBytes {
					reason := (ErrResponseLimit{Limit: request.MaxResponseBytes}).Error()
					_, _ = s.pusher.Cancel(context.Background(), request.NodeID, request.CorrelationID, reason)
					s.auditTerminal(context.Background(), request, audit.OutcomeFailed, reason)
					return Response{CorrelationID: request.CorrelationID, Kind: KindFailed, Reason: reason, TotalBytes: uint64(len(data)), Route: routeName, RouteCostUnits: routeCost, RouteLatencyMS: elapsedMillis(routeStarted)}, ErrResponseLimit{Limit: request.MaxResponseBytes}
				}
				data = append(data, response.Data...)
				continue
			}
			if response.Kind == KindAccepted {
				continue
			}
			response.Data = data
			if response.TotalBytes == 0 {
				response.TotalBytes = uint64(len(data))
			}
			response.Route, response.RouteCostUnits, response.RouteLatencyMS = routeName, routeCost, elapsedMillis(routeStarted)
			_ = s.commands.Update(context.Background(), request.CommandID, request.NodeID, stateFromResponse(response), response)
			outcome := audit.OutcomeFailed
			if response.Kind == KindCompleted {
				outcome = audit.OutcomeCompleted
			}
			s.auditTerminal(context.Background(), request, outcome, response.Reason)
			return response, nil
		}
	}
}

func elapsedMillis(start time.Time) uint64 {
	if start.IsZero() {
		return 0
	}
	elapsed := time.Since(start).Milliseconds()
	if elapsed < 1 {
		return 1
	}
	return uint64(elapsed)
}

func (s *service) Reconcile(ctx context.Context, request ReconcileRequest) (CommandRecord, error) {
	request.NodeID = strings.TrimSpace(request.NodeID)
	request.CommandID = normalizeCommandID(request.CommandID)
	if request.NodeID == "" || request.CommandID == "" {
		return CommandRecord{}, ErrInvalidRequest
	}
	return s.commands.Get(ctx, request.CommandID, request.NodeID)
}

// Admit is the relay-facing projection of dispatch.Admit. It exists so parity
// tests can drive the two public admission paths with the same typed input;
// there is no second allowlist or scope matcher here.
func Admit(request Request, node dispatch.TargetNode, manifest []string) error {
	return dispatch.Admit(dispatch.Job{
		NodeID: request.NodeID, Scenario: request.Scenario, Verb: request.Command,
		Args: request.Args, TimeoutSeconds: request.TimeoutSeconds,
	}, node, manifest)
}

func normalizeRequest(in Request) Request {
	in.CommandID = strings.TrimSpace(in.CommandID)
	in.CorrelationID = strings.TrimSpace(in.CorrelationID)
	in.Actor = strings.TrimSpace(in.Actor)
	in.NodeID = strings.TrimSpace(in.NodeID)
	in.Scenario = strings.TrimSpace(in.Scenario)
	in.Command = strings.TrimSpace(in.Command)
	in.Args = trimAll(in.Args)
	return in
}

func trimAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, value := range in {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func (s *service) auditAccepted(ctx context.Context, request Request) error {
	if s.audit == nil {
		return nil
	}
	_, err := s.audit.Append(ctx, audit.Record{
		Action: audit.ActionDispatch, Actor: actor(request), NodeID: request.NodeID,
		Scenario: request.Scenario, Verb: request.Command, Args: request.Args,
		Outcome: audit.OutcomeAccepted, Detail: "relay accepted", RunID: request.CorrelationID,
	})
	return err
}

func (s *service) auditTerminal(ctx context.Context, request Request, outcome audit.Outcome, detail string) {
	if s.audit == nil {
		return
	}
	_, _ = s.audit.Append(ctx, audit.Record{
		Action: audit.ActionDispatch, Actor: actor(request), NodeID: request.NodeID,
		Scenario: request.Scenario, Verb: request.Command, Args: request.Args,
		Outcome: outcome, Detail: detail, RunID: request.CorrelationID,
	})
}

func actor(request Request) string {
	if request.Actor != "" {
		return request.Actor
	}
	return "relay"
}

// Deliver is the node-authenticated handler seam. It is intentionally exposed
// by the broker rather than the service so a response cannot be injected into
// a call without matching both correlation id and node id.
func (s *service) Deliver(ctx context.Context, nodeID string, response Response) error {
	return s.broker.Deliver(ctx, nodeID, response)
}

// Broker returns the response broker used by this service for handler wiring.
func BrokerOf(svc Service) *Broker {
	if concrete, ok := svc.(*service); ok {
		return concrete.broker
	}
	return nil
}
