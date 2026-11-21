package skills

import (
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/researchhome"
)

// Research skills (research/<topic>/<name>, research/<name>) are files in the
// research home, read fresh on every request. They are not in the catalog:
// no metadata index, versions or variants. See packages/api-core/researchhome.

// SetResearchHome pins the research home; empty resolves it per read from
// VROOLI_RESEARCH_HOME or ~/.vrooli/research.
func (h *Handlers) SetResearchHome(dir string) { h.researchHome = dir }

func (h *Handlers) readResearchSkill(identifier string, variables map[string]string) (Response, *ReadIssue) {
	ref, file, err := researchhome.ReadSkill(h.researchHome, identifier)
	if err != nil {
		return Response{}, &ReadIssue{Identifier: identifier, Reason: researchhome.Reason(err)}
	}
	return researchResponse(ref, file, variables), nil
}

func researchResponse(ref researchhome.Ref, file researchhome.File, variables map[string]string) Response {
	// The file is served even when its frontmatter is malformed; Research
	// Desk reports the problem. Files are the authority.
	front, _ := parseFrontmatterContent(file.Path, file.Content)
	content := StripFrontmatter(string(file.Content))
	resp := Response{
		ID:          ref.ID(),
		Name:        ref.Name,
		Description: front.Description,
		Folder:      "research",
		File:        file.Path,
		SkillDir:    filepath.Dir(file.Path),
		ContentPath: file.Path,
		UpdatedAt:   file.Modified.UTC().Format(time.RFC3339),
		Variables:   ExtractVariables(content),
		ContentHash: contentSHA256(content),
		Modes:       []string{},
		Tags:        []string{},
	}
	if len(variables) > 0 {
		content = SubstituteVariables(content, variables)
	}
	resp.Content = content
	return resp
}
