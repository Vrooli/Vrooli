package memberflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MemberRef identifies a single team member.
type MemberRef struct {
	Team   string `json:"team"`
	Member string `json:"member"`
}

func (r MemberRef) String() string { return r.Team + "/" + r.Member }

// MemberTopics pairs a member identity with its declarations and the path
// where they live on disk. Path is the absolute path to the topics.json file
// (or empty when the file did not exist and was treated as `{}`).
type MemberTopics struct {
	Ref    MemberRef
	Topics Topics
	Path   string
	Exists bool
}

// LoadAll walks the store directory and returns every team member's
// declarations. Members without a topics.json get a populated MemberTopics
// entry with Exists=false and Topics=={}.
//
// storeDir is the absolute path to scenarios/prompt-manager/store/. It is
// expected to contain a teams/ subdirectory.
//
// If a topics.json is malformed (invalid JSON or fails Validate), the error is
// returned and any partial results are discarded — the caller gets either
// every member or an explicit failure.
func LoadAll(storeDir string) ([]MemberTopics, error) {
	teamsDir := filepath.Join(storeDir, "teams")
	teamEntries, err := os.ReadDir(teamsDir)
	if err != nil {
		return nil, fmt.Errorf("memberflow: read teams dir %q: %w", teamsDir, err)
	}

	var out []MemberTopics
	for _, te := range teamEntries {
		if !te.IsDir() || strings.HasPrefix(te.Name(), ".") {
			continue
		}
		team := te.Name()
		membersDir := filepath.Join(teamsDir, team, "members")
		memberEntries, err := os.ReadDir(membersDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("memberflow: read members dir %q: %w", membersDir, err)
		}
		for _, me := range memberEntries {
			if !me.IsDir() || strings.HasPrefix(me.Name(), ".") {
				continue
			}
			member := me.Name()
			path := filepath.Join(membersDir, member, "topics.json")
			mt, err := loadOne(team, member, path)
			if err != nil {
				return nil, err
			}
			out = append(out, mt)
		}
	}

	// Stable order: by team, then member.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ref.Team != out[j].Ref.Team {
			return out[i].Ref.Team < out[j].Ref.Team
		}
		return out[i].Ref.Member < out[j].Ref.Member
	})

	return out, nil
}

// LoadTeam returns declarations for every member of a single team. A
// non-existent team returns an empty slice with no error so that callers can
// distinguish "team has no members" from infrastructure failures.
func LoadTeam(storeDir, team string) ([]MemberTopics, error) {
	all, err := LoadAll(storeDir)
	if err != nil {
		return nil, err
	}
	var out []MemberTopics
	for _, m := range all {
		if m.Ref.Team == team {
			out = append(out, m)
		}
	}
	return out, nil
}

// LoadMember returns the declaration for one specific team member. Returns a
// MemberTopics with Exists=false (and IsEmpty()==true) when the file does not
// exist — that is a valid "no flow" declaration, not an error.
func LoadMember(storeDir, team, member string) (MemberTopics, error) {
	path := filepath.Join(storeDir, "teams", team, "members", member, "topics.json")
	return loadOne(team, member, path)
}

// WriteMember replaces the topics.json for one specific team member with the
// supplied content. Validates first; refuses to write malformed declarations.
// Creates the member directory if it does not exist.
func WriteMember(storeDir, team, member string, t Topics) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("memberflow: invalid topics for %s/%s: %w", team, member, err)
	}
	dir := filepath.Join(storeDir, "teams", team, "members", member)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("memberflow: ensure member dir %q: %w", dir, err)
	}
	path := filepath.Join(dir, "topics.json")
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("memberflow: marshal topics: %w", err)
	}
	// Trailing newline keeps editors and `git diff` happy.
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("memberflow: write %q: %w", path, err)
	}
	return nil
}

func loadOne(team, member, path string) (MemberTopics, error) {
	mt := MemberTopics{Ref: MemberRef{Team: team, Member: member}, Path: path}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return mt, nil // empty Topics, Exists=false
	}
	if err != nil {
		return MemberTopics{}, fmt.Errorf("memberflow: read %q: %w", path, err)
	}
	mt.Exists = true

	if len(strings.TrimSpace(string(data))) == 0 {
		// Empty file is the same as "{}" — a positive empty declaration.
		return mt, nil
	}
	if err := json.Unmarshal(data, &mt.Topics); err != nil {
		return MemberTopics{}, fmt.Errorf("memberflow: parse %q: %w", path, err)
	}
	if err := mt.Topics.Validate(); err != nil {
		return MemberTopics{}, fmt.Errorf("memberflow: invalid topics in %q: %w", path, err)
	}
	return mt, nil
}
