package administration

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/sessions"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"golang.org/x/crypto/bcrypt"
	"landing-page-business-suite-api/internal/administration"
)

func TestValidateProfilePasswordRejectsConfiguredDefault(t *testing.T) {
	currentHash, err := bcrypt.GenerateFromPassword([]byte("current-password-123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateProfilePassword("default-password-123", string(currentHash), "default-password-123"); err == nil {
		t.Fatal("expected configured default password to be rejected")
	}
}

func TestProfileConnectReturnsDisplaySafeProfile(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("not-the-default-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	manager := &fakeSessions{session: sessions.NewSession(nil, sessionName)}
	manager.session.Values = map[any]any{"email": "admin@example.test"}
	handler := NewProfileConnectHandler(profileDependencies(&fakeProfileService{profile: administration.AdminProfile{ID: 7, Email: "admin@example.test", PasswordHash: string(hash)}}, manager))

	response, err := handler.GetAdminProfile(context.Background(), connect.NewRequest(&lpbsv1.GetAdminProfileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetProfile(); got.GetEmail() != "admin@example.test" || got.GetIsDefaultEmail() || got.GetIsDefaultPassword() {
		t.Fatalf("profile = %+v", got)
	}
}

func TestProfileConnectUpdateRequiresCurrentPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("current-password-123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	manager := &fakeSessions{session: sessions.NewSession(nil, sessionName)}
	manager.session.Values = map[any]any{"email": "admin@example.test"}
	handler := NewProfileConnectHandler(profileDependencies(&fakeProfileService{profile: administration.AdminProfile{ID: 7, Email: "admin@example.test", PasswordHash: string(hash)}}, manager))

	_, err = handler.UpdateAdminProfile(context.Background(), connect.NewRequest(&lpbsv1.UpdateAdminProfileRequest{NewEmail: "changed@example.test"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("UpdateAdminProfile() error code = %v, want invalid argument", connect.CodeOf(err))
	}
}

type fakeProfileService struct{ profile administration.AdminProfile }

func (f *fakeProfileService) Profile(context.Context, string) (administration.AdminProfile, error) {
	return f.profile, nil
}

func (*fakeProfileService) EmailInUse(context.Context, string, int64) (bool, error) {
	return false, nil
}
func (*fakeProfileService) UpdateProfile(context.Context, int64, string, string) error { return nil }
func (*fakeProfileService) RevokeOtherSessions(context.Context, string, string) (int64, error) {
	return 0, nil
}

func profileDependencies(service ProfileService, manager SessionManager) ProfileDependencies {
	return ProfileDependencies{
		Auth: service, Sessions: manager,
		DefaultEmail:    func() string { return "default@example.test" },
		DefaultPassword: func() string { return "" },
		ValidateEmail:   func(string) error { return nil },
		Log:             func(string, map[string]any) {}, LogError: func(string, map[string]any) {},
	}
}
