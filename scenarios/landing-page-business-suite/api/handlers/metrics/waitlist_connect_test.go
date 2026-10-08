package metricshttp

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	metrics "landing-page-business-suite-api/internal/metrics"
)

type waitlistConnectFake struct {
	created   metrics.WaitlistEmail
	entries   []metrics.WaitlistEmail
	createErr error
	listErr   error
	deleteErr error
	deletedID int64
}

func (f *waitlistConnectFake) Create(_ context.Context, email, source string) (*metrics.WaitlistEmail, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = metrics.WaitlistEmail{ID: 7, Email: email, Source: source, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	return &f.created, nil
}

func (f *waitlistConnectFake) List(context.Context) ([]metrics.WaitlistEmail, error) {
	return f.entries, f.listErr
}

func (f *waitlistConnectFake) Delete(_ context.Context, id int64) error {
	f.deletedID = id
	return f.deleteErr
}

func (f *waitlistConnectFake) Count(context.Context) (int64, error) {
	return int64(len(f.entries)), nil
}

func TestWaitlistConnectCreatesNormalizedPublicEntry(t *testing.T) {
	fake := &waitlistConnectFake{}
	handler := &waitlistConnectHandler{deps: WaitlistConnectDependencies{
		Service: fake,
		ValidateEmail: func(value string) (string, error) {
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "" {
				return "", errors.New("email is required")
			}
			return value, nil
		},
	}}

	response, err := handler.CreateWaitlistEntry(context.Background(), connect.NewRequest(&lpbsv1.CreateWaitlistEntryRequest{Email: " CUSTOMER@example.com "}))
	if err != nil {
		t.Fatal(err)
	}
	if fake.created.Email != "customer@example.com" || fake.created.Source != "coming_soon" || response.Msg.GetEntry().GetId() != 7 {
		t.Fatalf("created entry = %+v, response = %+v", fake.created, response.Msg)
	}
	_, err = handler.CreateWaitlistEntry(context.Background(), connect.NewRequest(&lpbsv1.CreateWaitlistEntryRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid email code = %v", connect.CodeOf(err))
	}
}

func TestWaitlistConnectListsDeletesAndExportsTypedEntries(t *testing.T) {
	fake := &waitlistConnectFake{entries: []metrics.WaitlistEmail{{ID: 8, Email: "reader@example.com", Source: "campaign", CreatedAt: time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)}}}
	handler := &waitlistConnectHandler{deps: WaitlistConnectDependencies{Service: fake, ValidateEmail: func(value string) (string, error) { return value, nil }}}

	list, err := handler.ListWaitlistEntries(context.Background(), connect.NewRequest(&lpbsv1.ListWaitlistEntriesRequest{}))
	if err != nil || list.Msg.GetEntries()[0].GetEmail() != "reader@example.com" || !list.Msg.GetEntries()[0].GetCreatedAt().AsTime().Equal(fake.entries[0].CreatedAt) {
		t.Fatalf("list response = %+v, err=%v", list, err)
	}
	export, err := handler.ExportWaitlistEntries(context.Background(), connect.NewRequest(&lpbsv1.ExportWaitlistEntriesRequest{}))
	if err != nil || export.Msg.GetFilename() != "waitlist.csv" || !strings.Contains(export.Msg.GetCsv(), "8,reader@example.com,campaign") {
		t.Fatalf("export response = %+v, err=%v", export, err)
	}
	deleted, err := handler.DeleteWaitlistEntry(context.Background(), connect.NewRequest(&lpbsv1.DeleteWaitlistEntryRequest{Id: 8}))
	if err != nil || !deleted.Msg.GetDeleted() || fake.deletedID != 8 {
		t.Fatalf("delete response = %+v, id=%d, err=%v", deleted, fake.deletedID, err)
	}
}

func TestWaitlistConnectMapsInvalidAndNotFoundDeletes(t *testing.T) {
	fake := &waitlistConnectFake{deleteErr: sql.ErrNoRows}
	handler := &waitlistConnectHandler{deps: WaitlistConnectDependencies{Service: fake, ValidateEmail: func(value string) (string, error) { return value, nil }}}
	_, err := handler.DeleteWaitlistEntry(context.Background(), connect.NewRequest(&lpbsv1.DeleteWaitlistEntryRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("zero ID code = %v", connect.CodeOf(err))
	}
	_, err = handler.DeleteWaitlistEntry(context.Background(), connect.NewRequest(&lpbsv1.DeleteWaitlistEntryRequest{Id: 99}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing ID code = %v", connect.CodeOf(err))
	}
}

func TestWaitlistConnectProtectsEveryAdministrativeProcedure(t *testing.T) {
	router := mux.NewRouter()
	RegisterWaitlistConnectRoutes(router, WaitlistConnectDependencies{Service: &waitlistConnectFake{}, ValidateEmail: func(value string) (string, error) { return value, nil }}, func(http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }
	})
	for _, path := range []string{lpbsconnect.WaitlistServiceListWaitlistEntriesProcedure, lpbsconnect.WaitlistServiceDeleteWaitlistEntryProcedure, lpbsconnect.WaitlistServiceExportWaitlistEntriesProcedure} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusUnauthorized)
		}
	}
}
