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
	portName  string
	lines     int
	upload    bool
	uploadURL string
}

func main() {
	var cfg config
	flag.StringVar(&cfg.host, "host", "127.0.0.1", "Runtime host")
	flag.IntVar(&cfg.port, "port", 47710, "Runtime port")
	flag.StringVar(&cfg.appData, "app-data", "", "App data directory (used to find auth token)")
	flag.StringVar(&cfg.tokenFile, "token-file", "", "Path to auth token (overrides app-data lookup)")
	flag.StringVar(&cfg.serviceID, "service", "", "Service ID for log requests")
	flag.StringVar(&cfg.portName, "port-name", "http", "Port name for --port command")
	flag.IntVar(&cfg.lines, "lines", 200, "Lines for log tail")
	flag.BoolVar(&cfg.upload, "upload", false, "Upload telemetry to deployment-manager (requires upload URL)")
	flag.StringVar(&cfg.uploadURL, "upload-url", "", "Override telemetry upload URL")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("usage: runtimectl [flags] <health|ready|ports|status|port|logs|telemetry|shutdown>")
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
		callJSON(cfg, token, "/healthz")
	case "ready":
		callJSON(cfg, token, "/readyz")
	case "ports":
		callJSON(cfg, token, "/ports")
	case "status":
		if err := showStatus(cfg, token); err != nil {
			fail(err)
		}
	case "port":
		if err := showPort(cfg, token); err != nil {
			fail(err)
		}
	case "logs":
		if cfg.serviceID == "" {
			fmt.Fprintln(os.Stderr, "--service is required for logs")
			os.Exit(1)
		}
		path := fmt.Sprintf("/logs/tail?serviceId=%s&lines=%d", cfg.serviceID, cfg.lines)
		callText(cfg, token, path)
	case "telemetry":
		if err := handleTelemetry(cfg, token); err != nil {
			fail(err)
		}
	case "shutdown":
		callJSON(cfg, token, "/shutdown")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cfg.command)
		os.Exit(1)
	}
}

func showStatus(cfg config, token string) error {
	var ready struct {
		Ready   bool                      `json:"ready"`
		Details map[string]map[string]any `json:"details"`
	}
	if err := fetchJSON(cfg, token, "/readyz", &ready); err != nil {
		return err
	}

	var ports struct {
		Services map[string]map[string]int `json:"services"`
	}
	_ = fetchJSON(cfg, token, "/ports", &ports)

	status := map[string]interface{}{
		"ready":   ready.Ready,
		"details": ready.Details,
	}
	if len(ports.Services) > 0 {
		status["ports"] = ports.Services
	}
	pretty, _ := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(pretty))
	return nil
}

func showPort(cfg config, token string) error {
	if cfg.serviceID == "" {
		return fmt.Errorf("--service is required for port lookup")
	}
	var ports struct {
		Services map[string]map[string]int `json:"services"`
	}
	if err := fetchJSON(cfg, token, "/ports", &ports); err != nil {
		return err
	}
	servicePorts, ok := ports.Services[cfg.serviceID]
	if !ok {
		return fmt.Errorf("service %q has no allocated ports", cfg.serviceID)
	}
	port, ok := servicePorts[cfg.portName]
	if !ok {
		return fmt.Errorf("port %q not found for service %q", cfg.portName, cfg.serviceID)
	}
	fmt.Println(port)
	return nil
}

func handleTelemetry(cfg config, token string) error {
	var info struct {
		Path      string `json:"path"`
		UploadURL string `json:"upload_url"`
	}
	if err := fetchJSON(cfg, token, "/telemetry", &info); err != nil {
		return err
	}
	if info.Path == "" {
		return fmt.Errorf("runtime did not report telemetry path")
	}
	fmt.Printf("telemetry file: %s\n", info.Path)

	uploadURL := cfg.uploadURL
	if uploadURL == "" {
		uploadURL = info.UploadURL
	}
	if !cfg.upload {
		if uploadURL != "" {
			fmt.Printf("upload url available: %s (pass --upload to send now)\n", uploadURL)
		}
		return nil
	}
	if uploadURL == "" {
		return fmt.Errorf("no upload URL available (use --upload-url)")
	}

	data, err := os.ReadFile(info.Path)
	if err != nil {
		return fmt.Errorf("read telemetry: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, uploadURL, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/jsonl")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload telemetry: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	fmt.Println("upload successful")
	return nil
}

func callJSON(cfg config, token, path string) {
	var out map[string]any
	if err := fetchJSON(cfg, token, path, &out); err != nil {
		fail(err)
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(pretty))
}

func callText(cfg config, token, path string) {
	body, err := fetchText(cfg, token, path)
	if err != nil {
		fail(err)
	}
	fmt.Println(body)
}

func fetchJSON(cfg config, token, path string, out interface{}) error {
	body, err := doRequest(cfg, token, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func fetchText(cfg config, token, path string) (string, error) {
	body, err := doRequest(cfg, token, path)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func doRequest(cfg config, token, path string) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%d%s", cfg.host, cfg.port, path)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" && path != "/healthz" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
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

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
