package search

import (
	"context"
	"errors"
	"testing"

	"prompt-manager/store"
)

type mockTeamStore struct {
	teams []store.Team
}

func (m *mockTeamStore) List(ctx context.Context) ([]store.Team, error) {
	return m.teams, nil
}

func (m *mockTeamStore) Get(ctx context.Context, id string) (*store.Team, error) {
	for _, t := range m.teams {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockTeamStore) Create(ctx context.Context, team *store.Team) error {
	return nil
}

func (m *mockTeamStore) Update(ctx context.Context, id string, updates *store.Team) error {
	return nil
}

func (m *mockTeamStore) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockTeamStore) GetRoles(ctx context.Context, id string) (*store.TeamRoles, error) {
	return &store.TeamRoles{}, nil
}

func (m *mockTeamStore) SetRoles(ctx context.Context, id string, roles *store.TeamRoles) error {
	return nil
}

func (m *mockTeamStore) GetOrgChart(ctx context.Context, teamID string) (*store.OrgChart, error) {
	return &store.OrgChart{}, nil
}

func (m *mockTeamStore) SetOrgChart(ctx context.Context, teamID string, org *store.OrgChart) error {
	return nil
}

func (m *mockTeamStore) GetMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	return nil, nil
}

func (m *mockTeamStore) GetInbox(ctx context.Context, teamID, agentID string) (*store.TeamInbox, error) {
	return &store.TeamInbox{}, nil
}

func (m *mockTeamStore) SetInbox(ctx context.Context, teamID, agentID string, inbox *store.TeamInbox) error {
	return nil
}

type mockRelationStore struct {
	members map[string][]store.TeamMemberRelation // key: teamID
}

func (m *mockRelationStore) SetTeamMember(ctx context.Context, rel *store.TeamMemberRelation) error {
	return nil
}

func (m *mockRelationStore) GetTeamMember(ctx context.Context, teamID, agentID string) (*store.TeamMemberRelation, error) {
	return nil, errors.New("not found")
}

func (m *mockRelationStore) DeleteTeamMember(ctx context.Context, teamID, agentID string) error {
	return nil
}

func (m *mockRelationStore) ListTeamMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error) {
	return m.members[teamID], nil
}

func (m *mockRelationStore) ListAgentTeams(ctx context.Context, agentID string) ([]store.TeamMemberRelation, error) {
	return nil, nil
}

type mockTeamFileReader struct {
	files    map[string][]store.TeamFileEntry
	contents map[string]string // key: "teamID/path"
}

func (m *mockTeamFileReader) ListSharedFiles(ctx context.Context, id string) ([]store.TeamFileEntry, error) {
	files, ok := m.files[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return files, nil
}

func (m *mockTeamFileReader) ReadSharedFile(ctx context.Context, id, path string) (string, error) {
	key := id + "/" + path
	content, ok := m.contents[key]
	if !ok {
		return "", errors.New("not found")
	}
	return content, nil
}

func newTestTeamService(teams []store.Team, members map[string][]store.TeamMemberRelation, files map[string][]store.TeamFileEntry, contents map[string]string) *TeamSearchService {
	relStore := &mockRelationStore{members: members}
	if members == nil {
		relStore.members = map[string][]store.TeamMemberRelation{}
	}
	return NewTeamSearchService(
		&mockTeamStore{teams: teams},
		relStore,
		&mockTeamFileReader{files: files, contents: contents},
	)
}

func TestTeamSearch_ExactNameMatch(t *testing.T) {
	svc := newTestTeamService([]store.Team{
		{ID: "eng", DisplayName: "Engineering", Enabled: true},
		{ID: "design", DisplayName: "Design", Enabled: true},
	}, nil, nil, nil)

	resp, err := svc.Search(context.Background(), TeamSearchQuery{Query: "Engineering"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total == 0 {
		t.Fatal("expected at least one result")
	}
	if resp.Results[0].ID != "eng" {
		t.Errorf("expected eng, got %s", resp.Results[0].ID)
	}
	if resp.Results[0].Score < 10.0 {
		t.Errorf("expected exact name match score >= 10, got %f", resp.Results[0].Score)
	}
}

func TestTeamSearch_MissionMatch(t *testing.T) {
	svc := newTestTeamService([]store.Team{
		{ID: "t1", DisplayName: "Alpha", Mission: "Build machine learning pipelines", Enabled: true},
		{ID: "t2", DisplayName: "Beta", Mission: "Handle customer support", Enabled: true},
	}, nil, nil, nil)

	resp, err := svc.Search(context.Background(), TeamSearchQuery{Query: "machine learning"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total == 0 {
		t.Fatal("expected at least one result")
	}
	if resp.Results[0].ID != "t1" {
		t.Errorf("expected t1, got %s", resp.Results[0].ID)
	}
}

func TestTeamSearch_FilterByEnabled(t *testing.T) {
	svc := newTestTeamService([]store.Team{
		{ID: "t1", DisplayName: "Active Team", Enabled: true},
		{ID: "t2", DisplayName: "Disabled Team", Enabled: false},
	}, nil, nil, nil)

	enabled := true
	resp, err := svc.Search(context.Background(), TeamSearchQuery{Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 result, got %d", resp.Total)
	}
	if resp.Results[0].ID != "t1" {
		t.Errorf("expected t1, got %s", resp.Results[0].ID)
	}
}

func TestTeamSearch_NoQuery_ReturnsAll(t *testing.T) {
	svc := newTestTeamService([]store.Team{
		{ID: "t1", DisplayName: "Team A", Enabled: true},
		{ID: "t2", DisplayName: "Team B", Enabled: true},
	}, nil, nil, nil)

	resp, err := svc.Search(context.Background(), TeamSearchQuery{})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 2 {
		t.Fatalf("expected 2 results, got %d", resp.Total)
	}
}

func TestTeamSearch_MemberCountIncluded(t *testing.T) {
	svc := newTestTeamService(
		[]store.Team{
			{ID: "t1", DisplayName: "Team One", Enabled: true},
		},
		map[string][]store.TeamMemberRelation{
			"t1": {
				{TeamID: "t1", AgentID: "a1"},
				{TeamID: "t1", AgentID: "a2"},
			},
		},
		nil, nil,
	)

	resp, err := svc.Search(context.Background(), TeamSearchQuery{})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 result, got %d", resp.Total)
	}
	if resp.Results[0].MemberCount != 2 {
		t.Errorf("expected 2 members, got %d", resp.Results[0].MemberCount)
	}
}

func TestTeamContentSearch_Basic(t *testing.T) {
	svc := newTestTeamService(
		[]store.Team{
			{ID: "t1", DisplayName: "Team One", Enabled: true},
		},
		nil,
		map[string][]store.TeamFileEntry{
			"t1": {
				{Path: "README.md", IsDir: false, Size: 50},
			},
		},
		map[string]string{
			"t1/README.md": "Line one\nContains target text\nLine three",
		},
	)

	resp, err := svc.SearchContent(context.Background(), TeamContentSearchQuery{
		Query: "target",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 match, got %d", resp.Total)
	}
	if resp.Matches[0].LineNumber != 2 {
		t.Errorf("expected line 2, got %d", resp.Matches[0].LineNumber)
	}
	if resp.Matches[0].TeamID != "t1" {
		t.Errorf("expected team t1, got %s", resp.Matches[0].TeamID)
	}
}

func TestTeamContentSearch_EmptyQuery(t *testing.T) {
	svc := newTestTeamService(nil, nil, nil, nil)

	resp, err := svc.SearchContent(context.Background(), TeamContentSearchQuery{
		Query: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 0 {
		t.Errorf("expected 0 matches for empty query, got %d", resp.Total)
	}
}

func TestTeamContentSearch_Limit(t *testing.T) {
	svc := newTestTeamService(
		[]store.Team{
			{ID: "t1", DisplayName: "Team", Enabled: true},
		},
		nil,
		map[string][]store.TeamFileEntry{
			"t1": {{Path: "data.md", IsDir: false}},
		},
		map[string]string{
			"t1/data.md": "match\nmatch\nmatch\nmatch\nmatch",
		},
	)

	resp, err := svc.SearchContent(context.Background(), TeamContentSearchQuery{
		Query: "match",
		Limit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 3 {
		t.Errorf("expected 3 matches (limited), got %d", resp.Total)
	}
}
