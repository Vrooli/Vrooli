package portparse

import "testing"

func TestParseSSOutput(t *testing.T) {
	t.Parallel()

	in := `LISTEN 0 4096 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=123,fd=3))
LISTEN 0 4096 0.0.0.0:443 0.0.0.0:* users:(("caddy",pid=456,fd=8))`
	ports := ParseSSOutput(in)
	if len(ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(ports))
	}
	if ports[0].Port != 22 || ports[1].Port != 443 {
		t.Fatalf("unexpected ports: %+v", ports)
	}
}

func TestExtractPIDsFromSS(t *testing.T) {
	t.Parallel()

	in := `LISTEN 0 4096 *:80 *:* users:(("nginx",pid=111,fd=6))
LISTEN 0 4096 *:443 *:* users:(("nginx",pid=111,fd=7))
LISTEN 0 4096 *:3000 *:* users:(("node",pid=222,fd=9))`
	pids := ExtractPIDsFromSS(in)
	if len(pids) != 2 {
		t.Fatalf("expected 2 unique pids, got %d (%v)", len(pids), pids)
	}
}
