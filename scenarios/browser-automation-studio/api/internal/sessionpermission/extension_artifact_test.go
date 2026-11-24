package sessionpermission

import (
	"context"
	"errors"
	"testing"
)

func TestExtensionUnloadRequiresTheSameExplicitGrantAsLoad(t *testing.T) {
	profile := Profile{
		ID: "profile", Effect: EffectAct, Approval: ApprovalNone, Control: true, View: true,
		ExtensionAccess: true, ExtensionLoad: true, URLAllow: []string{"https://example.test/*"},
		ExtensionPermissions: map[string][]string{"abcdefghijklmnopabcdefghijklmnop": {"storage"}},
	}
	policy, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := ActionRequest{SessionID: "session", LeaseID: "lease", OwnerID: "owner", CallerID: "caller", Kind: "extension_unload", ExtensionID: "abcdefghijklmnopabcdefghijklmnop", CurrentURL: "https://example.test/page"}
	if err := policy.Authorize(context.Background(), request); err != nil {
		t.Fatalf("allowlisted unload denied: %v", err)
	}
	profile.ExtensionLoad = false
	denied, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := denied.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("unload without the extension lifecycle grant accepted: %v", err)
	}
}

func TestExtensionArtifactReadIsScopedPreflightAndRequiresInstallPermission(t *testing.T) {
	profile := Profile{
		ID: "profile", Effect: EffectAct, Approval: ApprovalNone, ProfileMode: "fresh",
		URLAllow: []string{"https://allowed.example/public/*"}, View: true, Control: true,
		ExtensionAccess: true, ExtensionInstall: true,
		ExtensionPermissions: map[string][]string{"abcdefghijklmnopabcdefghijklmnop": {"storage"}},
	}
	policy, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := ActionRequest{SessionID: "session", LeaseID: "lease", OwnerID: "owner", CallerID: "run:owner", Kind: "extension_artifact_read", CurrentURL: "https://allowed.example/public/page"}
	if err := policy.Authorize(context.Background(), request); err != nil {
		t.Fatalf("scoped artifact metadata resolution denied: %v", err)
	}
	request.CurrentURL = "https://outside.example/private"
	if err := policy.Authorize(context.Background(), request); err == nil {
		t.Fatal("artifact metadata lookup bypassed current URL scope")
	}
	profile.ExtensionInstall = false
	noInstall, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.CurrentURL = "https://allowed.example/public/page"
	if err := noInstall.Authorize(context.Background(), request); err == nil {
		t.Fatal("artifact metadata resolution was allowed without install permission")
	}
}
