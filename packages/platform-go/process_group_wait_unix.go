//go:build unix && !linux

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func processGroupHasLiveMembers(ctx context.Context, groupID int) (bool, error) {
	output, err := exec.CommandContext(ctx, "ps", "-axo", "pgid=,stat=").Output()
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 2 {
			return false, fmt.Errorf("malformed ps process-group row %q", line)
		}
		memberGroup, err := strconv.Atoi(fields[0])
		if err != nil {
			return false, fmt.Errorf("parse process group from ps row %q: %w", line, err)
		}
		if memberGroup == groupID && !strings.Contains(fields[1], "Z") {
			return true, nil
		}
	}
	return false, nil
}
