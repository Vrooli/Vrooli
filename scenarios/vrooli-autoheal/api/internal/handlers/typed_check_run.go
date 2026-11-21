package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	checksproto "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
)

var errObservationBusy = errors.New("another observation cycle is running")

// RunCheck is an explicit observation, not an instruction to heal. It shares
// admission with fleet ticks but never advances their completed-tick watermark.
func (s *typedChecks) RunCheck(ctx context.Context, req *connect.Request[checksproto.RunCheckRequest]) (*connect.Response[checksproto.RunCheckResponse], error) {
	id := strings.TrimSpace(req.Msg.GetCheckId())
	if id == "" || len(id) > 256 {
		return nil, invalidArgument("one check_id is required (maximum 256 bytes)")
	}
	if _, ok := s.h.registry.GetCheck(id); !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("check not registered"))
	}
	if err := ctx.Err(); err != nil {
		return nil, observationContextError(err)
	}
	s.h.tickLock.Lock()
	if s.h.tickRunning {
		s.h.tickLock.Unlock()
		return nil, connect.NewError(connect.CodeAlreadyExists, errObservationBusy)
	}
	s.h.tickRunning = true
	s.h.tickStarted = time.Now()
	s.h.tickLock.Unlock()
	defer func() {
		s.h.tickLock.Lock()
		s.h.tickRunning = false
		s.h.tickStarted = time.Time{}
		s.h.tickLock.Unlock()
	}()
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	results := s.h.registry.RunChecksForIDs(ctx, []string{id})
	if err := ctx.Err(); err != nil {
		return nil, observationContextError(err)
	}
	if len(results) != 1 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("check no longer registered"))
	}
	result := results[0]
	if err := s.h.store.SaveResult(ctx, result); err != nil {
		return nil, internalError("persist check observation", err)
	}
	if store, ok := s.h.store.(supervisedAvailabilityStore); ok {
		if err := store.ObserveSupervisedAvailability(ctx, result); err != nil {
			return nil, internalError("persist availability observation", err)
		}
	}
	if s.h.incidentService != nil {
		if _, err := s.h.incidentService.UpsertFromCheckResults(ctx, results); err != nil {
			return nil, internalError("update check incident", err)
		}
	}
	return connect.NewResponse(&checksproto.RunCheckResponse{Result: checkResult(result)}), nil
}

func observationContextError(err error) error {
	code := connect.CodeCanceled
	if errors.Is(err, context.DeadlineExceeded) {
		code = connect.CodeDeadlineExceeded
	}
	return connect.NewError(code, err)
}
