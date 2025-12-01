package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	host      string
	port      int
	appData   string
	tokenFile string
	command   string
	serviceID string
	lines     int
}

func main() {
	var cfg config
	flag.StringVar(&cfg.host, "host", "127.0.0.1", "Runtime host")
	flag.IntVar(&cfg.port, "port", 47710, "Runtime port")
	flag.StringVar(&cfg.appData, "app-data", "", "App data directory (used to find auth token)")
	flag.StringVar(&cfg.tokenFile, "token-file", "", "Path to auth token (overrides app-data lookup)")
	flag.StringVar(&cfg.serviceID, "service", "", "Service ID for log requests")
	flag.IntVar(&cfg.lines, "lines", 200, "Lines for log tail")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("usage: runtimectl [flags] <health|ports|ready|logs|shutdown>")
		os.Exit(1)
	}
	cfg.command = args[0]

	token, err := resolveToken(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve token: %v\n", err)
		os.Exit(1)
	}

	switch cfg.command {
	case "health":
		call(cfg, token, "/healthz")
	case "ready":
		call(cfg, token, "/readyz")
	case "ports":
		call(cfg, token, "/ports")
	case "logs":
		if cfg.serviceID == "" {
			fmt.Fprintln(os.Stderr, "--service is required for logs")
			os.Exit(1)
		}
		path := fmt.Sprintf("/logs/tail?serviceId=%s&lines=%d", cfg.serviceID, cfg.lines)
		call(cfg, token, path)
	case "shutdown":
		call(cfg, token, "/shutdown")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cfg.command)
		os.Exit(1)
	}
}

func call(cfg config, token, path string) {
	url := fmt.Sprintf("http://%s:%d%s", cfg.host, cfg.port, path)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" && path != "/healthz" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var out map[string]interface{}
		if err := json.Unmarshal(body, &out); err == nil {
			pretty, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(pretty))
			return
		}
	}
	fmt.Println(string(body))
}

func resolveToken(cfg config) (string, error) {
	if cfg.tokenFile != "" {
		data, err := os.ReadFile(cfg.tokenFile)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	if cfg.appData == "" {
		return "", nil
	}
	path := filepath.Join(cfg.appData, "runtime", "auth-token")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(data)), nil
}
