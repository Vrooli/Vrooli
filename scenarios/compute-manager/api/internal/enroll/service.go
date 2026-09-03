// Package enroll delegates node trust to vrooli-bridge. It deliberately has
// no SSH imports or host-repair implementation.
package enroll

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Bridge interface {
	GetOnboardingPublicKey(context.Context) (publicKey, fingerprint string, err error)
	StartOnboarding(context.Context, string, string, string) (operationID string, err error)
}

func RenderFirstBoot(publicKey string, expiry time.Time) (string, error) {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" || strings.ContainsAny(publicKey, "\r\n") || strings.Contains(publicKey, "PRIVATE KEY") || !strings.HasPrefix(publicKey, "ssh-") {
		return "", fmt.Errorf("invalid onboarding public key")
	}
	return fmt.Sprintf("#!/bin/sh\nset -eu\numask 077\nmkdir -p /root/.ssh\nprintf '%%s\\n' '%s' >> /root/.ssh/authorized_keys\ncat > /etc/systemd/system/vrooli-expiry.timer <<'EOF'\n[Timer]\nOnCalendar=%s\nPersistent=true\nEOF\nsystemctl enable --now vrooli-expiry.timer\n", strings.ReplaceAll(publicKey, "'", "'\\''"), expiry.UTC().Format("2006-01-02 15:04:05 UTC")), nil
}

type Service struct{ Bridge Bridge }

func (s Service) Start(ctx context.Context, host, user string) (string, error) {
	if s.Bridge == nil {
		return "", fmt.Errorf("bridge enrollment service unavailable")
	}
	return s.Bridge.StartOnboarding(ctx, host, user, "")
}
