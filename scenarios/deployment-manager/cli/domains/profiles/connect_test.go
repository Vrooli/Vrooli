package profiles

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles/profilesv1connect"
)

type profileConnectService struct{}

func (profileConnectService) ListProfiles(context.Context, *connect.Request[profilesv1.ListProfilesRequest]) (*connect.Response[profilesv1.ListProfilesResponse], error) {
	return connect.NewResponse(&profilesv1.ListProfilesResponse{Profiles: []*profilesv1.Profile{{Id: "p1", Name: "demo", Scenario: "example"}}}), nil
}
func (profileConnectService) GetProfile(context.Context, *connect.Request[profilesv1.GetProfileRequest]) (*connect.Response[profilesv1.GetProfileResponse], error) {
	return connect.NewResponse(&profilesv1.GetProfileResponse{Profile: &profilesv1.Profile{Id: "p1", Name: "demo", Scenario: "example"}}), nil
}
func (profileConnectService) CreateProfile(_ context.Context, request *connect.Request[profilesv1.CreateProfileRequest]) (*connect.Response[profilesv1.CreateProfileResponse], error) {
	return connect.NewResponse(&profilesv1.CreateProfileResponse{Profile: &profilesv1.Profile{Id: "created", Name: request.Msg.GetName(), Scenario: request.Msg.GetScenario(), Tiers: request.Msg.GetTiers()}}), nil
}
func (profileConnectService) UpdateProfile(_ context.Context, request *connect.Request[profilesv1.UpdateProfileRequest]) (*connect.Response[profilesv1.UpdateProfileResponse], error) {
	name := "updated"
	if request.Msg.Name != nil {
		name = request.Msg.GetName()
	}
	return connect.NewResponse(&profilesv1.UpdateProfileResponse{Profile: &profilesv1.Profile{Id: request.Msg.GetProfileId(), Name: name, Scenario: request.Msg.GetScenario()}}), nil
}
func (profileConnectService) DeleteProfile(_ context.Context, request *connect.Request[profilesv1.DeleteProfileRequest]) (*connect.Response[profilesv1.DeleteProfileResponse], error) {
	return connect.NewResponse(&profilesv1.DeleteProfileResponse{ProfileId: request.Msg.GetProfileId()}), nil
}
func (profileConnectService) ListProfileVersions(context.Context, *connect.Request[profilesv1.ListProfileVersionsRequest]) (*connect.Response[profilesv1.ListProfileVersionsResponse], error) {
	return connect.NewResponse(&profilesv1.ListProfileVersionsResponse{Versions: []*profilesv1.ProfileVersion{{Version: 1, Name: "old", Scenario: "example"}}}), nil
}

func TestConnectCommandsUseGeneratedProfilesService(t *testing.T) {
	_, handler := profilesconnect.NewProfilesServiceHandler(profileConnectService{})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := profilesconnect.NewProfilesServiceClient(server.Client(), server.URL)
	commands := &connectCommands{client: client}
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"list", func() error { return commands.list(nil) }},
		{"create", func() error { return commands.create([]string{"demo", "example", "--tier", "3"}) }},
		{"show", func() error { return commands.show([]string{"p1"}) }},
		{"delete", func() error { return commands.delete([]string{"p1"}) }},
		{"versions", func() error { return commands.versions([]string{"p1"}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
		})
	}

	ctx := fakeProfileOperationContext{
		flags:       map[string]string{"tier": "4"},
		provided:    map[string]bool{"tier": true, "name": true},
		positionals: map[string]string{"name": "demo", "scenario": "example", "profile_id": "p1"},
	}
	if _, err := commands.listCall(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.createCall(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.showCall(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.updateCall(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.deleteCall(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.versionsCall(ctx); err != nil {
		t.Fatal(err)
	}
	_ = commands.listReport(ctx, &profilesv1.ListProfilesResponse{Profiles: []*profilesv1.Profile{{Id: "p1", Name: "demo", Scenario: "example"}}})
	_ = commands.createReport(ctx, &profilesv1.CreateProfileResponse{Profile: &profilesv1.Profile{Id: "p1"}})
	_ = commands.showReport(ctx, &profilesv1.GetProfileResponse{Profile: &profilesv1.Profile{Id: "p1", Name: "demo", Scenario: "example"}})
	_ = commands.showReport(ctx, &profilesv1.GetProfileResponse{})
	_ = commands.updateReport(ctx, &profilesv1.UpdateProfileResponse{Profile: &profilesv1.Profile{Id: "p1"}})
	_ = commands.deleteReport(ctx, &profilesv1.DeleteProfileResponse{ProfileId: "p1"})
	_ = commands.versionsReport(ctx, &profilesv1.ListProfileVersionsResponse{Versions: []*profilesv1.ProfileVersion{{Version: 1, Name: "old", Scenario: "example"}}})

	if _, err := parseTier("-1"); err == nil {
		t.Fatal("negative tier should fail")
	}
	if got, err := parseTier("3"); err != nil || got != 3 {
		t.Fatalf("parse tier = %d, %v", got, err)
	}
}

type fakeProfileOperationContext struct {
	flags       map[string]string
	provided    map[string]bool
	positionals map[string]string
}

func (f fakeProfileOperationContext) Flag(name string) string         { return f.flags[name] }
func (f fakeProfileOperationContext) FlagValues(name string) []string { return []string{f.flags[name]} }
func (f fakeProfileOperationContext) BoolFlag(string) bool            { return false }
func (f fakeProfileOperationContext) Positional(name string) string   { return f.positionals[name] }
func (f fakeProfileOperationContext) Positionals(name string) []string {
	return []string{f.positionals[name]}
}
func (f fakeProfileOperationContext) Args() []string                       { return nil }
func (f fakeProfileOperationContext) FlagBindings() []cliapp.FlagBindEntry { return nil }
func (f fakeProfileOperationContext) FlagDeclared(name string) bool {
	_, ok := f.flags[name]
	return ok
}
func (f fakeProfileOperationContext) FlagProvided(name string) bool { return f.provided[name] }
func (f fakeProfileOperationContext) Core() *cliapp.ScenarioApp     { return nil }
