package tasks

import (
	"strings"
	"time"
)

// CooldownRemaining returns the remaining cooldown duration for a task.
// Returns 0 if no cooldown is set or if it has expired.
func CooldownRemaining(task *TaskItem) time.Duration {
	if task == nil {
		return 0
	}
	if strings.TrimSpace(task.CooldownUntil) == "" {
		return 0
	}
	ts, err := time.Parse(time.RFC3339, task.CooldownUntil)
	if err != nil {
		return 0
	}
	remaining := time.Until(ts)
	if remaining <= 0 {
		return 0
	}
	return remaining
}
