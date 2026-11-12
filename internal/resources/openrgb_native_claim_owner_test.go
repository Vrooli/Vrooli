package resources

import (
	"context"
	"errors"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func TestNativeClaimOwnerAllowsOrdinaryCLIWithoutSupervisorSelfSlot(t *testing.T) {
	c, _, _, _, _ := nativeTransportFixture(t)
	c.OwnerPID = 123
	c.OwnerPrincipal = "unix:501"
	c.OwnerIncarnation = "operator-cli-birth"
	// This kernel owner is an ordinary CLI; no machine-supervisor PID or
	// enrollment observation is provided to the admission decision.
	ops := openRGBNativeClaimOwnerOps{pid: func() int { return 123 }, principal: func() (string, error) { return "unix:501", nil }, incarnation: func(pid int) (string, error) {
		if pid != 123 {
			t.Fatal("foreign PID inspected")
		}
		return "operator-cli-birth", nil
	}, security: func(int) (platform.ProcessSecurityContext, error) {
		return platform.ProcessSecurityContext{Principal: "501", EffectiveUID: 501}, nil
	}, account: func(context.Context, string) (platform.OwnedServiceAccount, error) {
		return platform.OwnedServiceAccount{Name: "uid:501", Principal: "uid:501"}, nil
	}}
	if err := verifyOpenRGBNativeClaimOwner(t.Context(), c, ops); err != nil {
		t.Fatal("ordinary exact operator CLI refused", err)
	}
	for _, name := range []string{"foreignPID", "foreignPrincipal", "oldIncarnation", "elevated", "unknownSecurity", "foreignAccount", "foreignSecurity"} {
		t.Run(name, func(t *testing.T) {
			copy := ops
			switch name {
			case "foreignPID":
				copy.pid = func() int { return 456 }
			case "foreignPrincipal":
				copy.principal = func() (string, error) { return "unix:502", nil }
			case "oldIncarnation":
				copy.incarnation = func(int) (string, error) { return "reused-cli-PID", nil }
			case "elevated":
				copy.security = func(int) (platform.ProcessSecurityContext, error) {
					return platform.ProcessSecurityContext{Principal: "501", Elevated: true}, nil
				}
			case "unknownSecurity":
				copy.security = func(int) (platform.ProcessSecurityContext, error) {
					return platform.ProcessSecurityContext{}, errors.New("kernel token unavailable")
				}
			case "foreignAccount":
				copy.account = func(context.Context, string) (platform.OwnedServiceAccount, error) {
					return platform.OwnedServiceAccount{Name: "uid:501", Principal: "uid:502"}, nil
				}
			case "foreignSecurity":
				copy.security = func(int) (platform.ProcessSecurityContext, error) {
					return platform.ProcessSecurityContext{Principal: "502"}, nil
				}
			}
			if err := verifyOpenRGBNativeClaimOwner(t.Context(), c, copy); err == nil {
				t.Fatal("foreign, elevated or unobserved owner admitted")
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	ops.pid = func() int { t.Fatal("cancelled admission observed process"); return 123 }
	if err := verifyOpenRGBNativeClaimOwner(ctx, c, ops); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestNativeClaimOwnerAcceptsActualEntraOperatorWithoutServiceIdentity(t *testing.T) {
	c, _, _, _, _ := nativeTransportFixture(t)
	account := "sid:S-1-12-1-4141952679-1282074057-627758481-2916039507"
	raw := strings.TrimPrefix(account, "sid:")
	c.Account = account
	c.Intent.NativeAccount = account
	c.OwnerPID = 123
	c.OwnerPrincipal = "windows:" + raw
	c.OwnerIncarnation = "windows-cli-birth"
	ops := openRGBNativeClaimOwnerOps{pid: func() int { return 123 }, principal: func() (string, error) { return c.OwnerPrincipal, nil }, incarnation: func(int) (string, error) { return c.OwnerIncarnation, nil }, security: func(int) (platform.ProcessSecurityContext, error) {
		return platform.ProcessSecurityContext{Principal: raw, EffectiveUID: -1}, nil
	}, account: func(context.Context, string) (platform.OwnedServiceAccount, error) {
		return platform.OwnedServiceAccount{Name: account, Principal: account}, nil
	}}
	if err := verifyOpenRGBNativeClaimOwner(t.Context(), c, ops); err != nil {
		t.Fatal("actual standard Entra operator refused", err)
	}
}
