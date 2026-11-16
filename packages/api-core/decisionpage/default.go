package decisionpage

import (
	"os"
	"path/filepath"
)

// DefaultRoot is where every sender's jobs live, so one sender can read what
// another asked for the same subject: ~/.vrooli/state/vrooli/decision-pages.
func DefaultRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vrooli", "state", "vrooli", "decision-pages")
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
