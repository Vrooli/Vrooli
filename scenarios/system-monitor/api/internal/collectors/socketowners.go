package collectors

import (
	"os"
	"sort"
	"strconv"
)

// defaultSocketAttributionThreshold is the established-connection count above
// which per-process attribution is worth its cost. Below it the aggregate
// number is not alarming and the /proc walk is pure overhead; above it the
// aggregate is useless on its own. During the 2026-08-21 incident the host
// showed 57,706 established connections, and the only question that mattered —
// which process owns them — took a manual `ss -antp` to answer. It was 93% one
// process.
const defaultSocketAttributionThreshold = 5000

// socketAttributionThresholdEnv overrides the threshold; 0 disables attribution.
const socketAttributionThresholdEnv = "SYSTEM_MONITOR_SOCKET_ATTRIBUTION_THRESHOLD"

// SocketOwner is one process's share of the host's TCP sockets.
type SocketOwner struct {
	PID   int    `json:"pid"`
	Comm  string `json:"name"`
	Count int    `json:"connections"`
}

// SocketAttribution reports which processes own the host's sockets, along with
// how much of the total it could actually account for. Attributed is never
// assumed to equal the total: mapping requires reading /proc/<pid>/fd, which
// fails for processes owned by other users, and sockets open and close during
// the walk. Reporting the coverage keeps a partial answer from reading as a
// complete one.
type SocketAttribution struct {
	Owners     []SocketOwner `json:"owners"`
	Attributed int           `json:"attributed"`
	Total      int           `json:"total"`
	Supported  bool          `json:"supported"`
	Reason     string        `json:"reason,omitempty"`
}

func socketAttributionThreshold() int {
	raw := os.Getenv(socketAttributionThresholdEnv)
	if raw == "" {
		return defaultSocketAttributionThreshold
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return defaultSocketAttributionThreshold
	}
	return value
}

// shouldAttributeSockets reports whether the established count justifies the
// /proc walk. A zero threshold disables attribution entirely.
func shouldAttributeSockets(established int) bool {
	threshold := socketAttributionThreshold()
	if threshold == 0 {
		return false
	}
	return established >= threshold
}

// topSocketOwners reduces an inode-to-pid attribution to the heaviest holders.
func topSocketOwners(counts map[int]int, names map[int]string, limit int) []SocketOwner {
	owners := make([]SocketOwner, 0, len(counts))
	for pid, count := range counts {
		owners = append(owners, SocketOwner{PID: pid, Comm: names[pid], Count: count})
	}
	sort.SliceStable(owners, func(i, j int) bool {
		if owners[i].Count != owners[j].Count {
			return owners[i].Count > owners[j].Count
		}
		return owners[i].PID < owners[j].PID
	})
	if limit > 0 && len(owners) > limit {
		owners = owners[:limit]
	}
	return owners
}
