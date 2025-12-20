package livecapture

import (
	driver "github.com/vrooli/browser-automation-studio/automation/driver"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// RecordedActionFromTimelineEntry converts a TimelineEntry to the legacy RecordedAction type.
func RecordedActionFromTimelineEntry(entry *bastimeline.TimelineEntry) RecordedAction {
	return driver.RecordedActionFromTimelineEntry(entry)
}
