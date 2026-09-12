package app

import (
	"os"
	"strings"

	"architecture-cartographer/internal/graph/scenariopath"
)

func TSProjectCandidates() []scenariopath.Candidate {
	return projectCandidates("CARTOGRAPHER_TS_PROJECT_DIRS", []string{"ui", "."}, "tsconfig.json")
}

func GoProjectCandidates() []scenariopath.Candidate {
	return projectCandidates("CARTOGRAPHER_GO_PROJECT_DIRS", []string{"api", "cli", "."}, "go.mod")
}

func projectCandidates(envKey string, defaultSubdirs []string, marker string) []scenariopath.Candidate {
	subdirs := defaultSubdirs
	if raw := strings.TrimSpace(os.Getenv(envKey)); raw != "" {
		parsed := make([]string, 0, len(defaultSubdirs))
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				parsed = append(parsed, p)
			}
		}
		if len(parsed) > 0 {
			subdirs = parsed
		}
	}
	out := make([]scenariopath.Candidate, 0, len(subdirs))
	for _, d := range subdirs {
		out = append(out, scenariopath.Candidate{Subdir: d, Marker: marker})
	}
	return out
}
