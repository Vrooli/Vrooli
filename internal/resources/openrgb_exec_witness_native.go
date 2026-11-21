//go:build darwin || windows

package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
)

func openRGBExecContext(ref platform.ServiceRef, incarnation string) (string, error) {
	if ref.RootPID != os.Getpid() {
		return "", errors.New("OpenRGB launcher is outside the admitted native root")
	}
	return openRGBNativeSecurityContext(ref, incarnation)
}
func openRGBServingContext(ref platform.ServiceRef, incarnation string) (string, error) {
	return openRGBNativeSecurityContext(ref, incarnation)
}

// Native context binds the observed non-elevated operator and exact manager
// containment. Account identity alone is insufficient to identify a launch.
func openRGBNativeSecurityContext(ref platform.ServiceRef, incarnation string) (string, error) {
	if !ref.NativeOwnedIdentityComplete() || ref.Native == nil || !ref.MatchesIncarnation(incarnation) {
		return "", errors.New("OpenRGB native containment or root incarnation is incomplete")
	}
	current, err := process.Incarnation(ref.RootPID)
	if err != nil || current != incarnation {
		return "", errors.Join(errors.New("OpenRGB native root incarnation changed"), err)
	}
	observed, err := platform.InspectOwnedService(context.Background(), ref)
	if err != nil || !observed.RootAlive || !observed.Populated || observed.State != platform.OperationRunning || !observed.Ref.Equal(platform.OperationRef(ref)) {
		return "", errors.Join(errors.New("OpenRGB native manager identity changed"), err)
	}
	account, err := platform.ResolveOwnedServiceAccount(context.Background(), ref.Native.Account)
	if err != nil {
		return "", err
	}
	security, err := platform.ProcessSecurity(ref.RootPID)
	if err != nil || security.Elevated {
		return "", errors.Join(errors.New("OpenRGB requires an observed non-elevated native account"), err)
	}
	principal := "sid:" + security.Principal
	if ref.Native.Platform == "darwin" {
		principal = "uid:" + security.Principal
	}
	if principal != account.Principal {
		return "", errors.New("OpenRGB process principal differs from the admitted service account")
	}
	final, err := process.Incarnation(ref.RootPID)
	if err != nil || final != incarnation {
		return "", errors.Join(errors.New("OpenRGB native root changed during security observation"), err)
	}
	data, err := json.Marshal(struct {
		Native   *platform.NativeWorkloadIdentity
		Security platform.ProcessSecurityContext
	}{ref.Native, security})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
