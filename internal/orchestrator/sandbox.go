package orchestrator

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/process"
)

func SandboxAffectedScenarios(home, mergedPath string) ([]string, error) {
	processRoot := filepath.Join(home, ".vrooli", "processes", "scenarios")
	entries, err := os.ReadDir(processRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	affected := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		records, readErr := process.ReadScenarioRecords(home, name)
		if readErr != nil {
			return nil, readErr
		}
		for _, record := range records {
			if strings.HasPrefix(record.WorkingDir, mergedPath) {
				affected = append(affected, name)
				break
			}
		}
	}
	sort.Strings(affected)
	return affected, nil
}
