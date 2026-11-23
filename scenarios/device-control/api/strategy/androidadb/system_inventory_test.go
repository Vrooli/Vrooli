package androidadb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemInventoryParsesBoundedAndroidFactsWithoutWrites(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s moto shell cat /proc/cpuinfo": []byte("processor : 0\nprocessor : 1\nprocessor : 2\nprocessor : 3\n"),
		"adb -s moto shell cat /proc/loadavg": []byte("0.10 0.20 0.30 1/200 1234\n"),
		"adb -s moto shell cat /proc/meminfo": []byte("MemTotal: 4096 kB\nMemFree: 500 kB\nMemAvailable: 2048 kB\n"),
		"adb -s moto shell df -k /data":       []byte("Filesystem 1K-blocks Used Available Use% Mounted on\n/dev/block/dm-1 10000 4000 6000 40% /data\n"),
		"adb -s moto shell pm list packages":  []byte("package:com.android.settings\npackage:com.example.app\npackage:bad package\npackage:com.example.app\n"),
	}}
	inventory, err := NewWithRunner(runner, "moto").InspectSystemInventory(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint32(4), inventory.CPULogicalCores)
	require.Equal(t, 0.1, *inventory.Load1)
	require.Equal(t, uint64(4096*1024), *inventory.MemoryTotalBytes)
	require.Equal(t, uint64(2048*1024), *inventory.MemoryAvailableBytes)
	require.Equal(t, uint64(10000*1024), *inventory.DataTotalBytes)
	require.Equal(t, uint64(6000*1024), *inventory.DataAvailableBytes)
	require.Equal(t, []string{"com.android.settings", "com.example.app"}, inventory.InstalledPackages)
	require.False(t, inventory.PackagesTruncated)
	require.NotContains(t, strings.Join(runner.calls, "\n"), "settings put")
}

func TestSystemInventoryPackageParserCapsAndValidatesNames(t *testing.T) {
	lines := make([]string, 0, maxInstalledPackages+1)
	for i := 0; i < maxInstalledPackages+1; i++ {
		lines = append(lines, fmt.Sprintf("package:com.example.app%d", i))
	}
	packages, truncated := parseInstalledPackages(strings.Join(lines, "\n"))
	require.Len(t, packages, maxInstalledPackages)
	require.True(t, truncated)
}
