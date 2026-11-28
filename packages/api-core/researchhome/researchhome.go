// Package researchhome resolves research-scoped skills and programs by their
// prefix id. The research home is $VROOLI_RESEARCH_HOME, else
// ~/.vrooli/research.
//
//	research/<topic>/<name>  a topic's skill or program
//	research/<name>          a research-home (supervisor) skill or program
//
// A skill is <scope>/skills/<name>/SKILL.md; a program is
// <scope>/programs/<name>.json with <name>.py beside it, where <scope> is
// <home>/topics/<topic> or <home> itself. Files are the authority: every call
// reads the disk, nothing is cached, and a path that resolves outside the
// research home is refused.
package researchhome

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// EnvHome overrides the research home location.
	EnvHome = "VROOLI_RESEARCH_HOME"
	// Prefix starts every research-scoped id.
	Prefix = "research/"
)

var (
	// ErrNotResearch: the id does not start with Prefix.
	ErrNotResearch = errors.New("not a research/ id")
	// ErrInvalid: the id starts with Prefix but is not research/<name> or
	// research/<topic>/<name> with valid segments.
	ErrInvalid = errors.New("invalid research id")
	// ErrOutside: the path resolves outside the research home.
	ErrOutside = errors.New("path resolves outside the research home")
	// ErrNoHome: the research home cannot be located.
	ErrNoHome = errors.New("research home unavailable")

	namePattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	topicPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

// Dir returns the research home. getenv may be nil (os.Getenv).
func Dir(getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if dir := strings.TrimSpace(getenv(EnvHome)); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", fmt.Errorf("%w: %s must be an absolute path, got %q", ErrNoHome, EnvHome, dir)
		}
		return filepath.Clean(dir), nil
	}
	user, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNoHome, err)
	}
	return filepath.Join(user, ".vrooli", "research"), nil
}

// Ref is a parsed research id. Topic is empty for a research-home item.
type Ref struct {
	Topic string
	Name  string
}

// IsID reports whether id uses the research prefix (valid or not).
func IsID(id string) bool { return strings.HasPrefix(strings.TrimSpace(id), Prefix) }

// Parse splits a research id. It returns ErrNotResearch for any other id and
// ErrInvalid for a malformed one; traversal segments never parse.
func Parse(id string) (Ref, error) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(id), Prefix)
	if !ok {
		return Ref{}, ErrNotResearch
	}
	parts := strings.Split(rest, "/")
	var ref Ref
	switch len(parts) {
	case 1:
		ref.Name = parts[0]
	case 2:
		ref.Topic, ref.Name = parts[0], parts[1]
		if !topicPattern.MatchString(ref.Topic) {
			return Ref{}, fmt.Errorf("%w %q: topic %q is not a topic folder name", ErrInvalid, id, ref.Topic)
		}
	default:
		return Ref{}, fmt.Errorf("%w %q: use research/<topic>/<name> or research/<name>", ErrInvalid, id)
	}
	if err := ValidName(ref.Name); err != nil {
		return Ref{}, fmt.Errorf("%w %q: %v", ErrInvalid, id, err)
	}
	return ref, nil
}

// ValidName checks a skill or program name: lowercase letters, digits and
// single hyphens, at most 64 characters.
func ValidName(name string) error {
	if len(name) > 64 || !namePattern.MatchString(name) {
		return fmt.Errorf("name %q must be lowercase letters, digits and single hyphens (at most 64)", name)
	}
	return nil
}

// ID is the ref's canonical research id.
func (r Ref) ID() string {
	if r.Topic == "" {
		return Prefix + r.Name
	}
	return Prefix + r.Topic + "/" + r.Name
}

// ScopeDir is the folder holding the ref's skills/ and programs/.
func (r Ref) ScopeDir(home string) string {
	if r.Topic == "" {
		return home
	}
	return filepath.Join(home, "topics", r.Topic)
}

// SkillPath is where the ref's SKILL.md lives (unresolved).
func (r Ref) SkillPath(home string) string {
	return filepath.Join(r.ScopeDir(home), "skills", r.Name, "SKILL.md")
}

// ProgramPath is where the ref's program contract lives (unresolved); the
// source is the same path with .py.
func (r Ref) ProgramPath(home string) string {
	return filepath.Join(r.ScopeDir(home), "programs", r.Name+".json")
}

// Resolve follows symbolic links and refuses a path outside the home.
func Resolve(home, path string) (string, error) {
	root, err := filepath.EvalSymlinks(home)
	if err != nil {
		return "", fmt.Errorf("research home %s: %w", home, err)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !within(root, real) {
		return "", fmt.Errorf("%w: %s", ErrOutside, path)
	}
	return real, nil
}

// File is one research file read from disk.
type File struct {
	ID       string
	Path     string // resolved
	Content  []byte
	Modified time.Time
}

// ReadSkill reads the SKILL.md for id; home "" resolves Dir(nil). A missing
// file wraps fs.ErrNotExist; Reason classifies any error.
func ReadSkill(home, id string) (Ref, File, error) {
	home, ref, err := locate(home, id)
	if err != nil {
		return ref, File{}, err
	}
	file, err := readFile(home, ref.ID(), ref.SkillPath(home))
	return ref, file, err
}

// Program is a research program's contract and source.
type Program struct {
	Contract File
	Source   File
}

// ReadProgram reads the contract and source for id; home "" resolves
// Dir(nil). A missing contract wraps fs.ErrNotExist; a missing source is an
// error naming it, with the contract still returned.
func ReadProgram(home, id string) (Program, error) {
	home, ref, err := locate(home, id)
	if err != nil {
		return Program{}, err
	}
	contract, err := readFile(home, ref.ID(), ref.ProgramPath(home))
	if err != nil {
		return Program{}, err
	}
	// The source sits beside the resolved contract, where the standalone
	// runner (given contract.Path) also looks for it.
	source, err := readFile(home, ref.ID(), strings.TrimSuffix(contract.Path, ".json")+".py")
	if err != nil {
		return Program{Contract: contract}, fmt.Errorf("program source: %w", err)
	}
	return Program{Contract: contract, Source: source}, nil
}

func locate(home, id string) (string, Ref, error) {
	ref, err := Parse(id)
	if err != nil {
		return "", Ref{}, err
	}
	if home == "" {
		home, err = Dir(nil)
	}
	return home, ref, err
}

// Reason is the stable word a reader reports for a read error.
func Reason(err error) string {
	switch {
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrNotResearch):
		return "invalid_research_id"
	case errors.Is(err, ErrNoHome):
		return "research_home_unavailable"
	case errors.Is(err, ErrOutside):
		return "outside_research_home"
	case IsNotExist(err):
		return "not_found"
	default:
		return "unreadable"
	}
}

func readFile(home, id, path string) (File, error) {
	real, err := Resolve(home, path)
	if err != nil {
		return File{}, err
	}
	info, err := os.Stat(real)
	if err != nil {
		return File{}, err
	}
	if !info.Mode().IsRegular() {
		return File{}, fmt.Errorf("%s is not a regular file", path)
	}
	content, err := os.ReadFile(real)
	if err != nil {
		return File{}, err
	}
	return File{ID: id, Path: real, Content: content, Modified: info.ModTime()}, nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// IsNotExist reports whether err means the research file is absent.
func IsNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }
