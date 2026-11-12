//go:build windows

package platform

import (
	"context"
	"golang.org/x/sys/windows"
	"testing"
)

func TestWindowsActualOperatorPeerPreparationHasNoGrantLifecycle(t *testing.T) {
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		t.Fatal(e)
	}
	account := "sid:" + user.User.Sid.String()
	_, standard := windowsProcessOperatorSecurityProof(windows.CurrentProcess(), account)
	cleanup, e := prepareOwnedServicePeerObservation(context.Background(), account)
	if standard != nil {
		if e == nil {
			t.Fatal("nonstandard operator accepted")
		}
		return
	}
	if e != nil {
		t.Fatal(e)
	}
	if e = cleanup(); e != nil {
		t.Fatal(e)
	}
	if e = cleanup(); e != nil {
		t.Fatal("cleanup not idempotent", e)
	}
}
func TestWindowsActualOperatorPeerRefusesRetiredAndBroadAccounts(t *testing.T) {
	for _, account := range []string{`NT SERVICE\vrooli-owned-openrgb`, "LocalSystem", "sid:S-1-5-18", "sid:S-1-1-0"} {
		if _, e := prepareOwnedServicePeerObservation(context.Background(), account); e == nil {
			t.Fatal("retired/broad peer accepted", account)
		}
	}
}
