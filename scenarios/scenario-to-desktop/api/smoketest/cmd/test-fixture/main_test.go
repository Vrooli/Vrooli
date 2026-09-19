package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunSmokeTestFixtureReportsCompleteLifecycleAndTelemetry(t *testing.T) {
	t.Setenv("SMOKE_TEST", "1")
	t.Setenv("SMOKE_TEST_DELAY_MS", "1")
	output := captureStdout(t, func() {
		runSmokeTestFixture(0, false, false, false, false, false, true, true, "/tmp/events.json")
	})
	for _, marker := range []string{
		"SMOKE_TEST_INIT=started",
		"SMOKE_TEST_READY=true",
		"Telemetry initialized at /tmp/events.json",
		"Delaying for 1 ms",
		"SMOKE_TEST_UPLOAD=ok",
		"SMOKE_TEST_UPLOAD=error",
		"SMOKE_TEST_RESULT=passed",
		"SMOKE_TEST_EXIT=clean",
		"Test fixture completed successfully",
	} {
		if !strings.Contains(output, marker) {
			t.Fatalf("fixture output omitted %q: %s", marker, output)
		}
	}
}

func TestRunSmokeTestFixtureCanOmitCleanExitMarker(t *testing.T) {
	t.Setenv("SMOKE_TEST", "1")
	output := captureStdout(t, func() {
		runSmokeTestFixture(0, false, false, false, true, false, false, false, "")
	})
	if strings.Contains(output, "SMOKE_TEST_EXIT=clean") || !strings.Contains(output, "SMOKE_TEST_RESULT=passed") {
		t.Fatalf("unexpected no-exit output: %s", output)
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = old })
	run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return string(output)
}
