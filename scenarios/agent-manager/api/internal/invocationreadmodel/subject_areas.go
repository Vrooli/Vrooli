package invocationreadmodel

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"agent-manager/internal/domain"
)

// Subject qualifiers. A run's subject answers two different questions and they
// must stay tellable apart: which capabilities the agent used, and which part
// of the project it actually worked on. A flat list of both reads as tool
// telemetry and cannot answer "what changed".
const (
	SubjectTool = "tool:"
	SubjectPath = "path:"
)

// maxSubjectPaths bounds the projection. A long agent run touches many files;
// the areas it worked in are a much smaller set, and an unbounded list would
// make the subject unreadable and the row large.
const maxSubjectPaths = 24

// subjectPathPattern matches the repo areas work is attributed to. Only the
// root and the unit directly under it are captured — "scenarios/agent-manager"
// is the useful attribution, individual file paths are not.
var subjectPathPattern = regexp.MustCompile(`\b(scenarios|packages|resources|platforms|safeguards|tools|docs)/([A-Za-z0-9][A-Za-z0-9._-]*)`)

// DeriveRunAreas extracts the project areas a run touched from its retained
// tool-call events. Facts cannot answer this: they record the command that ran,
// not the paths it named.
func DeriveRunAreas(events []*domain.RunEvent) []string {
	seen := make(map[string]struct{})
	for _, event := range events {
		if event == nil {
			continue
		}
		call, ok := event.Data.(*domain.ToolCallEventData)
		if !ok {
			continue
		}
		// Only the tool's own input names paths. Scanning the whole event would
		// also match paths quoted in surrounding narration, which is not work.
		payload, err := json.Marshal(call.Input)
		if err != nil {
			continue
		}
		for _, match := range subjectPathPattern.FindAllStringSubmatch(string(payload), -1) {
			root, unit := match[1], strings.TrimRight(match[2], ".")
			if unit == "" {
				continue
			}
			seen[root+"/"+unit] = struct{}{}
		}
	}
	areas := make([]string, 0, len(seen))
	for area := range seen {
		areas = append(areas, area)
	}
	sort.Strings(areas)
	if len(areas) > maxSubjectPaths {
		areas = areas[:maxSubjectPaths]
	}
	return areas
}
