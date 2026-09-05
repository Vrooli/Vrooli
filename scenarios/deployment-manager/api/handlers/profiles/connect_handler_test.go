package profileshandler

import (
	"context"
	"errors"
	"testing"
	"time"

	profilesdomain "deployment-manager/profiles"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles"
)

type fakeProfileRepository struct {
	profiles map[string]*profilesdomain.Profile
	versions []profilesdomain.Version
	err      error
}

func (f *fakeProfileRepository) List(_ context.Context) ([]profilesdomain.Profile, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := make([]profilesdomain.Profile, 0, len(f.profiles))
	for _, profile := range f.profiles {
		result = append(result, *profile)
	}
	return result, nil
}

func (f *fakeProfileRepository) Get(_ context.Context, idOrName string) (*profilesdomain.Profile, error) {
	if f.err != nil {
		return nil, f.err
	}
	if profile, ok := f.profiles[idOrName]; ok {
		return profile, nil
	}
	for _, profile := range f.profiles {
		if profile.Name == idOrName {
			return profile, nil
		}
	}
	return nil, nil
}

func (f *fakeProfileRepository) Create(_ context.Context, profile *profilesdomain.Profile) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.profiles[profile.ID] = profile
	return profile.ID, nil
}

func (f *fakeProfileRepository) Update(_ context.Context, idOrName string, updates map[string]interface{}) (*profilesdomain.Profile, error) {
	if f.err != nil {
		return nil, f.err
	}
	profile, err := f.Get(context.Background(), idOrName)
	if err != nil || profile == nil {
		return profile, err
	}
	if value, ok := updates["tiers"]; ok {
		profile.Tiers = value
	}
	if value, ok := updates["name"].(string); ok {
		profile.Name = value
	}
	if value, ok := updates["scenario"].(string); ok {
		profile.Scenario = value
	}
	if value, ok := updates["swaps"]; ok {
		profile.Swaps = value
	}
	profile.Version++
	return profile, nil
}

func (f *fakeProfileRepository) Delete(_ context.Context, idOrName string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if _, ok := f.profiles[idOrName]; !ok {
		return false, nil
	}
	delete(f.profiles, idOrName)
	return true, nil
}

func (f *fakeProfileRepository) GetVersions(context.Context, string) ([]profilesdomain.Version, error) {
	return f.versions, f.err
}

func (f *fakeProfileRepository) GetScenarioAndTier(context.Context, string) (string, int, error) {
	return "", 0, profilesdomain.ErrNotFound
}

func (f *fakeProfileRepository) AddSwap(context.Context, string, profilesdomain.Swap) error {
	return f.err
}

func (f *fakeProfileRepository) GetSwaps(context.Context, string) ([]profilesdomain.Swap, error) {
	return nil, f.err
}

var _ profilesdomain.Repository = (*fakeProfileRepository)(nil)

func TestConnectProfilesCRUD(t *testing.T) {
	repo := &fakeProfileRepository{profiles: make(map[string]*profilesdomain.Profile)}
	handler := NewConnectHandler(repo)

	created, err := handler.CreateProfile(context.Background(), connect.NewRequest(&profilesv1.CreateProfileRequest{
		Name: "Desktop", Scenario: "demo", Tiers: []int32{1, 2},
	}))
	if err != nil || created.Msg.Profile == nil {
		t.Fatalf("create failed: response=%v error=%v", created, err)
	}
	profileID := created.Msg.Profile.Id
	if profileID == "" || created.Msg.Profile.Version != 1 {
		t.Fatalf("create returned incomplete profile: %+v", created.Msg.Profile)
	}

	listed, err := handler.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{}))
	if err != nil || len(listed.Msg.Profiles) != 1 || listed.Msg.Profiles[0].Id != profileID {
		t.Fatalf("list failed: response=%v error=%v", listed, err)
	}

	name := "Desktop Updated"
	updated, err := handler.UpdateProfile(context.Background(), connect.NewRequest(&profilesv1.UpdateProfileRequest{
		ProfileId: profileID, Name: &name, Tiers: []int32{2},
	}))
	if err != nil || updated.Msg.Profile == nil || updated.Msg.Profile.Version != 2 {
		t.Fatalf("update failed: response=%v error=%v", updated, err)
	}

	got, err := handler.GetProfile(context.Background(), connect.NewRequest(&profilesv1.GetProfileRequest{ProfileId: profileID}))
	if err != nil || got.Msg.Profile.Name != name || len(got.Msg.Profile.Tiers) != 1 {
		t.Fatalf("get failed: response=%v error=%v", got, err)
	}

	deleted, err := handler.DeleteProfile(context.Background(), connect.NewRequest(&profilesv1.DeleteProfileRequest{ProfileId: profileID}))
	if err != nil || deleted.Msg.ProfileId != profileID {
		t.Fatalf("delete failed: response=%v error=%v", deleted, err)
	}
	if _, err := handler.GetProfile(context.Background(), connect.NewRequest(&profilesv1.GetProfileRequest{ProfileId: profileID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestConnectProfilesValidationAndRepositoryErrors(t *testing.T) {
	handler := NewConnectHandler(&fakeProfileRepository{profiles: make(map[string]*profilesdomain.Profile), err: errors.New("database unavailable")})
	if _, err := handler.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("expected internal list error, got %v", err)
	}
	if _, err := handler.CreateProfile(context.Background(), connect.NewRequest(&profilesv1.CreateProfileRequest{Scenario: "demo"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected required-field validation, got %v", err)
	}
	if _, err := handler.GetProfile(context.Background(), nil); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected nil request validation, got %v", err)
	}
}

func TestProfileTimestampsRemainValid(t *testing.T) {
	profile := profileToProto(profilesdomain.Profile{ID: "p", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if profile.CreatedAt == nil || profile.UpdatedAt == nil {
		t.Fatal("expected timestamps in generated response")
	}
}

func TestConnectProfilesPaginationAndConversionBranches(t *testing.T) {
	now := time.Now()
	repo := &fakeProfileRepository{
		profiles: map[string]*profilesdomain.Profile{
			"p1": {ID: "p1", Name: "one", Scenario: "demo", Tiers: []interface{}{1}, CreatedAt: now, UpdatedAt: now},
			"p2": {ID: "p2", Name: "two", Scenario: "demo", Tiers: []interface{}{2}, CreatedAt: now, UpdatedAt: now},
		},
		versions: []profilesdomain.Version{{ProfileID: "p1", Version: 1, Name: "one", Scenario: "demo", Tiers: []interface{}{1}, CreatedAt: now}, {ProfileID: "p1", Version: 2, Name: "two", Scenario: "demo", Tiers: []interface{}{2}, CreatedAt: now}},
	}
	h := NewConnectHandler(repo)
	listed, err := h.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{PageSize: 1}))
	if err != nil || len(listed.Msg.Profiles) != 1 || listed.Msg.NextPageToken != "1" {
		t.Fatalf("paged list = %#v, %v", listed, err)
	}
	if _, err := h.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{PageToken: "1", PageSize: 1000})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ListProfiles(context.Background(), connect.NewRequest(&profilesv1.ListProfilesRequest{PageToken: "99"})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ListProfileVersions(context.Background(), connect.NewRequest(&profilesv1.ListProfileVersionsRequest{ProfileId: "p1", PageSize: 1})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ListProfileVersions(context.Background(), connect.NewRequest(&profilesv1.ListProfileVersionsRequest{ProfileId: "p1", PageToken: "1"})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.GetProfile(context.Background(), connect.NewRequest(&profilesv1.GetProfileRequest{ProfileId: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing profile error = %v", err)
	}
	if _, err := h.DeleteProfile(context.Background(), connect.NewRequest(&profilesv1.DeleteProfileRequest{ProfileId: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing delete error = %v", err)
	}

	name, scenario := "renamed", "new-demo"
	_, err = h.UpdateProfile(context.Background(), connect.NewRequest(&profilesv1.UpdateProfileRequest{
		ProfileId: "p1", Name: &name, Scenario: &scenario, Tiers: []int32{3},
		Swaps: &commonv1.JsonObject{}, Secrets: &commonv1.JsonObject{}, Settings: &commonv1.JsonObject{},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.UpdateProfile(context.Background(), connect.NewRequest(&profilesv1.UpdateProfileRequest{ProfileId: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing update error = %v", err)
	}
	if _, err := h.CreateProfile(context.Background(), nil); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("nil create error = %v", err)
	}

	for _, tc := range []struct {
		token string
		size  int32
	}{
		{"bad", 1}, {"", -1},
	} {
		if _, _, err := parsePage(tc.token, tc.size); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("parsePage(%q,%d) = %v", tc.token, tc.size, err)
		}
	}
	if offset, size, err := parsePage("2", 1000); err != nil || offset != 2 || size != 100 {
		t.Fatalf("page cap = %d,%d,%v", offset, size, err)
	}
	if valueToJSONObj(nil) != nil || jsonObjectToValue(nil) != nil {
		t.Fatal("nil JSON values should remain nil")
	}
	if got := jsonToInt32Slice([]interface{}{float64(1), int(2), int32(3), "ignored"}); len(got) != 3 {
		t.Fatalf("converted tiers = %#v", got)
	}
}
