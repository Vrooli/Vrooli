package credentials

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials"
	internalcredentials "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentials"
)

func TestRevealCredentialRequiresConfirmationAndReturnsOneValue(t *testing.T) {
	handler := NewConnectHandler(internalcredentials.Service{RevealFn: func(context.Context, string, string) (string, error) {
		return "s3cr3t-value", nil
	}})

	if _, err := handler.RevealCredential(context.Background(), connect.NewRequest(&credentialsv1.RevealCredentialRequest{
		LogicalId: "vrooli/demo", Field: "token",
	})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfirmed reveal code = %v, err = %v", connect.CodeOf(err), err)
	}

	response, err := handler.RevealCredential(context.Background(), connect.NewRequest(&credentialsv1.RevealCredentialRequest{
		LogicalId: "vrooli/demo", Field: "token", ConfirmReveal: true,
	}))
	if err != nil {
		t.Fatalf("confirmed reveal returned error: %v", err)
	}
	if response.Msg.GetValue() != "s3cr3t-value" {
		t.Fatalf("value = %q, want the resolved value", response.Msg.GetValue())
	}
	if response.Msg.GetLogicalId() != "vrooli/demo" || response.Msg.GetField() != "token" {
		t.Fatalf("identity not preserved: %+v", response.Msg)
	}
}

func TestRevealCredentialAuditsWithoutLoggingTheValue(t *testing.T) {
	const secret = "value-must-not-appear-in-logs"
	handler := NewConnectHandler(internalcredentials.Service{RevealFn: func(context.Context, string, string) (string, error) {
		return secret, nil
	}})

	var logged bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	if _, err := handler.RevealCredential(context.Background(), connect.NewRequest(&credentialsv1.RevealCredentialRequest{
		LogicalId: "vrooli/demo", Field: "token", ConfirmReveal: true,
	})); err != nil {
		t.Fatalf("reveal returned error: %v", err)
	}
	if strings.Contains(logged.String(), secret) {
		t.Fatalf("audit log leaked the value: %s", logged.String())
	}
	if !strings.Contains(logged.String(), "onboarding_credential_revealed") {
		t.Fatalf("audit event missing: %s", logged.String())
	}
}

func TestRevealCredentialMapsUnavailableAndUnconfigured(t *testing.T) {
	t.Run("unconfigured is not found", func(t *testing.T) {
		handler := NewConnectHandler(internalcredentials.Service{RevealFn: func(context.Context, string, string) (string, error) {
			return "", fmt.Errorf("resolve: %w", credentialauthority.ErrUnconfigured)
		}})
		_, err := handler.RevealCredential(context.Background(), connect.NewRequest(&credentialsv1.RevealCredentialRequest{
			LogicalId: "vrooli/demo", Field: "token", ConfirmReveal: true,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, err = %v", connect.CodeOf(err), err)
		}
	})
	t.Run("authority down is unavailable", func(t *testing.T) {
		handler := NewConnectHandler(internalcredentials.Service{RevealFn: func(context.Context, string, string) (string, error) {
			return "", fmt.Errorf("resolve: %w", credentialauthority.ErrProviderUnavailable)
		}})
		_, err := handler.RevealCredential(context.Background(), connect.NewRequest(&credentialsv1.RevealCredentialRequest{
			LogicalId: "vrooli/demo", Field: "token", ConfirmReveal: true,
		}))
		if connect.CodeOf(err) != connect.CodeUnavailable {
			t.Fatalf("code = %v, err = %v", connect.CodeOf(err), err)
		}
	})
}
