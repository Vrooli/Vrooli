// Vrooli Autoheal Loop - cross-platform boot recovery watchdog.
//
// The loop keeps the vrooli-autoheal scenario alive from a native scheduler
// unit and is the one component that must keep working when the scenario it
// supervises cannot build. See README.md for the state machine, the preflight
// checks, the status file, and the exit codes.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	opts, err := parseFlags(args)
	if err != nil {
		log.Printf("%v", err)
		return exitNoRoot
	}
	if opts.installTarget != "" {
		if err := installExecutable(opts.installTarget); err != nil {
			log.Printf("failed to install executable: %v", err)
			return 1
		}
		return 0
	}

	config := opts.config
	config.VrooliRoot = resolveVrooliRoot()
	config.resolveVrooliBinary(opts.vrooliBin)

	// One context for the whole process: every sleep, wait and child
	// invocation hangs off it, so SIGTERM cancels a lifecycle command in
	// flight instead of waiting behind it.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if opts.selfTest {
		result := Preflight(ctx, config)
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Printf("encode preflight result: %v", err)
		}
		if result.OK {
			return 0
		}
		return exitNonHealable
	}

	if config.VrooliRoot == "" {
		log.Printf("failed to resolve the repository root (VROOLI_SOURCE_ROOT, VROOLI_ROOT, working directory, executable path)")
		return exitNoRoot
	}
	status, err := newStatusWriter()
	if err != nil {
		log.Printf("state directory unwritable: %v", err)
		return exitStateUnwritable
	}

	log.Printf("Vrooli Autoheal Loop starting")
	log.Printf("  VROOLI_ROOT: %s", config.VrooliRoot)
	log.Printf("  vrooli: %s", config.VrooliCmdPath)
	log.Printf("  interval: %v, max tick failures: %d, lifecycle management: %v", config.TickInterval, config.MaxFailures, config.ManageAPILifecycle)
	log.Printf("  status file: %s", status.path)

	return newLoop(config, status).run(ctx)
}
