package main

import (
	sharedpath "scenario-to-desktop-api/shared/path"
)

// detectVrooliRoot finds the Vrooli root directory.
// This is a convenience wrapper around the shared path package.
func detectVrooliRoot() string {
	return sharedpath.DetectVrooliRoot()
}
