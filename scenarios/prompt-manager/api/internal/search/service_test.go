package search

import (
	"errors"
	"testing"

	"prompt-manager/internal/skills"
)

type mockSkillStore struct {
	skills   []skills.Metadata
	contents map[string]string
}

func (m *mockSkillStore) GetAll() ([]skills.Metadata, error) {
	return m.skills, nil
}

func (m *mockSkillStore) FindByID(id string) (*skills.Metadata, string, error) {
	return nil, "", errors.New("not implemented")
}

func (m *mockSkillStore) LoadMetadata(folder string) ([]skills.Metadata, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSkillStore) SaveMetadata(folder string, skills []skills.Metadata) error {
	return errors.New("not implemented")
}

func (m *mockSkillStore) GetContent(folder, filename string) (string, error) {
	key := filename
	if folder != "" {
		key = folder + "/" + filename
	}
	content, ok := m.contents[key]
	if !ok {
		return "", errors.New("content not found")
	}
	return content, nil
}

func (m *mockSkillStore) SaveContent(folder, filename, content string) error {
	return errors.New("not implemented")
}

func (m *mockSkillStore) DeleteContent(folder, filename string) error {
	return errors.New("not implemented")
}

func (m *mockSkillStore) GetVersions(skillID string) ([]skills.SkillVersion, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSkillStore) SaveVersion(skillID, folder string, skill *skills.Metadata, content string) error {
	return errors.New("not implemented")
}

func (m *mockSkillStore) GetVersionContent(skillID string, version int) (*skills.SkillVersion, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSkillStore) LoadVersions(folder string) (map[string]*skills.VersionFile, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSkillStore) SaveVersions(folder string, versions map[string]*skills.VersionFile) error {
	return errors.New("not implemented")
}

func (m *mockSkillStore) Rename(oldID, newID string) (*skills.Metadata, error) {
	return nil, errors.New("not implemented")
}

func TestSearch_UsesMetadataOnly(t *testing.T) {
	store := &mockSkillStore{
		skills: []skills.Metadata{
			{
				ID:          "body-only-skill",
				Name:        "Body Only Skill",
				Description: "No visible query here",
				File:        "core/body-only-skill.md",
				Modes:       []string{"authoring"},
			},
			{
				ID:          "mode-skill",
				Name:        "Mode Skill",
				Description: "Another unrelated description",
				File:        "local/mode-skill.md",
				Modes:       []string{"testing"},
			},
		},
		contents: map[string]string{
			"core/body-only-skill.md": "This body mentions regression-only text.",
			"local/mode-skill.md":     "No matching body.",
		},
	}

	service := NewService(store)

	bodyOnlyResp, err := service.Search(SearchQuery{Query: "regression-only"})
	if err != nil {
		t.Fatalf("unexpected body-only search error: %v", err)
	}
	if bodyOnlyResp.Total != 0 {
		t.Fatalf("body-only quick search results = %+v, want none", bodyOnlyResp.Results)
	}

	modeResp, err := service.Search(SearchQuery{Query: "testing"})
	if err != nil {
		t.Fatalf("unexpected mode search error: %v", err)
	}
	if modeResp.Total != 1 || modeResp.Results[0].ID != "mode-skill" {
		t.Fatalf("mode search results = %+v, want mode-skill", modeResp.Results)
	}
}

func TestSearchContent_CaseInsensitive(t *testing.T) {
	store := &mockSkillStore{
		skills: []skills.Metadata{
			{
				ID:          "alpha-skill",
				Name:        "Alpha Skill",
				Description: "Alpha description",
				File:        "core/alpha-skill.md",
				Tags:        []string{"test"},
			},
		},
		contents: map[string]string{
			"core/alpha-skill.md": "Alpha beta\ngamma",
		},
	}

	service := NewService(store)
	resp, err := service.SearchContent(ContentSearchQuery{
		Query: "alpha",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 match, got %d", resp.Total)
	}
	match := resp.Matches[0]
	if match.LineNumber != 1 {
		t.Errorf("expected line 1, got %d", match.LineNumber)
	}
	if len(match.MatchRanges) != 1 || match.MatchRanges[0].Start != 0 || match.MatchRanges[0].End != 5 {
		t.Errorf("unexpected match ranges: %+v", match.MatchRanges)
	}
}

func TestSearchContent_WholeWord(t *testing.T) {
	store := &mockSkillStore{
		skills: []skills.Metadata{
			{
				ID:          "cat-skill",
				Name:        "Cat Skill",
				Description: "Cat description",
				File:        "local/cat-skill.md",
				Tags:        []string{"animal"},
			},
		},
		contents: map[string]string{
			"local/cat-skill.md": "concatenate cat catapult",
		},
	}

	service := NewService(store)
	resp, err := service.SearchContent(ContentSearchQuery{
		Query:     "cat",
		WholeWord: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 match, got %d", resp.Total)
	}
	match := resp.Matches[0]
	if match.LineNumber != 1 {
		t.Errorf("expected line 1, got %d", match.LineNumber)
	}
	if len(match.MatchRanges) != 1 {
		t.Fatalf("expected 1 match range, got %d", len(match.MatchRanges))
	}
	if match.MatchRanges[0].Start != 12 || match.MatchRanges[0].End != 15 {
		t.Errorf("unexpected match range: %+v", match.MatchRanges[0])
	}
}

func TestSearchContent_InvalidRegex(t *testing.T) {
	store := &mockSkillStore{}
	service := NewService(store)

	_, err := service.SearchContent(ContentSearchQuery{
		Query: "([",
		Regex: true,
	})
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
	if !errors.Is(err, ErrInvalidPattern) {
		t.Fatalf("expected ErrInvalidPattern, got %v", err)
	}
}
