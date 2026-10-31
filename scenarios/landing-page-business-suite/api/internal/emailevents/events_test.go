package emailevents

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMapStatusClassifiesSendGridEvents(t *testing.T) {
	cases := []struct{ event, code, wantDelivery, wantReason string }{
		{"delivered", "", "delivered", ""},
		{"bounce", "550", "rejected", "invalid"},
		{"bounce", "552", "rejected", "mailbox_unavailable"},
		{"bounce", "451", "rejected", "blocked"},
		{"dropped", "", "rejected", "other"},
		{"spamreport", "", "rejected", "spam"},
		{"deferred", "", "deferred", "other"},
	}
	for _, tc := range cases {
		got := MapStatus(Event{Event: tc.event, Status: tc.code})
		if got.Delivery != tc.wantDelivery || got.Reason != tc.wantReason {
			t.Errorf("%s/%s = %#v", tc.event, tc.code, got)
		}
	}
}

func TestRepositoryRecordsAndCorrelatesOnlyNewEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	r := &Repository{DB: db}
	event := Event{Event: "bounce", Timestamp: 1700000000, Email: "user@example.com", SGEventID: "evt-1", SGMessageID: "msg-1", Status: "550", CustomArgs: map[string]string{"lpbs_sign_in_request_id": "token-1"}}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO auth_email_events")).WithArgs("evt-1", "msg-1", "token-1", "bounce", "invalid", event.Timestamp, "user@example.com", "bounce", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE auth_tokens SET provider_status")).WithArgs("bounce", event.Timestamp, "invalid", "rejected", "token-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO email_suppressions")).WithArgs("user@example.com", "invalid", "evt-1").WillReturnResult(sqlmock.NewResult(0, 1))
	accepted, err := r.Record(context.Background(), event, time.Unix(1700000001, 0))
	if err != nil || !accepted {
		t.Fatalf("Record = %v, %v", accepted, err)
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO auth_email_events")).WithArgs("evt-1", "msg-1", "token-1", "bounce", "invalid", event.Timestamp, "user@example.com", "bounce", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	accepted, err = r.Record(context.Background(), event, time.Unix(1700000002, 0))
	if err != nil || accepted {
		t.Fatalf("duplicate Record = %v, %v", accepted, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
