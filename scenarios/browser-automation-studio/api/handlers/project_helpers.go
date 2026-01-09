package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/paths"
)

// validateAndPrepareFolderPath delegates to the paths package for folder validation and creation.
// This keeps domain-level path validation in the internal package while providing
// a handler-friendly interface.
func validateAndPrepareFolderPath(folderPath string, log *logrus.Logger) (string, error) {
	return paths.ValidateAndPrepareFolderPath(folderPath, log)
}
