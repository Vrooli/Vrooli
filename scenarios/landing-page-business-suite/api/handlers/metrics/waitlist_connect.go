package metricshttp

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	metrics "landing-page-business-suite-api/internal/metrics"
)

// WaitlistConnectDependencies keeps transport validation at the composition
// boundary while the metrics domain owns persistence.
type WaitlistConnectDependencies struct {
	Service       metrics.WaitlistServicer
	ValidateEmail func(string) (string, error)
}

type waitlistConnectHandler struct{ deps WaitlistConnectDependencies }

func (h *waitlistConnectHandler) CreateWaitlistEntry(ctx context.Context, request *connect.Request[lpbsv1.CreateWaitlistEntryRequest]) (*connect.Response[lpbsv1.CreateWaitlistEntryResponse], error) {
	email, err := h.deps.ValidateEmail(request.Msg.GetEmail())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	source := strings.TrimSpace(request.Msg.GetSource())
	if source == "" {
		source = "coming_soon"
	}
	entry, err := h.deps.Service.Create(ctx, email, source)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create waitlist entry: %w", err))
	}
	return connect.NewResponse(&lpbsv1.CreateWaitlistEntryResponse{Success: true, Message: "Email added to waitlist", Entry: waitlistEntryProto(entry)}), nil
}

func (h *waitlistConnectHandler) ListWaitlistEntries(ctx context.Context, _ *connect.Request[lpbsv1.ListWaitlistEntriesRequest]) (*connect.Response[lpbsv1.ListWaitlistEntriesResponse], error) {
	entries, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list waitlist entries: %w", err))
	}
	result := make([]*lpbsv1.WaitlistEntry, 0, len(entries))
	for index := range entries {
		result = append(result, waitlistEntryProto(&entries[index]))
	}
	return connect.NewResponse(&lpbsv1.ListWaitlistEntriesResponse{Entries: result}), nil
}

func (h *waitlistConnectHandler) DeleteWaitlistEntry(ctx context.Context, request *connect.Request[lpbsv1.DeleteWaitlistEntryRequest]) (*connect.Response[lpbsv1.DeleteWaitlistEntryResponse], error) {
	id := request.Msg.GetId()
	if id <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("waitlist entry ID must be positive"))
	}
	if err := h.deps.Service.Delete(ctx, id); errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("waitlist entry not found"))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete waitlist entry: %w", err))
	}
	return connect.NewResponse(&lpbsv1.DeleteWaitlistEntryResponse{Deleted: true}), nil
}

func (h *waitlistConnectHandler) ExportWaitlistEntries(ctx context.Context, _ *connect.Request[lpbsv1.ExportWaitlistEntriesRequest]) (*connect.Response[lpbsv1.ExportWaitlistEntriesResponse], error) {
	entries, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("export waitlist entries: %w", err))
	}
	var output strings.Builder
	writer := csv.NewWriter(&output)
	if err := writer.Write([]string{"ID", "Email", "Source", "Created At"}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write waitlist CSV header: %w", err))
	}
	for _, entry := range entries {
		if err := writer.Write([]string{fmt.Sprintf("%d", entry.ID), entry.Email, entry.Source, entry.CreatedAt.Format("2006-01-02 15:04:05")}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write waitlist CSV row: %w", err))
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("flush waitlist CSV: %w", err))
	}
	return connect.NewResponse(&lpbsv1.ExportWaitlistEntriesResponse{Csv: output.String(), Filename: "waitlist.csv"}), nil
}

func waitlistEntryProto(entry *metrics.WaitlistEmail) *lpbsv1.WaitlistEntry {
	if entry == nil {
		return &lpbsv1.WaitlistEntry{}
	}
	return &lpbsv1.WaitlistEntry{Id: entry.ID, Email: entry.Email, Source: entry.Source, CreatedAt: timestamppb.New(entry.CreatedAt)}
}

func RegisterWaitlistConnectRoutes(router *mux.Router, deps WaitlistConnectDependencies, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, handler := lpbsconnect.NewWaitlistServiceHandler(&waitlistConnectHandler{deps: deps})
	router.Handle(lpbsconnect.WaitlistServiceCreateWaitlistEntryProcedure, handler).Methods(http.MethodPost)
	for _, procedure := range []string{lpbsconnect.WaitlistServiceListWaitlistEntriesProcedure, lpbsconnect.WaitlistServiceDeleteWaitlistEntryProcedure, lpbsconnect.WaitlistServiceExportWaitlistEntriesProcedure} {
		router.Handle(procedure, requireAdmin(handler.ServeHTTP)).Methods(http.MethodPost)
	}
}

var _ lpbsconnect.WaitlistServiceHandler = (*waitlistConnectHandler)(nil)
