package decisionpage

import (
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

// DefaultRoot is where every sender's jobs live, so one sender can read what
// another asked for the same subject: ~/.vrooli/state/vrooli/decision-pages.
func DefaultRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join("state", "vrooli", "decision-pages")
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return filepath.Join(home, "state", "vrooli", "decision-pages")
	}
	return filepath.Join(root, "state", "vrooli", "decision-pages")
}

// New is the production writer: jobs under DefaultRoot, runs through Agent
// Manager with the Hub's declared writer profile, pages checked by the Hub.
// A caller that resolves scenario URLs its own way sets the BaseURL fields of
// the returned Launcher and Checker.
func New() *Writer {
	return &Writer{
		Root:     DefaultRoot(),
		Launcher: &AgentManager{},
		Checker:  &HubCheck{},
		SkillDir: filepath.Join(RepoRoot(), SkillRelDir),
		Command:  DefaultCommand,
	}
}
