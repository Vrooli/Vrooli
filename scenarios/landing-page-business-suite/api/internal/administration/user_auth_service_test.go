package administration

import (
	"context"
	"strings"
	"testing"
)

func TestRequestMagicLinkRejectsMissingConfiguredOriginBeforeWritingToken(t *testing.T) {
	service := NewUserAuthService(UserAuthServiceOptions{JWTSecret: "test-secret"})

	err := service.RequestMagicLink(context.Background(), "customer@example.test", "", "")
	if err == nil || !strings.Contains(err.Error(), "AUTH_MAGIC_LINK_BASE_URL") {
		t.Fatalf("RequestMagicLink() error = %v, want missing configured origin error", err)
	}
}
