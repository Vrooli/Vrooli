package skills

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/api-core/researchhome"
)

// Research skills (research/<topic>/<name>, research/<name>) are files in the
// research home. The API serves them like any skill; when it cannot — down,
// an older binary, or a different research home — the CLI reads the same
// file so research never depends on Prompt Manager running.

// ResearchRead reports whether args are `skill read …` naming a research id
// and returns the read arguments. Such a read skips the API preflight because
// cmdRead falls back to the files itself.
func ResearchRead(args []string) ([]string, bool) {
	if len(args) < 3 {
		return nil, false
	}
	switch args[0] {
	case "skill", "skills", "s":
	default:
		return nil, false
	}
	if args[1] != "read" && args[1] != "cat" {
		return nil, false
	}
	for _, arg := range args[2:] {
		if researchhome.IsID(arg) {
			return args[2:], true
		}
	}
	return nil, false
}

// researchFallback reads, from files, the research ids the API did not serve.
// apiErr is the API failure, or nil when the API answered. It returns the
// skills found and the issues for those still missing; one stderr line per
// file read names it.
func researchFallback(ids []string, resp ReadResponse, apiErr error, stderr io.Writer) ([]SkillResponse, []ReadIssue) {
	unserved := map[string]bool{}
	for _, miss := range resp.Missing {
		unserved[miss.Identifier] = true
	}
	var found []SkillResponse
	var missing []ReadIssue
	for _, id := range ids {
		if !researchhome.IsID(id) || (apiErr == nil && !unserved[id]) {
			continue
		}
		skill, reason := readResearchFile(id)
		if reason != "" {
			missing = append(missing, ReadIssue{Identifier: id, Reason: reason})
			continue
		}
		why := "Prompt Manager did not serve it"
		if apiErr != nil {
			why = "Prompt Manager unreachable"
		}
		fmt.Fprintf(stderr, "prompt-manager: read %s from %s (%s)\n", id, skill.ContentPath, why)
		found = append(found, skill)
	}
	return found, missing
}

func readResearchFile(id string) (SkillResponse, string) {
	ref, file, err := researchhome.ReadSkill("", id)
	if err != nil {
		return SkillResponse{}, researchhome.Reason(err) + ": " + err.Error()
	}
	return SkillResponse{
		ID:          ref.ID(),
		Name:        ref.Name,
		Content:     stripFrontmatter(string(file.Content)),
		Folder:      "research",
		SkillDir:    filepath.Dir(file.Path),
		ContentPath: file.Path,
	}, ""
}

// stripFrontmatter matches the API: the body after a leading --- block.
func stripFrontmatter(content string) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "")
		}
	}
	return content
}

// Read runs `skill read` with already-split arguments.
func Read(ctx appctx.Context, args []string) error { return cmdRead(ctx, args) }
