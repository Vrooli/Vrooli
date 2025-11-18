package main

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeProxyURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("proxy URL cannot be empty")
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "http://") && !strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid proxy URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("proxy URL must start with http:// or https://")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("proxy URL must include a hostname")
	}
	parsed.Fragment = ""
	parsed.RawQuery = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed.String(), nil
}

func proxyAPIURL(proxy string) string {
	if proxy == "" {
		return ""
	}
	if strings.HasSuffix(proxy, "/") {
		return proxy + "api"
	}
	return proxy + "/api"
}
