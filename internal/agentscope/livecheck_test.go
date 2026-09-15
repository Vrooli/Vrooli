package agentscope

import (
	"os"
	"testing"

	platformgo "github.com/vrooli/platform-go"
)

// TestLiveHostCensus reads this host's real slice. It is skipped unless
// asked for; it asserts nothing about the numbers, it prints what the census
// sees so a reviewer can compare it with systemd and the kernel directly.
func TestLiveHostCensus(t *testing.T) {
	if os.Getenv("VROOLI_AGENTSCOPE_LIVE") == "" {
		t.Skip("set VROOLI_AGENTSCOPE_LIVE=1 to read this host's agent slice")
	}
	slice := Ref("/user.slice/user-1000.slice/user@1000.service/vrooli.slice/" + Slice)
	occupancy, err := platformgo.ScopeOccupancy(slice)
	if err != nil {
		t.Skipf("no agent slice on this host: %v", err)
	}
	entries, err := NewReader().Census(slice)
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	t.Logf("tasks %d/%d (%.1f%%), memory %d/%d (%.1f%%), memory.high events %d",
		occupancy.Tasks, occupancy.TasksMax, occupancy.TaskSaturation()*100,
		occupancy.MemoryBytes, occupancy.MemoryMaxBytes, occupancy.MemorySaturation()*100,
		occupancy.MemoryHighEvents)
	t.Logf("%d session scopes, %d agentless holding %d tasks",
		len(entries), len(Agentless(entries)), AgentlessTasks(entries))
	for _, entry := range Agentless(entries) {
		t.Logf("  agentless: %s (%d tasks)", entry.Name, entry.Tasks)
	}
}
