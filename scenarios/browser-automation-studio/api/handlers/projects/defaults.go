package projects

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/internal/paths"
)

// defaultPaths is the production PathPreparer; tests inject a fake.
type defaultPaths struct {
	log *logrus.Logger
}

func (d defaultPaths) Prepare(folderPath string) (string, error) {
	return paths.ValidateAndPrepareFolderPath(folderPath, d.log)
}

func (d defaultPaths) MakeAll(absPath string, perm uint32) error {
	return os.MkdirAll(absPath, os.FileMode(perm))
}
