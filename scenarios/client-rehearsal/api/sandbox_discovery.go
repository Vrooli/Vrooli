package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type scenarioPortLookup func(context.Context, string, string) (int, error)

func resolveWorkspaceSandboxURL(ctx context.Context, configured string, lookup scenarioPortLookup) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		parsed, err := url.ParseRequestURI(configured)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return "", fmt.Errorf("configured Workspace Sandbox URL is invalid")
		}
		return configured, nil
	}
	if lookup == nil {
		return "", fmt.Errorf("Workspace Sandbox scenario port lookup is unavailable")
	}
	port, err := lookup(ctx, "workspace-sandbox", "API_PORT")
	if err != nil {
		return "", fmt.Errorf("resolve Workspace Sandbox API port: %w", err)
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("Workspace Sandbox API port is invalid")
	}
	return "http://localhost:" + strconv.Itoa(port), nil
}
