package scenario

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Call(context.Context, Request) (Response, error)
}

type service struct {
	nodes     NodeReader
	presence  Presence
	pusher    Pusher
	broker    *Broker
	admit     Admission
	maxBytes  uint64
	maxSecond int64
}

type Option func(*service)

func WithAdmission(admit Admission) Option { return func(s *service) { s.admit = admit } }
func WithLimits(maxBytes uint64, maxSeconds int64) Option {
	return func(s *service) {
		if maxBytes > 0 {
			s.maxBytes = maxBytes
		}
		if maxSeconds > 0 {
			s.maxSecond = maxSeconds
		}
	}
}

func NewService(nodes NodeReader, presence Presence, pusher Pusher, broker *Broker, options ...Option) Service {
	if broker == nil {
		broker = NewBroker()
	}
	s := &service{nodes: nodes, presence: presence, pusher: pusher, broker: broker, maxBytes: MaxResponseBytes, maxSecond: 300}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *service) Call(ctx context.Context, request Request) (Response, error) {
	request = normalize(request)
	if request.NodeID == "" || request.Scenario == "" || request.Service == "" || request.Method == "" {
		return Response{}, errors.New("node_id, scenario, service, and method are required")
	}
	if request.CorrelationID == "" {
		request.CorrelationID = uuid.NewString()
	}
	if request.MaxResponseBytes == 0 || request.MaxResponseBytes > s.maxBytes {
		request.MaxResponseBytes = s.maxBytes
	}
	node, err := s.nodes.GetTarget(ctx, request.NodeID)
	if err != nil {
		return Response{}, err
	}
	if node.Revoked {
		return Response{}, errors.New("target node is revoked")
	}
	if s.admit != nil {
		if err := s.admit(request, node); err != nil {
			return Response{}, err
		}
	}
	if !s.presence.IsOnline(request.NodeID) {
		return Response{}, errors.New("target node is offline")
	}
	if !s.presence.Dispatchable(request.NodeID) {
		return Response{}, errors.New("target node requires an agent update")
	}
	responses, unregister, err := s.broker.Register(request.CorrelationID, request.NodeID)
	if err != nil {
		return Response{}, err
	}
	defer unregister()
	if delivered, err := s.pusher.Push(ctx, request.NodeID, request); err != nil || delivered == 0 {
		if err == nil {
			err = errors.New("scenario request was not delivered")
		}
		return Response{}, err
	}
	callCtx := ctx
	if request.TimeoutSeconds > 0 {
		seconds := request.TimeoutSeconds
		if seconds > s.maxSecond {
			seconds = s.maxSecond
		}
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
		defer cancel()
	}
	select {
	case <-callCtx.Done():
		return Response{CorrelationID: request.CorrelationID, Error: callCtx.Err().Error(), TimedOut: errors.Is(callCtx.Err(), context.DeadlineExceeded)}, callCtx.Err()
	case response := <-responses:
		if uint64(len(response.Body)) > request.MaxResponseBytes {
			response.Body = response.Body[:request.MaxResponseBytes]
			response.Truncated = true
		}
		return response, responseError(response)
	}
}

func normalize(request Request) Request {
	request.Actor = strings.TrimSpace(request.Actor)
	request.NodeID = strings.TrimSpace(request.NodeID)
	request.Scenario = strings.TrimSpace(request.Scenario)
	request.Service = strings.Trim(request.Service, "/")
	request.Method = strings.Trim(request.Method, "/")
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.Body = append([]byte(nil), request.Body...)
	return request
}

func responseError(response Response) error {
	if response.Error == "" {
		return nil
	}
	return fmt.Errorf("target scenario: %s", response.Error)
}
