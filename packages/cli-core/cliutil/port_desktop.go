package cliutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DesktopRuntimeConfig contains paths and mode, never credential bytes.
type DesktopRuntimeConfig struct {
	AppData            string
	TokenFile          string
	AuthenticationMode string
}

// ReadDesktopRuntimeConfig reads the bundle-owned IPC paths and declared identity mode.
func ReadDesktopRuntimeConfig() (DesktopRuntimeConfig, error) {
	appData, bundleRoot := os.Getenv("APP_DATA_DIR"), os.Getenv("BUNDLE_ROOT")
	if appData == "" || bundleRoot == "" {
		return DesktopRuntimeConfig{}, fmt.Errorf("desktop runtime paths are unavailable")
	}
	data, err := os.ReadFile(filepath.Join(bundleRoot, "bundle.json"))
	if err != nil {
		return DesktopRuntimeConfig{}, err
	}
	var bundle struct {
		IPC struct {
			AuthTokenPath string `json:"auth_token_path"`
		} `json:"ipc"`
		Authentication struct {
			Mode string `json:"mode"`
		} `json:"authentication"`
	}
	if err := json.Unmarshal(data, &bundle); err != nil {
		return DesktopRuntimeConfig{}, err
	}
	tokenRel := bundle.IPC.AuthTokenPath
	if tokenRel == "" {
		tokenRel = "runtime/auth-token"
	}
	if !filepath.IsLocal(tokenRel) {
		return DesktopRuntimeConfig{}, fmt.Errorf("invalid runtime token path")
	}
	return DesktopRuntimeConfig{AppData: appData, TokenFile: filepath.Join(appData, filepath.FromSlash(tokenRel)), AuthenticationMode: bundle.Authentication.Mode}, nil
}

// lookupDesktopPort calls the runtime directly. Desktop discovery must never
// fall through to the installed control plane or bypass declared peer ranges.
func lookupDesktopPort(ctx context.Context, scenario, key string) ScenarioPortOutcome {
	fail := func(err error) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: err, Output: "no running runtime ports found: " + err.Error()}
	}
	config, err := ReadDesktopRuntimeConfig()
	if err != nil {
		return fail(err)
	}
	token, err := os.ReadFile(config.TokenFile)
	if err != nil {
		return fail(err)
	}
	data, err := os.ReadFile(filepath.Join(config.AppData, "runtime", "ipc_port"))
	if err != nil {
		return fail(err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || port < 1 || port > 65535 {
		return fail(fmt.Errorf("invalid runtime IPC port"))
	}
	ctx, cancel := context.WithTimeout(ctx, portLookupTimeout)
	defer cancel()
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/scenario-port?%s", port, url.Values{"scenario": {scenario}, "port_name": {key}}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fail(err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return fail(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fail(fmt.Errorf("runtime discovery returned HTTP %d", response.StatusCode))
	}
	var result struct {
		Port int `json:"port"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&result); err != nil {
		return fail(err)
	}
	if result.Port < 1 || result.Port > 65535 {
		return fail(fmt.Errorf("runtime returned invalid scenario port"))
	}
	return ScenarioPortOutcome{Port: strconv.Itoa(result.Port)}
}
