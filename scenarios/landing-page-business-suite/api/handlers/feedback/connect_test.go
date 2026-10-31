package feedback

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	metrics "landing-page-business-suite-api/internal/metrics"
)

type feedbackServiceFake struct {
	createdInput  *metrics.CreateFeedbackInput
	created       *metrics.FeedbackRequest
	createErr     error
	listedStatus  string
	listed        []metrics.FeedbackRequest
	getErr        error
	updatedStatus string
	deletedIDs    []int
	bulkIDs       []int
}

func (f *feedbackServiceFake) Create(_ context.Context, input *metrics.CreateFeedbackInput) (*metrics.FeedbackRequest, error) {
	f.createdInput = input
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.created == nil {
		f.created = &metrics.FeedbackRequest{ID: 7, Type: input.Type, Email: input.Email, Subject: input.Subject, Message: input.Message}
	}
	return f.created, nil
}

func (f *feedbackServiceFake) List(_ context.Context, status string) ([]metrics.FeedbackRequest, error) {
	f.listedStatus = status
	return f.listed, nil
}

func (f *feedbackServiceFake) GetByID(_ context.Context, _ int) (*metrics.FeedbackRequest, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &metrics.FeedbackRequest{ID: 8, Type: "bug", Status: "pending"}, nil
}

func (f *feedbackServiceFake) UpdateStatus(_ context.Context, id int, status string) (*metrics.FeedbackRequest, error) {
	f.updatedStatus = status
	return &metrics.FeedbackRequest{ID: id, Type: "general", Status: status}, nil
}

func (f *feedbackServiceFake) Delete(_ context.Context, id int) error {
	f.deletedIDs = append(f.deletedIDs, id)
	return nil
}

func (f *feedbackServiceFake) DeleteBulk(_ context.Context, ids []int) (int64, error) {
	f.bulkIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

type feedbackNotifierFake struct{ notified []*metrics.FeedbackRequest }

func (f *feedbackNotifierFake) Notify(item *metrics.FeedbackRequest) {
	f.notified = append(f.notified, item)
}

var _ metrics.FeedbackServicer = (*feedbackServiceFake)(nil)

func TestConnectCreateFeedbackDefaultsUnknownTypeAndNotifiesAfterPersistence(t *testing.T) {
	service := &feedbackServiceFake{}
	notifier := &feedbackNotifierFake{}
	handler := NewConnectHandler(service, notifier)

	response, err := handler.CreateFeedback(context.Background(), connect.NewRequest(&lpbsv1.FeedbackCreateRequest{
		Type: "unexpected", Email: "customer@example.test", Subject: "Question", Message: "Hello",
	}))
	if err != nil {
		t.Fatalf("CreateFeedback() error = %v", err)
	}
	if service.createdInput.Type != "general" {
		t.Fatalf("persisted type = %q, want general", service.createdInput.Type)
	}
	if response.Msg.GetId() != 7 || !response.Msg.GetSuccess() {
		t.Fatalf("response = %#v, want successful persisted feedback", response.Msg)
	}
	if len(notifier.notified) != 1 || notifier.notified[0] != service.created {
		t.Fatalf("notifier = %#v, want exactly persisted feedback", notifier.notified)
	}
}

func TestConnectCreateFeedbackRejectsInvalidInputWithoutSideEffects(t *testing.T) {
	service := &feedbackServiceFake{}
	notifier := &feedbackNotifierFake{}
	_, err := NewConnectHandler(service, notifier).CreateFeedback(context.Background(), connect.NewRequest(&lpbsv1.FeedbackCreateRequest{Subject: "Question", Message: "Hello"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("CreateFeedback() code = %v, want invalid argument", connect.CodeOf(err))
	}
	if service.createdInput != nil || len(notifier.notified) != 0 {
		t.Fatal("invalid feedback must not persist or notify")
	}
}

func TestConnectFeedbackAdminOperationsUseTypedContracts(t *testing.T) {
	createdAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	service := &feedbackServiceFake{listed: []metrics.FeedbackRequest{{ID: 3, Type: "feature", Status: "in_progress", CreatedAt: createdAt, UpdatedAt: createdAt}}}
	handler := NewConnectHandler(service, nil)

	status := lpbsv1.FeedbackStatus_FEEDBACK_STATUS_IN_PROGRESS
	listed, err := handler.ListFeedback(context.Background(), connect.NewRequest(&lpbsv1.ListFeedbackRequest{Status: &status}))
	if err != nil {
		t.Fatalf("ListFeedback() error = %v", err)
	}
	if service.listedStatus != "in_progress" || listed.Msg.GetFeedback()[0].GetType() != lpbsv1.FeedbackType_FEEDBACK_TYPE_FEATURE {
		t.Fatalf("ListFeedback() = %#v, status = %q", listed.Msg, service.listedStatus)
	}

	updated, err := handler.UpdateFeedbackStatus(context.Background(), connect.NewRequest(&lpbsv1.UpdateFeedbackStatusRequest{Id: 3, Status: lpbsv1.FeedbackStatus_FEEDBACK_STATUS_RESOLVED}))
	if err != nil || service.updatedStatus != "resolved" || updated.Msg.GetFeedback().GetStatus() != lpbsv1.FeedbackStatus_FEEDBACK_STATUS_RESOLVED {
		t.Fatalf("UpdateFeedbackStatus() = %#v, %v; stored status = %q", updated, err, service.updatedStatus)
	}

	deleted, err := handler.DeleteFeedbackBulk(context.Background(), connect.NewRequest(&lpbsv1.DeleteFeedbackBulkRequest{Ids: []int64{3, 4}}))
	if err != nil || deleted.Msg.GetDeleted() != 2 || len(service.bulkIDs) != 2 {
		t.Fatalf("DeleteFeedbackBulk() = %#v, %v; ids = %#v", deleted, err, service.bulkIDs)
	}
}

func TestConnectGetFeedbackMapsMissingRecordToNotFound(t *testing.T) {
	handler := NewConnectHandler(&feedbackServiceFake{getErr: sql.ErrNoRows}, nil)
	_, err := handler.GetFeedback(context.Background(), connect.NewRequest(&lpbsv1.GetFeedbackRequest{Id: 99}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("GetFeedback() code = %v, want not found", connect.CodeOf(err))
	}
}

func TestConnectCreateFeedbackDoesNotNotifyWhenPersistenceFails(t *testing.T) {
	service := &feedbackServiceFake{createErr: errors.New("database unavailable")}
	notifier := &feedbackNotifierFake{}
	_, err := NewConnectHandler(service, notifier).CreateFeedback(context.Background(), connect.NewRequest(&lpbsv1.FeedbackCreateRequest{Email: "customer@example.test", Subject: "Question", Message: "Hello"}))
	if connect.CodeOf(err) != connect.CodeInternal || len(notifier.notified) != 0 {
		t.Fatalf("CreateFeedback() = %v, notifications = %#v", err, notifier.notified)
	}
}

func TestRegisterConnectRoutesProtectsEveryAdminProcedure(t *testing.T) {
	router := mux.NewRouter()
	requireAdmin := func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusUnauthorized)
		}
	}
	RegisterConnectRoutes(router, &feedbackServiceFake{}, nil, requireAdmin)

	for _, procedure := range []string{
		lpbsconnect.FeedbackServiceListFeedbackProcedure,
		lpbsconnect.FeedbackServiceGetFeedbackProcedure,
		lpbsconnect.FeedbackServiceUpdateFeedbackStatusProcedure,
		lpbsconnect.FeedbackServiceDeleteFeedbackProcedure,
		lpbsconnect.FeedbackServiceDeleteFeedbackBulkProcedure,
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, procedure, nil))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s status = %d, want %d", procedure, response.Code, http.StatusUnauthorized)
		}
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, lpbsconnect.FeedbackServiceCreateFeedbackProcedure, nil))
	if response.Code == http.StatusUnauthorized {
		t.Fatal("public feedback creation must not require an admin session")
	}
}
