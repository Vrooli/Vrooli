package topcli

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/vroolierr"
)

type CleanupRequest struct {
	Target string
	Args   []string
}

func ParseCleanupRequest(args []string) (CleanupRequest, error) {
	if len(args) == 0 {
		return CleanupRequest{}, commandHelpOnly("")
	}
	target := strings.TrimSpace(args[0])
	switch target {
	case "help", "--help", "-h":
		return CleanupRequest{}, commandHelpOnly("")
	case "orphans", "locks":
		return CleanupRequest{Target: target, Args: append([]string(nil), args[1:]...)}, nil
	default:
		return CleanupRequest{}, &vroolierr.Error{
			Err:         fmt.Errorf("unknown cleanup target: %s", target),
			Category:    usageCategory,
			Hint:        usageHint("cleanup"),
			Suggestions: []string{"orphans", "locks"},
		}
	}
}
