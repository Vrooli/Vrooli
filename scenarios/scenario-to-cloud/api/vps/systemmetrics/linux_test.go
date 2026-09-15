package systemmetrics

import "testing"

func TestParseOSRelease(t *testing.T) {
	t.Parallel()

	id, version := ParseOSRelease("ID=ubuntu\nVERSION_ID=\"24.04\"\n")
	if id != "ubuntu" || version != "24.04" {
		t.Fatalf("unexpected parse result: id=%q version=%q", id, version)
	}
}

func TestLinuxCollector_ParseSystemState_MeminfoPreferred(t *testing.T) {
	t.Parallel()

	c := linuxCollector{}
	state := c.ParseSystemState(map[string]CommandResult{
		"meminfo": {
			Stdout: "MemTotal:       1024000 kB\nMemFree:         50000 kB\nMemAvailable:   300000 kB\nBuffers:         10000 kB\nCached:          70000 kB\nSwapTotal:       512000 kB\nSwapFree:        400000 kB\n",
		},
	})

	if state.Memory.TotalMB != 1000 {
		t.Fatalf("total memory mismatch: got %d", state.Memory.TotalMB)
	}
	if state.Memory.TotalBytes != 1024000*1024 {
		t.Fatalf("total memory bytes mismatch: got %d", state.Memory.TotalBytes)
	}
	if state.Memory.FreeMB != 292 {
		t.Fatalf("free(available) memory mismatch: got %d", state.Memory.FreeMB)
	}
	if state.Memory.UsedMB != 707 {
		t.Fatalf("used memory mismatch: got %d", state.Memory.UsedMB)
	}
	if state.Swap.TotalMB != 500 || state.Swap.UsedMB != 109 {
		t.Fatalf("swap mismatch: total=%d used=%d", state.Swap.TotalMB, state.Swap.UsedMB)
	}
}

func TestLinuxCollector_ParseSystemState_WithoutMeminfoLeavesMemoryEmpty(t *testing.T) {
	t.Parallel()

	c := linuxCollector{}
	state := c.ParseSystemState(map[string]CommandResult{})

	if state.Memory.TotalMB != 0 || state.Memory.TotalBytes != 0 {
		t.Fatalf("expected empty memory when remote meminfo snapshot is absent, got %+v", state.Memory)
	}
	if state.Swap.TotalMB != 0 || state.Swap.UsedMB != 0 {
		t.Fatalf("expected empty swap when remote meminfo snapshot is absent, got %+v", state.Swap)
	}
}

func TestLinuxCollector_ParseSystemState_DiskFromDFKB(t *testing.T) {
	t.Parallel()

	c := linuxCollector{}
	state := c.ParseSystemState(map[string]CommandResult{
		"df_kb": {Stdout: "/dev/sda1 25165824 15728640 9437184 62% /"},
	})

	if state.Disk.TotalGB != 24 || state.Disk.UsedGB != 15 || state.Disk.FreeGB != 9 {
		t.Fatalf("disk mismatch: total=%d used=%d free=%d", state.Disk.TotalGB, state.Disk.UsedGB, state.Disk.FreeGB)
	}
	if state.Disk.TotalBytes != 25165824*1024 {
		t.Fatalf("disk total bytes mismatch: got %d", state.Disk.TotalBytes)
	}
	if state.Disk.UsagePercent != 62 {
		t.Fatalf("disk usage mismatch: got %.1f", state.Disk.UsagePercent)
	}
}

func TestParseCPUUsageFromProcStat(t *testing.T) {
	t.Parallel()

	usage := ParseCPUUsageFromProcStat("cpu  1000 50 200 50000 100 10 5 0\ncpu  1010 50 200 50100 100 10 5 0")
	if usage < 0 || usage > 30 {
		t.Fatalf("unexpected usage range: %.2f", usage)
	}
}

func TestParseCPUUsageFromProcStat_MultiSampleAverage(t *testing.T) {
	t.Parallel()

	usage := ParseCPUUsageFromProcStat(
		"cpu  100 0 100 1000 0 0 0 0\n" +
			"cpu  150 0 150 1050 0 0 0 0\n" +
			"cpu  160 0 160 1150 0 0 0 0\n" +
			"cpu  260 0 260 1200 0 0 0 0",
	)

	if usage < 40 || usage > 75 {
		t.Fatalf("unexpected multi-sample usage average: %.2f", usage)
	}
}
