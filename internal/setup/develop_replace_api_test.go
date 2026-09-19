package setup

import (
	"errors"
	"testing"
	"time"
)

// Only the project API's own process is stopped; a port held by another
// program is left alone for the launch to report.
func TestReplaceUnhealthyProjectAPIStopsOnlyTheProjectAPI(t *testing.T) {
	running := map[int]bool{11: true, 22: true, 33: true}
	var killed []int
	ops := apiProcessOps{
		listeners: func(int) []int { return []int{11, 22, 33} },
		exePath: func(pid int) (string, error) {
			switch pid {
			case 11:
				return "/Users/owner/vrooli/.vrooli/build/vrooli-api (deleted)", nil
			case 22:
				return "/usr/sbin/nginx", nil
			}
			return "", errors.New("gone")
		},
		kill: func(pid int, _ bool) error {
			killed = append(killed, pid)
			running[pid] = false
			return nil
		},
		alive: func(pid int) bool { return running[pid] },
		grace: time.Second,
	}
	stopped := replaceUnhealthyProjectAPI(8092, ops)
	if len(stopped) != 1 || stopped[0] != 11 || len(killed) != 1 {
		t.Fatalf("stopped=%v killed=%v, want only the vrooli-api pid", stopped, killed)
	}
}
