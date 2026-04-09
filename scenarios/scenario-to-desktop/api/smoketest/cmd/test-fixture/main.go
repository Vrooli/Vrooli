// Package main provides a test fixture application for smoke test integration testing.
// This binary mimics the behavior of a real desktop app under smoke test conditions.
//
// Usage:
//
//	test-fixture --smoke-test [options]
//
// Options:
//
//	--smoke-test      Required. Enable smoke test mode.
//	--delay=<ms>      Delay before outputting success marker (for timeout testing).
//	--fail-init       Fail during initialization (don't output SMOKE_TEST_INIT).
//	--fail-ready      Output init but fail before ready marker.
//	--fail-result     Output init and ready but not the success result.
//	--no-exit         Don't output clean exit marker.
//	--crash           Panic after outputting init marker.
//	--upload-success  Simulate successful telemetry upload.
//	--upload-error    Simulate failed telemetry upload.
//	--telemetry-path=<path>  Output telemetry path for fallback testing.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	// Parse flags
	smokeTest := flag.Bool("smoke-test", false, "Enable smoke test mode")
	delay := flag.Int("delay", 0, "Delay in milliseconds before success output")
	failInit := flag.Bool("fail-init", false, "Fail during initialization")
	failReady := flag.Bool("fail-ready", false, "Fail before ready marker")
	failResult := flag.Bool("fail-result", false, "Fail before result marker")
	noExit := flag.Bool("no-exit", false, "Don't output exit marker")
	crash := flag.Bool("crash", false, "Panic after init")
	uploadSuccess := flag.Bool("upload-success", false, "Simulate telemetry upload success")
	uploadError := flag.Bool("upload-error", false, "Simulate telemetry upload error")
	telemetryPath := flag.String("telemetry-path", "", "Output telemetry path")
	flag.Parse()

	// Check for smoke test mode
	if !*smokeTest {
		fmt.Println("Test fixture - run with --smoke-test flag")
		os.Exit(0)
	}

	runSmokeTestFixture(*delay, *failInit, *failReady, *failResult, *noExit, *crash, *uploadSuccess, *uploadError, *telemetryPath)
}

func runSmokeTestFixture(delay int, failInit, failReady, failResult, noExit, crash, uploadSuccess, uploadError bool, telemetryPath string) {
	// Check environment
	if os.Getenv("SMOKE_TEST") != "1" {
		fmt.Println("Warning: SMOKE_TEST environment variable not set to 1")
	}

	// Check for environment-based delay override
	if envDelay := os.Getenv("SMOKE_TEST_DELAY_MS"); envDelay != "" {
		if d, err := strconv.Atoi(envDelay); err == nil && d > 0 {
			delay = d
		}
	}

	if os.Getenv("SMOKE_TEST_STDERR") != "" {
		fmt.Fprintln(os.Stderr, "STDERR: Test fixture stderr output")
		fmt.Fprintln(os.Stderr, "STDERR: This should appear in stderr logs")
	}

	if failInit {
		fmt.Println("Failing during initialization...")
		os.Exit(1)
	}

	fmt.Println("SMOKE_TEST_INIT=started")

	if crash {
		fmt.Println("About to crash...")
		panic("simulated crash for testing")
	}

	if failReady {
		fmt.Println("Failing before ready...")
		os.Exit(1)
	}

	fmt.Println("SMOKE_TEST_READY=true")

	if telemetryPath != "" {
		fmt.Printf("[Desktop App] Telemetry initialized at %s\n", telemetryPath)
	}

	if delay > 0 {
		fmt.Printf("Delaying for %d ms...\n", delay)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	if failResult {
		fmt.Println("Failing before result marker...")
		os.Exit(0)
	}

	if uploadSuccess {
		fmt.Println("SMOKE_TEST_UPLOAD=ok")
	}
	if uploadError {
		fmt.Println("SMOKE_TEST_UPLOAD=error")
	}

	fmt.Println("SMOKE_TEST_RESULT=passed")

	if !noExit {
		fmt.Println("SMOKE_TEST_EXIT=clean")
	}

	fmt.Println("Test fixture completed successfully")
}
