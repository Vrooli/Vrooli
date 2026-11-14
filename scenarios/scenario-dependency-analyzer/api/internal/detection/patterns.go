package detection

import "regexp"

// patterns.go contains only vocabulary-independent patterns. Resource-specific
// heuristics are compiled from the live resource manifest catalog in
// resource_scanner.go so a new resource is detected without a code edit.
var resourceCommandPattern = regexp.MustCompile(`resource-([a-z0-9-]+)`)

var (
	vrooliScenarioPattern       = regexp.MustCompile(`vrooli\s+scenario\s+(?:run|test|status|start|stop)\s+([a-z0-9-]+)`)
	resourceDetectionExtensions = []string{".go", ".js", ".ts", ".tsx", ".sh", ".py", ".md", ".json", ".yml", ".yaml"}
	scenarioDetectionExtensions = []string{".go", ".js", ".sh", ".py", ".md"}
)
