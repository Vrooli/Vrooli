package profiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles/profilesv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// connectCommands is the canonical profile CLI surface. It deliberately uses
// generated request/response messages; the older APIClient-backed commands
// remain available only for profile subcommands not yet represented by the
// v1 ProfilesService contract.
type connectCommands struct {
	client profilesconnect.ProfilesServiceClient
}

func newConnectCommands(app *cliapp.ScenarioApp) *connectCommands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	return &connectCommands{client: profilesconnect.NewProfilesServiceClient(httpClient, baseURL)}
}

func (c *connectCommands) list(_ []string) error {
	response, err := c.client.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{PageSize: 100}))
	if err != nil {
		return cliapp.WrapAPIError("list profiles", err, nil)
	}
	return writeJSON(response.Msg)
}

func (c *connectCommands) create(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: profile create <name> <scenario> [--tier <tier>]")
	}
	tier := 2
	for i := 2; i+1 < len(args); i++ {
		if args[i] == "--tier" {
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid tier: %w", err)
			}
			tier = parsed
		}
	}
	response, err := c.client.CreateProfile(context.Background(), connect.NewRequest(&profilesv1.CreateProfileRequest{
		Name: args[0], Scenario: args[1], Tiers: []int32{int32(tier)},
	}))
	if err != nil {
		return cliapp.WrapAPIError("create profile", err, nil)
	}
	return writeJSON(response.Msg)
}

func (c *connectCommands) show(args []string) error {
	if len(args) == 0 {
		return errors.New("profile id is required")
	}
	response, err := c.client.GetProfile(context.Background(), connect.NewRequest(&profilesv1.GetProfileRequest{ProfileId: args[0]}))
	if err != nil {
		return cliapp.WrapAPIError("get profile", err, nil)
	}
	return writeJSON(response.Msg)
}

func (c *connectCommands) delete(args []string) error {
	if len(args) == 0 {
		return errors.New("profile id is required")
	}
	response, err := c.client.DeleteProfile(context.Background(), connect.NewRequest(&profilesv1.DeleteProfileRequest{ProfileId: args[0]}))
	if err != nil {
		return cliapp.WrapAPIError("delete profile", err, nil)
	}
	return writeJSON(response.Msg)
}

func (c *connectCommands) versions(args []string) error {
	if len(args) == 0 {
		return errors.New("profile id is required")
	}
	response, err := c.client.ListProfileVersions(context.Background(), connect.NewRequest(&profilesv1.ListProfileVersionsRequest{ProfileId: args[0], PageSize: 100}))
	if err != nil {
		return cliapp.WrapAPIError("list profile versions", err, nil)
	}
	return writeJSON(response.Msg)
}

func (c *connectCommands) listCall(_ cliapp.OperationContext) (*profilesv1.ListProfilesResponse, error) {
	response, err := c.client.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{PageSize: 100}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list profiles", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no profiles response")
	}
	return response.Msg, nil
}

func (c *connectCommands) listReport(_ cliapp.OperationContext, response *profilesv1.ListProfilesResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetProfiles()))
	for _, profile := range response.GetProfiles() {
		results = append(results, fmt.Sprintf("%s — %s (%s)", profile.GetId(), profile.GetName(), profile.GetScenario()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d profile(s).", len(response.GetProfiles()))}, ResultsHeading: "Profiles", Results: results}
}

func (c *connectCommands) createCall(ctx cliapp.OperationContext) (*profilesv1.CreateProfileResponse, error) {
	tier, err := parseTier(ctx.Flag("tier"))
	if err != nil {
		return nil, err
	}
	response, err := c.client.CreateProfile(context.Background(), connect.NewRequest(&profilesv1.CreateProfileRequest{
		Name: ctx.Positional("name"), Scenario: ctx.Positional("scenario"), Tiers: []int32{tier},
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create profile", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no created profile")
	}
	return response.Msg, nil
}

func (c *connectCommands) createReport(_ cliapp.OperationContext, response *profilesv1.CreateProfileResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created profile %s.", response.GetProfile().GetId())}}
}

func (c *connectCommands) showCall(ctx cliapp.OperationContext) (*profilesv1.GetProfileResponse, error) {
	response, err := c.client.GetProfile(context.Background(), connect.NewRequest(&profilesv1.GetProfileRequest{ProfileId: ctx.Positional("profile_id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get profile", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no profile")
	}
	return response.Msg, nil
}

func (c *connectCommands) showReport(_ cliapp.OperationContext, response *profilesv1.GetProfileResponse) cliapp.ListReport {
	profile := response.GetProfile()
	if profile == nil {
		return cliapp.ListReport{Summary: []string{"Profile not found."}}
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Profile %s — %s (%s).", profile.GetId(), profile.GetName(), profile.GetScenario())}}
}

func (c *connectCommands) updateCall(ctx cliapp.OperationContext) (*profilesv1.UpdateProfileResponse, error) {
	request := &profilesv1.UpdateProfileRequest{ProfileId: ctx.Positional("profile_id")}
	if ctx.FlagProvided("name") {
		name := ctx.Flag("name")
		request.Name = &name
	}
	if ctx.FlagProvided("scenario") {
		scenario := ctx.Flag("scenario")
		request.Scenario = &scenario
	}
	if ctx.FlagProvided("tier") {
		tier, err := parseTier(ctx.Flag("tier"))
		if err != nil {
			return nil, err
		}
		request.Tiers = []int32{tier}
	}
	if request.Name == nil && request.Scenario == nil && len(request.Tiers) == 0 {
		return nil, fmt.Errorf("at least one update flag is required")
	}
	response, err := c.client.UpdateProfile(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("update profile", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no updated profile")
	}
	return response.Msg, nil
}

func (c *connectCommands) updateReport(_ cliapp.OperationContext, response *profilesv1.UpdateProfileResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Updated profile %s.", response.GetProfile().GetId())}}
}

func (c *connectCommands) deleteCall(ctx cliapp.OperationContext) (*profilesv1.DeleteProfileResponse, error) {
	response, err := c.client.DeleteProfile(context.Background(), connect.NewRequest(&profilesv1.DeleteProfileRequest{ProfileId: ctx.Positional("profile_id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("delete profile", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no delete response")
	}
	return response.Msg, nil
}

func (c *connectCommands) deleteReport(_ cliapp.OperationContext, response *profilesv1.DeleteProfileResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Deleted profile %s.", response.GetProfileId())}}
}

func (c *connectCommands) versionsCall(ctx cliapp.OperationContext) (*profilesv1.ListProfileVersionsResponse, error) {
	response, err := c.client.ListProfileVersions(context.Background(), connect.NewRequest(&profilesv1.ListProfileVersionsRequest{ProfileId: ctx.Positional("profile_id"), PageSize: 100}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list profile versions", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no profile versions response")
	}
	return response.Msg, nil
}

func (c *connectCommands) versionsReport(_ cliapp.OperationContext, response *profilesv1.ListProfileVersionsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetVersions()))
	for _, version := range response.GetVersions() {
		results = append(results, fmt.Sprintf("v%d — %s (%s)", version.GetVersion(), version.GetName(), version.GetScenario()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d profile version(s).", len(response.GetVersions()))}, ResultsHeading: "Versions", Results: results}
}

func parseTier(value string) (int32, error) {
	tier, err := strconv.ParseInt(value, 10, 32)
	if err != nil || tier < 0 {
		return 0, fmt.Errorf("invalid tier %q", value)
	}
	return int32(tier), nil
}

func writeJSON(value interface{ ProtoReflect() protoreflect.Message }) error {
	data, err := protojson.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	_, err = os.Stdout.Write(append(data, '\n'))
	return err
}
