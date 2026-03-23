// Package skills provides a centralized registry of prompt-manager skill IDs
// used across the swarm-manager API. This avoids scattering hardcoded skill
// strings throughout handler code and makes the mapping testable.
package skills

// researchSkillIDs maps (mode, kind) to prompt-manager skill IDs for
// backlog research operations.
var researchSkillIDs = map[string]map[string]string{
	"workshop": {
		"idea":     "swarm-manager-workshop",
		"research": "swarm-manager-workshop",
		"fix":      "swarm-manager-workshop",
		"execute":  "swarm-manager-workshop",
		"chore":    "swarm-manager-workshop",
	},
	"research": {
		"idea":     "swarm-manager-research-idea",
		"fix":      "swarm-manager-research-fix",
		"execute":  "swarm-manager-research-general",
		"research": "swarm-manager-research-general",
		"chore":    "swarm-manager-research-general",
	},
	"initialize": {
		"idea":     "swarm-manager-initialize-backlog",
		"research": "swarm-manager-initialize-backlog",
		"fix":      "swarm-manager-initialize-backlog",
		"execute":  "swarm-manager-initialize-backlog",
		"chore":    "swarm-manager-initialize-backlog",
	},
}

// classifyCaptureSkillID is the skill used to classify captured items.
const classifyCaptureSkillID = "swarm-manager-classify-capture"

// Resolve returns the prompt-manager skill ID for a given research mode and
// backlog kind. If the combination is not found, it falls back to
// "swarm-manager-research-general".
func Resolve(mode, kind string) string {
	if kindMap, ok := researchSkillIDs[mode]; ok {
		if id, ok := kindMap[kind]; ok {
			return id
		}
	}
	return "swarm-manager-research-general"
}

// ClassifyCaptureSkillID returns the skill ID used for capture classification.
func ClassifyCaptureSkillID() string {
	return classifyCaptureSkillID
}
