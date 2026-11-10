package androidadb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

// AttachWebView discovers the debugging socket owned by packageName and asks
// adb to allocate a local TCP forward. This is intentionally implemented at
// the Android strategy boundary: callers above device-control never execute
// adb and never need to know the device's abstract socket naming convention.
func (a *Adapter) AttachWebView(ctx context.Context, packageName string) (strategy.WebViewEndpoint, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRequestValidation, Err: fmt.Errorf("webview attach requires an application package")}
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.]*$`).MatchString(packageName) {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRequestValidation, Err: fmt.Errorf("webview attach rejected invalid package %q", packageName)}
	}
	socket, err := a.webViewSocket(ctx, packageName)
	if err != nil {
		return strategy.WebViewEndpoint{}, err
	}
	forward := "localabstract:" + socket
	allocated, err := a.runner.Run(ctx, "adb", a.args("forward", "tcp:0", forward)...)
	if err != nil {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStagePortForward, Err: fmt.Errorf("forward WebView socket %q: %w", socket, err)}
	}
	port, err := forwardedPort(allocated, socket)
	if err != nil {
		listing, listErr := a.runner.Run(ctx, "adb", a.args("forward", "--list")...)
		if listErr == nil {
			port, err = forwardedPort(listing, socket)
		}
	}
	if err != nil {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageForwardReadback, Err: err}
	}
	cdpEndpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	var rendererID, rendererURL string
	if a.rendererID != nil {
		rendererID, err = a.rendererID(ctx, cdpEndpoint)
	} else {
		readRenderer := a.rendererTarget
		if readRenderer == nil {
			readRenderer = discoverWebViewRenderer
		}
		rendererID, rendererURL, err = readRenderer(ctx, cdpEndpoint)
	}
	if err != nil {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRendererDiscovery, Err: fmt.Errorf("discover WebView renderer for %q: %w", packageName, err)}
	}
	if strings.TrimSpace(rendererID) == "" || (a.rendererID == nil && strings.TrimSpace(rendererURL) == "") {
		return strategy.WebViewEndpoint{}, &strategy.WebViewAttachFailure{Stage: strategy.WebViewStageRendererIdentity, Err: fmt.Errorf("discover WebView renderer for %q returned incomplete identity", packageName)}
	}
	return strategy.WebViewEndpoint{Package: packageName, Socket: socket, CDPEndpoint: cdpEndpoint, RendererID: rendererID, RendererURL: rendererURL, Transport: "adb-forward"}, nil
}

func (a *Adapter) webViewSocket(ctx context.Context, packageName string) (string, error) {
	var lastErr error
	lastStage := strategy.WebViewStageProcessLookup
	for attempt := 0; attempt < 20; attempt++ {
		pids, err := a.runner.Run(ctx, "adb", a.args("shell", "pidof", packageName)...)
		if err != nil {
			lastStage = strategy.WebViewStageProcessLookup
			lastErr = fmt.Errorf("find WebView process for %q: %w", packageName, err)
		} else {
			pid := strings.Fields(string(pids))
			if len(pid) > 0 && regexp.MustCompile(`^\d+$`).MatchString(pid[0]) {
				sockets, socketErr := a.runner.Run(ctx, "adb", a.args("shell", "cat", "/proc/net/unix")...)
				if socketErr != nil {
					lastStage = strategy.WebViewStageSocketTableRead
					lastErr = fmt.Errorf("list Android WebView sockets: %w", socketErr)
				} else if socket, socketErr := webViewSocketForPID(sockets, pid[0]); socketErr == nil {
					return socket, nil
				} else {
					lastStage = strategy.WebViewStageSocketDiscovery
					lastErr = fmt.Errorf("find WebView socket for %q: %w", packageName, socketErr)
				}
			} else {
				lastStage = strategy.WebViewStageProcessLookup
				lastErr = fmt.Errorf("application %q has no running process", packageName)
			}
		}
		if attempt == 19 {
			break
		}
		select {
		case <-ctx.Done():
			return "", &strategy.WebViewAttachFailure{Stage: lastStage, Err: ctx.Err()}
		case <-time.After(250 * time.Millisecond):
		}
	}
	return "", &strategy.WebViewAttachFailure{Stage: lastStage, Err: lastErr}
}

func discoverWebViewRenderer(ctx context.Context, endpoint string) (string, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/json/list", nil)
	if err != nil {
		return "", "", err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("CDP target listing returned %s", response.Status)
	}
	var targets []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&targets); err != nil {
		return "", "", fmt.Errorf("decode CDP target listing: %w", err)
	}
	for _, target := range targets {
		if strings.EqualFold(target.Type, "page") && strings.TrimSpace(target.ID) != "" {
			if strings.TrimSpace(target.URL) == "" {
				return "", "", fmt.Errorf("CDP page renderer %q omitted its URL", target.ID)
			}
			return strings.TrimSpace(target.ID), strings.TrimSpace(target.URL), nil
		}
	}
	return "", "", fmt.Errorf("CDP target listing contained no page renderer")
}

func webViewSocketForPID(data []byte, pid string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "@")
		if strings.HasPrefix(name, "webview_devtools_remote_") && strings.Contains(name, "_"+pid) {
			return name, nil
		}
	}
	return "", fmt.Errorf("no webview_devtools_remote socket matched process %s", pid)
}

func forwardedPort(output []byte, socket string) (int, error) {
	remote := "localabstract:" + socket
	portPattern := regexp.MustCompile(`(?:^|\s)(?:tcp:)?([1-9][0-9]{1,4})(?:\s|$)`)
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, remote) {
			fields := strings.Fields(line)
			for _, field := range fields {
				if match := portPattern.FindStringSubmatch(" " + field + " "); len(match) == 2 {
					port, _ := strconv.Atoi(match[1])
					if port > 0 && port <= 65535 {
						return port, nil
					}
				}
			}
		}
	}
	if match := portPattern.FindStringSubmatch(" " + strings.TrimSpace(string(output)) + " "); len(match) == 2 {
		port, _ := strconv.Atoi(match[1])
		if port > 0 && port <= 65535 {
			return port, nil
		}
	}
	// Some adb versions do not print the allocated port for tcp:0. The list
	// command is the authoritative fallback and keeps the allocation owned by
	// this adapter.
	return 0, fmt.Errorf("adb did not report a local port for WebView socket %q", socket)
}
