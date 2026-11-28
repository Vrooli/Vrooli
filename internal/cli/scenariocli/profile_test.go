package scenariocli

import "testing"

func TestParseProfileRequest(t *testing.T) {
	got, err := ParseProfileRequest(false, []string{"web-console", "--instance", "shadow", "--seconds", "3", "--out", "/tmp/sample.pprof", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Scenario != "web-console@shadow" || got.Step != "start-api" || got.Seconds != 3 || !got.JSON {
		t.Fatalf("request=%+v", got)
	}
	for _, args := range [][]string{{"web-console"}, {"web-console", "--out", "x", "--seconds", "31"}, {"web-console", "--out", "x", "--seconds", "0"}, {"web-console", "--out", "x", "--seconds", "1.5"}, {"node/web-console", "--out", "x"}, {"web-console", "--node", "remote", "--out", "x"}} {
		if _, err := ParseProfileRequest(false, args); err == nil {
			t.Fatalf("invalid request accepted: %v", args)
		}
	}
}
