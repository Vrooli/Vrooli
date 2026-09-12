package main

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	walkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type walkConnectService struct {
	server *Server
	ledger journalconnect.JournalServiceClient
}

func (s walkConnectService) Read(ctx context.Context, req *connect.Request[walkv1.ReadRequest]) (*connect.Response[walkv1.ReadResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit == 0 {
		limit = 40
	}
	if limit < 1 || limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("limit must be 1-100"))
	}
	// Reuse the board's owner-qualified reading path; never derive health from activity.
	entries, _ := s.server.readings(ctx, s.server.registry.Metrics)
	out := &walkv1.ReadResponse{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Total: int32(len(entries)), Truncated: len(entries) > limit}
	for i, m := range entries {
		if i >= limit {
			break
		}
		out.Readings = append(out.Readings, walkReading(m))
	}
	return connect.NewResponse(out), nil
}

func walkReading(m MetricEntry) *walkv1.Reading {
	r := &walkv1.Reading{Id: m.ID, Label: m.Label, Owner: m.Source.Team, Source: m.Source.Read, Coverage: string(m.Coverage), Trust: string(m.Trust), Empirical: string(m.Empirical), Unit: m.Unit, TtlSeconds: int32(m.TTLSeconds), Reason: m.TrustReason}
	if m.Owner != nil {
		r.Owner = *m.Owner
	}
	if m.ObservedAt != nil {
		r.ObservedAt = m.ObservedAt.UTC().Format(time.RFC3339Nano)
	}
	if r.Reason == "" && m.WhatIsNeeded != nil {
		r.Reason = *m.WhatIsNeeded
	}
	// Samples and panel payloads are deliberately excluded from the operator briefing.
	if m.Value != nil {
		value, err := structpb.NewValue(m.Value)
		if err == nil {
			r.Value = value
		} else {
			r.Trust = string(TrustUntrusted)
			r.Reason = "unsupported reading value"
		}
	}
	return r
}
