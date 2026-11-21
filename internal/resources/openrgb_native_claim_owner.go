package resources

import (
	"context"
	"errors"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// Admission belongs to the ordinary calling operator. A machine supervisor
// PID is neither required nor substituted for the exact kernel owner facts.
type openRGBNativeClaimOwnerOps struct {
	pid         func() int
	principal   func() (string, error)
	incarnation func(int) (string, error)
	security    func(int) (platform.ProcessSecurityContext, error)
	account     func(context.Context, string) (platform.OwnedServiceAccount, error)
}

func verifyOpenRGBNativeClaimOwner(ctx context.Context, c openRGBNativeCapsule, o openRGBNativeClaimOwnerOps) error {
	if ctx == nil || c.validate() != nil || o.pid == nil || o.principal == nil || o.incarnation == nil || o.security == nil || o.account == nil {
		return errOpenRGBNativeRefused
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	pid := o.pid()
	if pid <= 0 || pid != c.OwnerPID {
		return errOpenRGBNativeRefused
	}
	principal, err := o.principal()
	if err != nil || principal != c.OwnerPrincipal {
		return errors.Join(errOpenRGBNativeRefused, err)
	}
	incarnation, err := o.incarnation(pid)
	if err != nil || incarnation == "" || incarnation != c.OwnerIncarnation {
		return errors.Join(errOpenRGBNativeRefused, err)
	}
	security, err := o.security(pid)
	if err != nil || security.Elevated {
		return errors.Join(errOpenRGBNativeRefused, err)
	}
	goos := "windows"
	raw := strings.TrimPrefix(c.Account, "sid:")
	if strings.HasPrefix(c.Account, "uid:") {
		goos = "darwin"
		raw = strings.TrimPrefix(c.Account, "uid:")
	}
	if hostreqkit.ValidateNativeOpenRGBAccount(goos, c.Account) != nil || security.Principal != raw || nativeCapsuleOwnerPrincipal(c.Account) != principal {
		return errOpenRGBNativeRefused
	}
	account, err := o.account(ctx, c.Account)
	if err != nil || account.Name != c.Account || account.Principal != c.Account {
		return errors.Join(errOpenRGBNativeRefused, err)
	}
	return ctx.Err()
}
