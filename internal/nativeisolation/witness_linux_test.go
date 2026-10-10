//go:build linux

package nativeisolation

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func showFixture(s ProbeSpec) string {
	values := map[string]string{"Id": s.unit(), "LoadState": "loaded", "ActiveState": "active", "SubState": "running", "MainPID": "42", "ExecMainStartTimestampMonotonic": "1000000", "InvocationID": strings.Repeat("a", 32), "NRestarts": "0", "TasksCurrent": "1", "User": fmt.Sprint(s.Manifest.NativeUID), "Group": fmt.Sprint(s.Manifest.NativeGID), "RootDirectory": s.Manifest.RootFS, "WorkingDirectory": s.repository(), "BindPaths": s.Manifest.WorkspaceSource + ":" + s.repository(), "ReadWritePaths": s.repository(), "NoNewPrivileges": "yes", "CapabilityBoundingSet": "", "AmbientCapabilities": "", "Restart": "no", "KillMode": "control-group", "Delegate": "no", "PrivateMounts": "yes", "PrivateDevices": "yes", "ProtectSystem": "strict", "ProtectHome": "tmpfs", "RestrictNamespaces": "yes", "RestrictSUIDSGID": "yes", "ProtectKernelTunables": "yes", "ProtectKernelModules": "yes", "ProtectKernelLogs": "yes", "ProtectControlGroups": "yes", "RestrictAddressFamilies": "AF_INET AF_INET6", "IPAddressDeny": "any", "IPAddressAllow": strings.Join(s.Manifest.NetworkAllow, " "), "RuntimeMaxUSec": "1h", "TimeoutStopUSec": "5s"}
	var lines []string
	for _, k := range witnessProperties {
		lines = append(lines, k+"="+values[k])
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestLiveUnitFactsRefuseExitedRestartedAndWeakenedUnits(t *testing.T) {
	s, _ := probeFixture()
	good := showFixture(s)
	if pid, _, e := witnessShow([]byte(good), s); e != nil || pid != 42 {
		t.Fatal("valid exact unit fixture refused", e)
	}
	for _, pair := range [][2]string{{"MainPID=42", "MainPID=0"}, {"ExecMainStartTimestampMonotonic=1000000", "ExecMainStartTimestampMonotonic=0"}, {"SubState=running", "SubState=exited"}, {"NRestarts=0", "NRestarts=1"}, {"TasksCurrent=1", "TasksCurrent=2"}, {"NoNewPrivileges=yes", "NoNewPrivileges=no"}, {"CapabilityBoundingSet=\n", "CapabilityBoundingSet=cap_setuid\n"}, {"Restart=no", "Restart=always"}, {"PrivateMounts=yes", "PrivateMounts=no"}, {"RestrictAddressFamilies=AF_INET AF_INET6", "RestrictAddressFamilies=AF_UNIX"}, {"IPAddressDeny=any", "IPAddressDeny="}, {"IPAddressAllow=\n", "IPAddressAllow=0.0.0.0/0\n"}, {"RuntimeMaxUSec=1h", "RuntimeMaxUSec=infinity"}, {"RuntimeMaxUSec=1h", "RuntimeMaxUSec=9h"}, {"TimeoutStopUSec=5s", "TimeoutStopUSec=0"}, {"WorkingDirectory=" + s.repository(), "WorkingDirectory=/"}, {"BindPaths=" + s.Manifest.WorkspaceSource + ":" + s.repository(), "BindPaths=/foreign:/foreign"}, {"ReadWritePaths=" + s.repository(), "ReadWritePaths=/"}} {
		if _, _, e := witnessShow([]byte(strings.Replace(good, pair[0], pair[1], 1)), s); e == nil {
			t.Fatal("unsafe unit accepted", pair)
		}
	}
	for _, bad := range []string{good + "MainPID=42\n", strings.Replace(good, "MainPID=42\n", "", 1), good + "Unknown=true\n"} {
		if _, _, e := witnessShow([]byte(bad), s); e == nil {
			t.Fatal("ambiguous unit facts accepted")
		}
	}
}

func TestProcStartCgroupAndGIDFacts(t *testing.T) {
	fields := make([]string, 20)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0] = "S"
	fields[19] = "100"
	raw := "42 (fixed helper) " + strings.Join(fields, " ")
	if ticks, e := startIdentity([]byte(raw), 42); e != nil || ticks != "100" {
		t.Fatal("valid proc start refused", e)
	}
	for _, bad := range []string{strings.Replace(raw, "42 ", "43 ", 1), strings.Replace(raw, " S ", " Z ", 1), "42 (helper) S 1"} {
		if _, e := startIdentity([]byte(bad), 42); e == nil {
			t.Fatal("dead/reused malformed proc accepted")
		}
	}
	if c, e := cgroupIdentity([]byte("0::/system.slice/exact.service\n"), "exact.service"); e != nil || c == "" {
		t.Fatal(e)
	}
	for _, bad := range []string{"0::/system.slice/foreign.service", "0::/system.slice/exact.service/child", "1:cpu:/system.slice/exact.service", "0::/system.slice/exact.service\n0::/foreign"} {
		if _, e := cgroupIdentity([]byte(bad), "exact.service"); e == nil {
			t.Fatal("foreign cgroup accepted")
		}
	}
	if witnessGID([]byte("Gid:\t1234 1234 1234 1234\n"), 1234) != nil {
		t.Fatal("valid gid refused")
	}
	for _, bad := range []string{"Gid: 1234 0 1234 1234", "Gid: 1234 1234", "", "Gid: 1234 1234 1234 1234\nGid: 1234 1234 1234 1234"} {
		if witnessGID([]byte(bad), 1234) == nil {
			t.Fatal("unsafe gid accepted")
		}
	}
}

func TestProcMountNamespaceIdentity(t *testing.T) {
	if mountNamespaceIdentity("mnt:[123]") != nil {
		t.Fatal("valid namespace refused")
	}
	for _, raw := range []string{"", "net:[123]", "mnt:[0]", "mnt:[0123]", "mnt:[-1]", "mnt:[123]extra"} {
		if mountNamespaceIdentity(raw) == nil {
			t.Fatal("ambiguous namespace accepted", raw)
		}
	}
}

func TestProbeRuntimeDurationRejectsUnboundedEvidence(t *testing.T) {
	for _, valid := range []string{"1h", "59min 59s", "5s", "3600000000us"} {
		if _, e := systemdWitnessDuration(valid); e != nil {
			t.Fatal(valid, e)
		}
	}
	for _, invalid := range []string{"", "infinity", "0", "0s", "-1s", "9h", "8h 1us", "5", "1h garbage"} {
		if _, e := systemdWitnessDuration(invalid); e == nil {
			t.Fatal(invalid)
		}
	}
}

func TestProbeRemainingRuntimeCapsOriginalDeadline(t *testing.T) {
	remaining, e := witnessRemainingRuntime("1000000", "1h", []byte("61.00 200.00\n"))
	if e != nil || remaining != 59*time.Minute {
		t.Fatal(remaining, e)
	}
	for _, pair := range [][3]string{{"0", "1h", "61.00 0.00"}, {"1000000", "infinity", "61.00 0.00"}, {"1000000", "1h", "3601.00 0.00"}, {"1000000", "1h", "4001.00 0.00"}, {"1000000", "1h", "0.00 0.00"}, {"01000000", "1h", "61.00 0.00"}, {"18446744073709551615", "1h", "61.00 0.00"}, {"1000000", "1h", "1e3 0.00"}, {"1000000", "1h", "-1.00 0.00"}, {"1000000", "1h", "61.00"}} {
		if _, e := witnessRemainingRuntime(pair[0], pair[1], []byte(pair[2])); e == nil {
			t.Fatal("invalid lifetime accepted", pair)
		}
	}
}

func TestProbeRemainingRuntimeNeverExtendsOriginalDeadline(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for _, limit := range []time.Duration{59 * time.Minute, time.Hour} {
		if witnessRuntimeDeadline(now, now.Add(limit), "1000000", "1h", []byte("61.00 200.00")) != nil {
			t.Fatal("bounded original expiry refused", limit)
		}
	}
	for _, limit := range []time.Duration{0, -time.Second, 59*time.Minute - time.Nanosecond} {
		if witnessRuntimeDeadline(now, now.Add(limit), "1000000", "1h", []byte("61.00 200.00")) == nil {
			t.Fatal("original expiry extended", limit)
		}
	}
}
