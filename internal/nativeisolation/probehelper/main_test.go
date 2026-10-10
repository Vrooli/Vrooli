package main

import (
	"testing"
	"time"
)

func TestFixedProbeArguments(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	until := now.Add(time.Hour).Format(time.RFC3339Nano)
	if _, e := deadline([]string{"--until", until}, now); e != nil {
		t.Fatal(e)
	}
	for _, args := range [][]string{nil, {"--command", "sh"}, {"--until", until, "model"}, {"--until", now.Format(time.RFC3339Nano)}, {"--until", now.Add(9 * time.Hour).Format(time.RFC3339Nano)}, {"--until", "2026-10-04T13:00:00+01:00"}} {
		if _, e := deadline(args, now); e == nil {
			t.Fatal("arbitrary/expired deadline accepted", args)
		}
	}
}
