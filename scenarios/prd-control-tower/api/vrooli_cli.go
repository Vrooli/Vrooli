package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func resolveScenarioPortViaCLI(ctx context.Context, scenarioName, portLabel string) (int, error) {
	port, err := executeScenarioPortCommand(ctx, scenarioName, portLabel)
	if err == nil && port > 0 {
		return port, nil
	}

	fallbackPorts, fallbackErr := executeScenarioPortList(ctx, scenarioName)
	if fallbackErr == nil {
		candidate := strings.ToUpper(strings.TrimSpace(portLabel))
		if value, ok := fallbackPorts[candidate]; ok && value > 0 {
			return value, nil
		}
	}

	if err != nil {
		if fallbackErr != nil {
			return 0, fmt.Errorf("%v; fallback error: %w", err, fallbackErr)
		}
		return 0, err
	}

	return 0, errors.New("port not found")
}

func executeScenarioPortCommand(ctx context.Context, scenarioName, portLabel string) (int, error) {
	if isEmptyOrWhitespace(scenarioName) || isEmptyOrWhitespace(portLabel) {
		return 0, errors.New("scenario and port labels are required")
	}

	output, err := executeVrooliCommand(ctx, 10*time.Second, "scenario", "port", scenarioName, portLabel)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve %s port for %s: %w", portLabel, scenarioName, err)
	}

	return parsePortFromAny(strings.TrimSpace(string(output)))
}

func executeScenarioPortList(ctx context.Context, scenarioName string) (map[string]int, error) {
	if isEmptyOrWhitespace(scenarioName) {
		return nil, errors.New("scenario name is required")
	}

	output, err := executeVrooliCommand(ctx, 10*time.Second, "scenario", "port", scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to list ports for %s: %w", scenarioName, err)
	}

	ports := make(map[string]int)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		port, err := parsePortFromAny(value)
		if err != nil {
			continue
		}

		if port > 0 {
			ports[key] = port
		}
	}

	if len(ports) == 0 {
		return nil, errors.New("no port mappings returned")
	}

	return ports, nil
}

func executeVrooliCommand(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", args...)
	cmd.Dir = vrooliRoot
	return cmd.CombinedOutput()
}

func parsePortFromAny(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "\"")
	if trimmed == "" {
		return 0, errors.New("empty port value")
	}
	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func isEmptyOrWhitespace(input string) bool {
	return strings.TrimSpace(input) == ""
}
