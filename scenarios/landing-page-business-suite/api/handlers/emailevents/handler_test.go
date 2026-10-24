package emailevents

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	email "landing-page-business-suite-api/internal/emailevents"
)

func TestHandlerRejectsUnsignedAndAcceptsSignedBatch(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := base64.StdEncoding.EncodeToString(der)
	now := time.Unix(1_700_000_000, 0)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	requestID := "11111111-1111-1111-1111-111111111111"
	body, _ := json.Marshal([]email.Event{{Event: "delivered", Timestamp: now.Unix(), Email: "user@example.com", SGEventID: "evt-delivered", SGMessageID: "msg.abc", CustomArgs: map[string]string{"lpbs_sign_in_request_id": requestID}}})
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO auth_email_events")).WithArgs("evt-delivered", "msg.abc", requestID, "delivered", "", now.Unix(), "user@example.com", "delivered", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE auth_tokens SET provider_status")).WithArgs("delivered", now.Unix(), "", "delivered", requestID).WillReturnResult(sqlmock.NewResult(0, 1))

	h := Handler(Dependencies{Store: &email.Repository{DB: db}, PublicKey: func() string { return publicKey }, Now: func() time.Time { return now }})
	unsigned := httptest.NewRecorder()
	h.ServeHTTP(unsigned, httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/sendgrid", bytes.NewReader(body)))
	if unsigned.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned status = %d", unsigned.Code)
	}

	timestamp := "1700000000"
	digest := sha256.Sum256(append([]byte(timestamp), body...))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	sig, err := asn1.Marshal(struct{ R, S *big.Int }{r, s})
	if err != nil {
		t.Fatal(err)
	}
	signed := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/sendgrid", bytes.NewReader(body))
	req.Header.Set("X-Twilio-Email-Event-Webhook-Timestamp", timestamp)
	req.Header.Set("X-Twilio-Email-Event-Webhook-Signature", base64.StdEncoding.EncodeToString(sig))
	h.ServeHTTP(signed, req)
	if signed.Code != http.StatusOK {
		t.Fatalf("signed status = %d, body=%s", signed.Code, signed.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
