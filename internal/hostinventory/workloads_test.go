package hostinventory

import (
	"context"
	"testing"
)

func TestParseSS(t *testing.T) {
	ports := parseSS([]byte("State Recv-Q Send-Q Local Address:Port Peer Address:Port Process\nLISTEN 0 128 127.0.0.1:8080 0.0.0.0:* users:((\"api\",pid=42))\n"))
	if len(ports) != 1 || ports[0].Port != 8080 || ports[0].Process == "" {
		t.Fatalf("unexpected listeners: %#v", ports)
	}
}

func TestParseWindowsTaskCSV(t *testing.T) {
	tasks, err := ParseWindowsTaskCSV([]byte("\"\\Task\\Vrooli\",\"Vrooli\",\"Running\"\n"))
	if err != nil || len(tasks) != 1 || tasks[0].Kind != "scheduled-task" || !tasks[0].Running {
		t.Fatalf("unexpected tasks: %#v, %v", tasks, err)
	}
}

func TestParseLaunchctlAndDarwinNetstat(t *testing.T) {
	units := parseLaunchctl([]byte("PID\tStatus\tLabel\n42\t0\tcom.vrooli.autoheal\n-\t0\tcom.example.stopped\n"))
	if len(units) != 2 || !units[0].Running || units[1].Running {
		t.Fatalf("unexpected launchctl units: %#v", units)
	}
	ports := parseNetstat([]byte("tcp4 0 0 127.0.0.1.17573 *.* LISTEN\n"), "darwin")
	if len(ports) != 1 || ports[0].Port != 17573 {
		t.Fatalf("unexpected darwin listeners: %#v", ports)
	}
}

func TestParseSCQueryAndWindowsNetstat(t *testing.T) {
	units := parseSCQuery([]byte("SERVICE_NAME: VrooliAutoheal\n        STATE              : 4  RUNNING\nSERVICE_NAME: stopped\n        STATE              : 1  STOPPED\n"))
	if len(units) != 2 || !units[0].Running || units[1].Running {
		t.Fatalf("unexpected sc query units: %#v", units)
	}
	ports := parseNetstat([]byte("TCP    0.0.0.0:17573    0.0.0.0:0    LISTENING    1234\n"), "windows")
	if len(ports) != 1 || ports[0].Port != 17573 || ports[0].Process != "1234" {
		t.Fatalf("unexpected windows listeners: %#v", ports)
	}
}

func TestCollectWorkloadsIncludesRestartEvidenceAndUnreadStates(t *testing.T) {
	docker := "/usr/bin/docker"
	c := Collector{GOOS: "linux", Commands: fakeCommandRunner{
		paths: map[string]string{"docker": docker, "systemctl": "/usr/bin/systemctl", "ss": "/usr/bin/ss"},
		out: map[string][]byte{
			docker + " ps -a --format {{json .}}": []byte(`{"Names":"airbyte-abctl-control-plane","Image":"kindest/node:v1.32.2","State":"running"}
`),
			docker + " inspect --format {{.Name}}\t{{.RestartCount}} airbyte-abctl-control-plane": []byte("/airbyte-abctl-control-plane\t191985\n"),
			"/usr/bin/systemctl --user list-units --all --no-legend --no-pager":                   []byte("vrooli-emergency-watchdog.service loaded active running watchdog\n"),
			"/usr/bin/systemctl list-units --all --no-legend --no-pager":                          []byte("docker.service loaded active running docker\n"),
			"/usr/bin/ss -ltnup": []byte("LISTEN 0 128 127.0.0.1:8080 0.0.0.0:* users:((\"api\",pid=42))\n"),
		},
	}}
	got, err := c.CollectWorkloads(context.Background())
	if err != nil || len(got.Containers) != 1 || got.Containers[0].RestartCount != 191985 || len(got.ServiceUnits) != 2 || len(got.Listening) != 1 {
		t.Fatalf("unexpected workload census: %#v, %v", got, err)
	}
}
