package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/vrooli/vrooli/internal/shell"
)

const (
	CronStatusPassed      = "passed"
	CronStatusIssuesFound = "issues_found"
	CronStatusUnsupported = "unsupported"

	CronFindingDeclaredTargetMissing = "declared_target_missing"
	CronFindingDeclaredEntryMissing  = "declared_entry_missing"
	CronFindingUndeclaredRepository  = "undeclared_repository_entry"
)

// CronDeclaration is a repository-owned user-cron contract. Target is always
// relative to the repository root so the declaration remains portable.
type CronDeclaration struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Target   string `json:"target"`
}

// CronEntry is one executable crontab line. Comments, blank lines, and
// environment assignments are intentionally not entries.
type CronEntry struct {
	LineNumber int    `json:"line_number"`
	Schedule   string `json:"schedule"`
	Command    string `json:"command"`
	Raw        string `json:"raw"`
}

type CronFinding struct {
	Code        string `json:"code"`
	Declaration string `json:"declaration,omitempty"`
	LineNumber  int    `json:"line_number,omitempty"`
	Target      string `json:"target,omitempty"`
	Message     string `json:"message"`
}

type CronAudit struct {
	Status       string            `json:"status"`
	Declarations []CronDeclaration `json:"declarations"`
	Entries      []CronEntry       `json:"entries"`
	Findings     []CronFinding     `json:"findings"`
	Detail       string            `json:"detail,omitempty"`
}

type cronManifest struct {
	HostCron []CronDeclaration `json:"hostCron"`
}

// LoadCronDeclarations reads the root service manifest, the declaration
// authority for Vrooli-owned user-cron entries.
func LoadCronDeclarations(root string) ([]CronDeclaration, error) {
	path := filepath.Join(root, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cron declarations from %s: %w", path, err)
	}
	var manifest cronManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse cron declarations from %s: %w", path, err)
	}
	if err := validateCronDeclarations(manifest.HostCron); err != nil {
		return nil, fmt.Errorf("validate cron declarations in %s: %w", path, err)
	}
	return manifest.HostCron, nil
}

func validateCronDeclarations(declarations []CronDeclaration) error {
	seen := make(map[string]struct{}, len(declarations))
	for i, declaration := range declarations {
		declaration.Name = strings.TrimSpace(declaration.Name)
		declaration.Schedule = strings.TrimSpace(declaration.Schedule)
		declaration.Target = filepath.Clean(strings.TrimSpace(declaration.Target))
		if declaration.Name == "" || declaration.Schedule == "" || declaration.Target == "" || declaration.Target == "." {
			return fmt.Errorf("entry %d requires non-empty name, schedule, and target", i)
		}
		if filepath.IsAbs(declaration.Target) || declaration.Target == ".." || strings.HasPrefix(declaration.Target, ".."+string(filepath.Separator)) {
			return fmt.Errorf("entry %q target must stay relative to the repository root", declaration.Name)
		}
		if _, duplicate := seen[declaration.Name]; duplicate {
			return fmt.Errorf("duplicate entry name %q", declaration.Name)
		}
		seen[declaration.Name] = struct{}{}
	}
	return nil
}

// AuditUserCron observes the current user's crontab and checks it against the
// root manifest. It never installs, removes, or executes a cron entry.
func AuditUserCron(ctx context.Context, root string, runner shell.Runner) (CronAudit, error) {
	declarations, err := LoadCronDeclarations(root)
	if err != nil {
		return CronAudit{}, err
	}
	path, err := runner.LookPath("crontab")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CronAudit{Status: CronStatusUnsupported, Declarations: declarations, Entries: []CronEntry{}, Findings: []CronFinding{}, Detail: "crontab command is not available on this host"}, nil
		}
		return CronAudit{}, fmt.Errorf("locate crontab command: %w", err)
	}
	output, err := runner.Run(ctx, path, "-l")
	if err != nil {
		return CronAudit{}, fmt.Errorf("read current user crontab: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return AuditCron(root, declarations, string(output))
}

// AuditCron is the platform-neutral policy engine used by the live command and
// tests. It reports both sides of drift: broken declarations and undeclared
// host entries that point back into the repository.
func AuditCron(root string, declarations []CronDeclaration, crontab string) (CronAudit, error) {
	if err := validateCronDeclarations(declarations); err != nil {
		return CronAudit{}, err
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return CronAudit{}, fmt.Errorf("resolve repository root: %w", err)
	}
	root = filepath.Clean(root)
	entries := ParseCronEntries(crontab)
	report := CronAudit{Status: CronStatusPassed, Declarations: declarations, Entries: entries, Findings: []CronFinding{}}

	matched := make(map[string]bool, len(declarations))
	for _, declaration := range declarations {
		target := filepath.Join(root, filepath.FromSlash(declaration.Target))
		if _, statErr := os.Stat(target); statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				report.Findings = append(report.Findings, CronFinding{Code: CronFindingDeclaredTargetMissing, Declaration: declaration.Name, Target: declaration.Target, Message: fmt.Sprintf("declared cron target %q does not exist", declaration.Target)})
			} else {
				return CronAudit{}, fmt.Errorf("inspect declared cron target %s: %w", target, statErr)
			}
		}
		for _, entry := range entries {
			if entry.Schedule == declaration.Schedule && commandReferencesPath(entry.Command, target) {
				matched[declaration.Name] = true
			}
		}
	}

	for _, entry := range entries {
		if !commandReferencesPath(entry.Command, root) {
			continue
		}
		declared := false
		for _, declaration := range declarations {
			target := filepath.Join(root, filepath.FromSlash(declaration.Target))
			if entry.Schedule == declaration.Schedule && commandReferencesPath(entry.Command, target) {
				declared = true
				break
			}
		}
		if !declared {
			report.Findings = append(report.Findings, CronFinding{Code: CronFindingUndeclaredRepository, LineNumber: entry.LineNumber, Target: root, Message: "cron entry points inside the repository but has no matching hostCron declaration"})
		}
	}

	for _, declaration := range declarations {
		if !matched[declaration.Name] {
			report.Findings = append(report.Findings, CronFinding{Code: CronFindingDeclaredEntryMissing, Declaration: declaration.Name, Target: declaration.Target, Message: "declared cron entry is not installed with the declared schedule and target"})
		}
	}
	if len(report.Findings) > 0 {
		report.Status = CronStatusIssuesFound
	}
	return report, nil
}

// ParseCronEntries recognizes standard five-field schedules and @macros.
func ParseCronEntries(crontab string) []CronEntry {
	lines := strings.Split(crontab, "\n")
	entries := make([]CronEntry, 0, len(lines))
	for index, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || isCronEnvironmentAssignment(trimmed) {
			continue
		}
		fields := strings.Fields(trimmed)
		fieldCount := 5
		if strings.HasPrefix(fields[0], "@") {
			fieldCount = 1
		}
		if len(fields) <= fieldCount {
			continue
		}
		entries = append(entries, CronEntry{LineNumber: index + 1, Schedule: strings.Join(fields[:fieldCount], " "), Command: strings.Join(fields[fieldCount:], " "), Raw: raw})
	}
	return entries
}

func isCronEnvironmentAssignment(line string) bool {
	equals := strings.IndexByte(line, '=')
	if equals <= 0 {
		return false
	}
	for _, char := range line[:equals] {
		if char != '_' && !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

func commandReferencesPath(command, wanted string) bool {
	wanted = filepath.Clean(wanted)
	for _, candidate := range commandPathTokens(command) {
		candidate = filepath.Clean(candidate)
		if candidate == wanted || strings.HasPrefix(candidate, wanted+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func commandPathTokens(command string) []string {
	return strings.FieldsFunc(command, func(char rune) bool {
		switch char {
		case ' ', '\t', '\r', '\n', '\'', '"', '`', ';', '|', '&', '>', '<', '(', ')':
			return true
		default:
			return false
		}
	})
}
