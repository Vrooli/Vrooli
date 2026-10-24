// Package expiry renders the instance-side expiry guardrail.
package expiry

import (
	"fmt"
	"time"
)

func RenderTimer(expiry time.Time) string {
	return fmt.Sprintf("[Unit]\nDescription=Vrooli compute instance expiry\n\n[Timer]\nOnCalendar=%s\nPersistent=true\n\n[Install]\nWantedBy=timers.target\n", expiry.UTC().Format("2006-01-02 15:04:05 UTC"))
}
