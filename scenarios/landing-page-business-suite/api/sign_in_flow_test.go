package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
)

const testBinding = "browser-binding-for-tests-0001"

type capturedSignIn struct{ token, code string }

func signInServiceWithCapture(t *testing.T, db *sql.DB, sender administration.MagicLinkSender) (*administration.UserAuthService, *capturedSignIn) {
	t.Helper()
	service := administration.NewUserAuthService(administration.UserAuthServiceOptions{
		Store: db, EmailService: sender, JWTIssuer: "test", BaseURL: "http://localhost:3000/auth/verify", AppName: "Test App",
	})
	captured := &capturedSignIn{}
	service.UseTokenCallback(func(_, token, _ string) { captured.token = token })
	service.UseCodeCallback(func(_, code string) { captured.code = code })
	return service, captured
}

func userExists(t *testing.T, db *sql.DB, email string) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = $1`, email).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count > 0
}

func TestSignInRequestDoesNotCreateAccountUntilVerified(t *testing.T) {
	email := "signin-no-account@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, captured := signInServiceWithCapture(t, db, NewEmailService())
	ctx := context.Background()

	if _, err := service.RequestSignIn(ctx, administration.SignInRequest{Email: email, BrowserBinding: testBinding}); err != nil {
		t.Fatal(err)
	}
	if userExists(t, db, email) {
		t.Fatal("requesting a sign-in created an account before the address was proven")
	}
	result, err := service.VerifySignIn(ctx, administration.SignInVerification{Email: email, Code: captured.code, BrowserBinding: testBinding})
	if err != nil {
		t.Fatal(err)
	}
	if !result.User.EmailVerified || !userExists(t, db, email) {
		t.Fatalf("verified user = %+v", result.User)
	}
}

func TestSignInCodeRequiresTheRequestingBrowser(t *testing.T) {
	email := "signin-code-binding@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, captured := signInServiceWithCapture(t, db, NewEmailService())
	ctx := context.Background()
	if _, err := service.RequestSignIn(ctx, administration.SignInRequest{Email: email, BrowserBinding: testBinding}); err != nil {
		t.Fatal(err)
	}
	_, err := service.VerifySignIn(ctx, administration.SignInVerification{Email: email, Code: captured.code, BrowserBinding: "another-browser-binding-0002"})
	if !errors.Is(err, administration.ErrCodeInvalid) {
		t.Fatalf("code accepted from another browser: %v", err)
	}
	wrong := "000000"
	if captured.code == wrong {
		wrong = "111111"
	}
	if _, err := service.VerifySignIn(ctx, administration.SignInVerification{Email: email, Code: wrong, BrowserBinding: testBinding}); !errors.Is(err, administration.ErrCodeInvalid) {
		t.Fatalf("wrong code: %v", err)
	}
	if _, err := service.VerifySignIn(ctx, administration.SignInVerification{Email: email, Code: captured.code, BrowserBinding: testBinding}); err != nil {
		t.Fatalf("right code from right browser: %v", err)
	}
}

func TestResendKeepsEarlierCodeValidAndSignInRetiresEveryOutstandingLink(t *testing.T) {
	email := "signin-resend@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, captured := signInServiceWithCapture(t, db, NewEmailService())
	ctx := context.Background()

	if _, err := service.RequestSignIn(ctx, administration.SignInRequest{Email: email, BrowserBinding: testBinding}); err != nil {
		t.Fatal(err)
	}
	first := *captured
	if _, err := service.RequestSignIn(ctx, administration.SignInRequest{Email: email, BrowserBinding: testBinding}); err != nil {
		t.Fatal(err)
	}
	second := *captured
	if _, err := service.VerifySignIn(ctx, administration.SignInVerification{Email: email, Code: first.code, BrowserBinding: testBinding}); err != nil {
		t.Fatalf("code from the first email rejected after resend: %v", err)
	}
	if _, err := service.VerifySignIn(ctx, administration.SignInVerification{Token: second.token}); !errors.Is(err, administration.ErrTokenUsed) {
		t.Fatalf("second link still usable after sign-in completed: %v", err)
	}
}

func TestPreviewDoesNotConsumeLinkAndReturnsContextOnlyToSameBrowser(t *testing.T) {
	email := "signin-preview@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, captured := signInServiceWithCapture(t, db, NewEmailService())
	ctx := context.Background()
	requestContext := json.RawMessage(`{"redirect_uri":"http://127.0.0.1:41234/callback","code_challenge":"abc","code_challenge_method":"S256","state":"s"}`)
	if _, err := service.RequestSignIn(ctx, administration.SignInRequest{Email: email, BrowserBinding: testBinding, Context: requestContext}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		preview, err := service.PreviewSignIn(ctx, captured.token, "")
		if err != nil {
			t.Fatalf("preview %d: %v", i, err)
		}
		if preview.Flow != administration.SignInFlowNativeApp || preview.SameBrowser || !strings.Contains(preview.EmailHint, "@example.com") || strings.Contains(preview.EmailHint, "signin-preview") {
			t.Fatalf("preview = %+v", preview)
		}
	}
	result, err := service.VerifySignIn(ctx, administration.SignInVerification{Token: captured.token})
	if err != nil {
		t.Fatalf("link unusable after previews: %v", err)
	}
	if result.SameBrowser || len(result.Context) != 0 {
		t.Fatalf("another browser received the native callback context: %+v", result)
	}
}

func TestSameBrowserLinkReturnsStoredContext(t *testing.T) {
	email := "signin-context@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, captured := signInServiceWithCapture(t, db, NewEmailService())
	ctx := context.Background()
	if _, err := service.RequestSignIn(ctx, administration.SignInRequest{Email: email, BrowserBinding: testBinding, Context: json.RawMessage(`{"app":"Desk","redirect_uri":"http://127.0.0.1:41234/callback"}`)}); err != nil {
		t.Fatal(err)
	}
	result, err := service.VerifySignIn(ctx, administration.SignInVerification{Token: captured.token, BrowserBinding: testBinding})
	if err != nil {
		t.Fatal(err)
	}
	if !result.SameBrowser || !strings.Contains(string(result.Context), "41234") {
		t.Fatalf("result = %+v", result)
	}
}

type failingSignInSender struct{}

func (failingSignInSender) SendMagicLink(string, string, string) error {
	return errors.New("provider down")
}

func (failingSignInSender) SendSignIn(administration.SignInMessage) (*administration.SignInDelivery, error) {
	return nil, errors.New("provider down")
}

type recordingSignInSender struct{}

func (recordingSignInSender) SendMagicLink(string, string, string) error { return nil }
func (recordingSignInSender) SendSignIn(administration.SignInMessage) (*administration.SignInDelivery, error) {
	return &administration.SignInDelivery{Provider: "sendgrid", ProviderMessageID: "provider-message-id"}, nil
}

func TestSignInRequestStoresDeliveryProviderMetadata(t *testing.T) {
	email := "signin-delivery-metadata@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, _ := signInServiceWithCapture(t, db, recordingSignInSender{})
	if _, err := service.RequestSignIn(context.Background(), administration.SignInRequest{Email: email, BrowserBinding: testBinding}); err != nil {
		t.Fatal(err)
	}
	var provider, messageID, status string
	if err := db.QueryRow(`SELECT provider, provider_message_id, delivery_status FROM auth_tokens WHERE email = $1 ORDER BY created_at DESC LIMIT 1`, email).Scan(&provider, &messageID, &status); err != nil {
		t.Fatal(err)
	}
	if provider != "sendgrid" || messageID != "provider-message-id" || status != "sent" {
		t.Fatalf("provider=%q messageID=%q status=%q", provider, messageID, status)
	}
}

func TestUndeliveredSignInIsReportedUsableAndVisibleInHealth(t *testing.T) {
	email := "signin-delivery-failure@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, captured := signInServiceWithCapture(t, db, failingSignInSender{})
	ctx := context.Background()
	_, err := service.RequestSignIn(ctx, administration.SignInRequest{Email: email, BrowserBinding: testBinding})
	if !errors.Is(err, administration.ErrDeliveryUnavailable) {
		t.Fatalf("delivery failure reported as %v", err)
	}
	if _, err := service.VerifySignIn(ctx, administration.SignInVerification{Token: captured.token}); err != nil {
		t.Fatalf("an undelivered link should remain usable for retry: %v", err)
	}
	health, err := service.DeliveryHealth(ctx, time.Hour)
	if err != nil || health.Failed == 0 || health.LastFailure == nil || !strings.Contains(health.LastError, "provider down") {
		t.Fatalf("health = %+v err=%v", health, err)
	}
}

func TestAuthThrottlePersistsAcrossInstances(t *testing.T) {
	db := setupTestDB(t)
	bucket := administration.ThrottleBucket("test-persist", "throttle@example.com")
	_, _ = db.Exec(`DELETE FROM auth_rate_events WHERE bucket = $1`, bucket)
	defer func() { _, _ = db.Exec(`DELETE FROM auth_rate_events WHERE bucket = $1`, bucket) }()
	rule := administration.ThrottleRule{Limit: 2, Window: time.Minute}
	ctx := context.Background()
	first := administration.NewAuthThrottle(db)
	for i := 0; i < 2; i++ {
		if ok, err := first.Allow(ctx, bucket, rule); !ok || err != nil {
			t.Fatalf("attempt %d ok=%t err=%v", i, ok, err)
		}
	}
	restarted := administration.NewAuthThrottle(db)
	if ok, err := restarted.Allow(ctx, bucket, rule); ok || err != nil {
		t.Fatalf("a new instance (restart/replica) forgot the limit: ok=%t err=%v", ok, err)
	}
}

func TestSignInRequestHandlerEnforcesDurablePerEmailLimit(t *testing.T) {
	email := "signin-handler-limit@example.com"
	db := setupTestDB(t)
	defer cleanupUserTestData(t, db, email)
	service, _ := signInServiceWithCapture(t, db, NewEmailService())
	deps := userAuthHandlerDependencies(service, nil)
	deps.Throttle = administration.NewAuthThrottle(db)
	statuses := []int{}
	for i := 0; i <= administration.SignInPerEmail.Limit; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", strings.NewReader(`{"email":"`+email+`"}`))
		request.RemoteAddr = "203.0.113.9:1234"
		adminhttp.RequestMagicLink(deps).ServeHTTP(recorder, request)
		statuses = append(statuses, recorder.Code)
	}
	if statuses[len(statuses)-1] != http.StatusTooManyRequests {
		t.Fatalf("statuses = %v", statuses)
	}
	for _, status := range statuses[:len(statuses)-1] {
		if status != http.StatusOK {
			t.Fatalf("statuses = %v", statuses)
		}
	}
}
