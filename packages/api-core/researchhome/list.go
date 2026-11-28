package researchhome

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Kind is what a research item is.
type Kind string

const (
	KindSkill   Kind = "skill"
	KindProgram Kind = "program"
)

// maxItems bounds one scope's listing per kind.
const maxItems = 200

// Item is one skill or program found under a scope, with every problem that
// stops it resolving or running. An empty Problems means it is usable.
type Item struct {
	ID       string
	Kind     Kind
	Path     string // SKILL.md or the program contract, unresolved
	Problems []string
}

// Scope is a folder that may hold skills/ and programs/: the research home
// (Topic "") or one topic folder.
type Scope struct {
	Topic string
	Dir   string
}

// Scopes lists the research home and every topic folder. A topic folder that
// is a symbolic link is an alias and is skipped, so nothing is listed twice;
// its items still resolve by id through the link.
func Scopes(home string) ([]Scope, error) {
	scopes := []Scope{{Dir: home}}
	entries, err := os.ReadDir(filepath.Join(home, "topics"))
	if IsNotExist(err) {
		return scopes, nil
	}
	if err != nil {
		return scopes, err
	}
	for _, entry := range entries {
		if entry.IsDir() && topicPattern.MatchString(entry.Name()) {
			scopes = append(scopes, Scope{Topic: entry.Name(), Dir: filepath.Join(home, "topics", entry.Name())})
		}
	}
	return scopes, nil
}

// List returns the scope's skills then programs, sorted by id. A missing
// skills/ or programs/ folder lists nothing; an unreadable one is an error.
func List(home, topic string) ([]Item, error) {
	ref := Ref{Topic: topic}
	if topic != "" && !topicPattern.MatchString(topic) {
		return nil, fmt.Errorf("%w: topic %q is not a topic folder name", ErrInvalid, topic)
	}
	scope := ref.ScopeDir(home)
	skills, err := listSkills(home, ref, filepath.Join(scope, "skills"))
	if err != nil {
		return nil, err
	}
	programs, err := listPrograms(home, ref, filepath.Join(scope, "programs"))
	if err != nil {
		return nil, err
	}
	return append(skills, programs...), nil
}

func listSkills(home string, scope Ref, dir string) ([]Item, error) {
	entries, err := readDir(dir)
	if err != nil {
		return nil, err
	}
	var items []Item
	for _, entry := range entries {
		if len(items) == maxItems {
			break
		}
		ref := Ref{Topic: scope.Topic, Name: entry.Name()}
		item := Item{ID: ref.ID(), Kind: KindSkill, Path: ref.SkillPath(home)}
		if err := ValidName(entry.Name()); err != nil {
			item.Problems = append(item.Problems, err.Error())
		}
		if real, err := Resolve(home, filepath.Join(dir, entry.Name())); err != nil {
			item.Problems = append(item.Problems, err.Error())
		} else if info, err := os.Stat(real); err != nil || !info.IsDir() {
			item.Problems = append(item.Problems, "not a skill folder; a skill is skills/<name>/SKILL.md")
		} else {
			item.Problems = append(item.Problems, skillProblems(home, item.Path, entry.Name())...)
		}
		items = append(items, item)
	}
	return items, nil
}

func skillProblems(home, path, name string) []string {
	real, err := Resolve(home, path)
	if IsNotExist(err) {
		return []string{"SKILL.md is missing"}
	}
	if err != nil {
		return []string{err.Error()}
	}
	content, err := os.ReadFile(real)
	if err != nil {
		return []string{err.Error()}
	}
	front := frontmatter(content)
	var problems []string
	switch got := front["name"]; {
	case got == "":
		problems = append(problems, "SKILL.md frontmatter has no name")
	case got != name:
		problems = append(problems, fmt.Sprintf("SKILL.md name %q must match its folder %q", got, name))
	}
	if front["description"] == "" {
		problems = append(problems, "SKILL.md frontmatter has no description")
	}
	return problems
}

// frontmatter reads the top-level scalar keys of a leading --- block.
func frontmatter(content []byte) map[string]string {
	out := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return out
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		if key, value, ok := strings.Cut(line, ":"); ok {
			out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return out
}

func listPrograms(home string, scope Ref, dir string) ([]Item, error) {
	entries, err := readDir(dir)
	if err != nil {
		return nil, err
	}
	contracts := map[string]bool{}
	var items []Item
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".output.schema.json") {
			continue
		}
		if len(items) == maxItems {
			break
		}
		stem := strings.TrimSuffix(name, ".json")
		contracts[stem] = true
		ref := Ref{Topic: scope.Topic, Name: stem}
		item := Item{ID: ref.ID(), Kind: KindProgram, Path: ref.ProgramPath(home)}
		if err := ValidName(stem); err != nil {
			item.Problems = append(item.Problems, err.Error())
		}
		item.Problems = append(item.Problems, programProblems(home, ref)...)
		items = append(items, item)
	}
	for _, entry := range entries {
		stem, isSource := strings.CutSuffix(entry.Name(), ".py")
		if isSource && !entry.IsDir() && !contracts[stem] && len(items) < maxItems {
			ref := Ref{Topic: scope.Topic, Name: stem}
			items = append(items, Item{ID: ref.ID(), Kind: KindProgram, Path: ref.ProgramPath(home), Problems: []string{"program source has no " + stem + ".json contract"}})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func programProblems(home string, ref Ref) []string {
	program, err := ReadProgram(home, ref.ID())
	if err != nil && program.Contract.Path == "" {
		return []string{err.Error()}
	}
	var problems []string
	if err != nil {
		problems = append(problems, err.Error())
	}
	var contract struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(program.Contract.Content, &contract); err != nil {
		return append(problems, "contract is not valid JSON: "+err.Error())
	}
	if contract.Name != ref.ID() {
		problems = append(problems, fmt.Sprintf("contract name %q must be %q", contract.Name, ref.ID()))
	}
	return problems
}

// readDir lists visible entries; a missing folder has none.
func readDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	visible := entries[:0]
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") && !strings.HasPrefix(entry.Name(), "_") {
			visible = append(visible, entry)
		}
	}
	return visible, nil
}
