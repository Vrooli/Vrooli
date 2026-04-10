package handlers

import (
	"os"
	"path/filepath"
)

// FSTargetValidator checks target existence by looking for the directory on disk.
type FSTargetValidator struct {
	vrooliRoot string
}

// NewFSTargetValidator creates a validator rooted at the given Vrooli workspace path.
func NewFSTargetValidator(vrooliRoot string) *FSTargetValidator {
	return &FSTargetValidator{vrooliRoot: vrooliRoot}
}

// TargetExists returns true if the target directory exists under scenarios/ or resources/.
func (v *FSTargetValidator) TargetExists(taskType, targetName string) bool {
	dir := "scenarios"
	if taskType == "resource" {
		dir = "resources"
	}
	p := filepath.Join(v.vrooliRoot, dir, targetName)
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
