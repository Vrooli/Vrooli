package commerce

import (
	"context"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"landing-page-business-suite-api/internal/opsalert"
)

type alertHTTP struct{}

func (alertHTTP) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
}

func TestAuthDeliveryAlertEvaluatorDispatchesDegradedWindowOnce(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(1_700_000_000, 0)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*), COUNT(*) FILTER")).WillReturnRows(sqlmock.NewRows([]string{"count", "count"}).AddRow(4, 3))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO auth_alert_log")).WithArgs("sign_in_delivery_degraded", sqlmock.AnyArg(), now.UTC().Truncate(5*time.Minute)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE auth_alert_log SET dispatch_status='sent'")).WithArgs("sign_in_delivery_degraded", now.UTC().Truncate(5*time.Minute)).WillReturnResult(sqlmock.NewResult(0, 1))
	evaluator := &AuthDeliveryAlertEvaluator{DB: db, Transport: opsalert.New(), Now: func() time.Time { return now }}
	evaluator.Transport.UseHTTPClient(alertHTTP{})
	if err := evaluator.Evaluate(context.Background(), "https://ops.example.test/hook", true); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
