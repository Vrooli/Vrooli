// Package livecapture provides live recording session management and related types.
// This package wraps the unified driver package for backward compatibility.
package livecapture

import "github.com/vrooli/browser-automation-studio/automation/driver"

// Type aliases for backward compatibility.
// The canonical types are now in the driver package.
type (
	RecordedAction    = driver.RecordedAction
	SelectorSet       = driver.SelectorSet
	SelectorCandidate = driver.SelectorCandidate
	ElementMeta       = driver.ElementMeta
)
