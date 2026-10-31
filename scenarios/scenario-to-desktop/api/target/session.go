package target

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultElectronStartupTimeout  = 60 * time.Second
	defaultElectronShutdownTimeout = 5 * time.Second
)

// ElectronProcess is the small lifecycle boundary owned by the target
// adapter. Keeping it injectable makes process-tree and cleanup behavior
// testable without replacing the production launch path.
type ElectronProcess interface {
	PID() int
	Signal(os.Signal) error
	Kill() error
	Wait() error
}

// ProcessStarter starts one exact artifact with one argv/env vector. It must
// not invoke a shell.
type ProcessStarter func(context.Context, string, []string, []string) (ElectronProcess, error)

// ElectronSessionOptions describes one validation-owned application launch.
// The identity fields are mandatory because a renderer endpoint without cell
// provenance is not safe to attach to.
type ElectronSessionOptions struct {
	ArtifactPath string
	Launch       ElectronLaunchOptions
	Renderer     RendererExpectation
	Environment  map[string]string
	// OfflineNetwork isolates the full packaged application process tree from
	// external networking. Unsupported platforms fail closed.
	OfflineNetwork bool

	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	StartProcess    ProcessStarter
	networkRoot     string
}

// ElectronSession owns the launched artifact, CDP target, and temporary
// profile. Call Close exactly once when the cell ends; it is idempotent and
// reports cleanup failures rather than hiding them.
type ElectronSession struct {
	lifecycleMu      sync.Mutex
	options          ElectronSessionOptions
	closed           bool
	process          ElectronProcess
	waitDone         chan struct{}
	waitMu           sync.Mutex
	waitErr          error
	target           *domainv1.AppTarget
	endpoint         string
	profileDir       string
	ownedProfile     bool
	networkRoot      string
	ownedNetworkRoot bool
	cdpBridge        *unixCDPBridge
	apiBridge        *unixCDPBridge
	shutdownLimit    time.Duration

	closeOnce sync.Once
	closeErr  error
}

// StartElectronSession launches the selected artifact, waits for a controlled
// loopback CDP endpoint, and selects a unique attachable renderer. It cleans
// up all resources when startup fails.
func StartElectronSession(ctx context.Context, opts ElectronSessionOptions) (*ElectronSession, error) {
	if err := validateArtifactPath(opts.ArtifactPath); err != nil {
		return nil, err
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = defaultElectronStartupTimeout
	}
	if opts.ShutdownTimeout <= 0 {
		opts.ShutdownTimeout = defaultElectronShutdownTimeout
	}
	ownedNetworkRoot := false
	if opts.OfflineNetwork {
		root, created, err := prepareOfflineNetworkRoot(opts.networkRoot)
		if err != nil {
			return nil, err
		}
		opts.networkRoot = root
		ownedNetworkRoot = created
		if opts.Launch.UserDataDir == "" {
			opts.Launch.UserDataDir = filepath.Join(root, "profile")
		}
		if !pathWithin(root, opts.Launch.UserDataDir) {
			if created {
				_ = os.RemoveAll(root)
			}
			return nil, fmt.Errorf("offline Electron profile must stay inside its private session root")
		}
		if err := os.MkdirAll(opts.Launch.UserDataDir, 0o700); err != nil {
			if created {
				_ = os.RemoveAll(root)
			}
			return nil, fmt.Errorf("create offline Electron profile: %w", err)
		}
		if opts.Launch.BundledRuntime {
			if opts.Environment == nil {
				opts.Environment = map[string]string{}
			}
			descriptorPath := filepath.Join(root, "bundled-target.json")
			if err := os.Remove(descriptorPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				if created {
					_ = os.RemoveAll(root)
				}
				return nil, fmt.Errorf("clear previous bundled target descriptor: %w", err)
			}
			opts.Environment["S2D_BUNDLED_TARGET_PATH"] = descriptorPath
		}
	} else if opts.Launch.CDPPort == 0 {
		port, err := allocateLoopbackPort()
		if err != nil {
			return nil, fmt.Errorf("allocate Electron CDP port: %w", err)
		}
		opts.Launch.CDPPort = port
	}
	ownedProfile := false
	if opts.OfflineNetwork {
		// The private session root owns the profile and its lifecycle.
	} else if opts.Launch.UserDataDir == "" {
		profile, err := os.MkdirTemp("", "vrooli-electron-validation-")
		if err != nil {
			return nil, fmt.Errorf("create isolated Electron profile: %w", err)
		}
		opts.Launch.UserDataDir = profile
		ownedProfile = true
	} else if err := os.MkdirAll(opts.Launch.UserDataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create Electron profile: %w", err)
	}

	var cdpBridge *unixCDPBridge
	if opts.OfflineNetwork {
		bridge, bridgeErr := startUnixCDPBridge(filepath.Join(opts.networkRoot, "cdp.sock"), opts.Launch.CDPPort)
		if bridgeErr != nil {
			if ownedNetworkRoot {
				_ = os.RemoveAll(opts.networkRoot)
			}
			return nil, fmt.Errorf("start private Electron CDP bridge: %w", bridgeErr)
		}
		cdpBridge = bridge
		opts.Launch.CDPPort = cdpBridge.Port()
	}

	args, scopedEnv, err := BuildElectronLaunchArguments(opts.Launch)
	if err != nil {
		if cdpBridge != nil {
			_ = cdpBridge.Close()
		}
		if ownedNetworkRoot {
			_ = os.RemoveAll(opts.networkRoot)
		}
		if ownedProfile {
			_ = os.RemoveAll(opts.Launch.UserDataDir)
		}
		return nil, err
	}
	env := append([]string(nil), os.Environ()...)
	for key, value := range opts.Environment {
		env = append(env, key+"="+value)
	}
	for key, value := range scopedEnv {
		env = append(env, key+"="+value)
	}
	if opts.Launch.BundledRuntime {
		env = withoutBundledValidationEndpointOverrides(env)
	}
	if opts.OfflineNetwork && opts.Launch.BundledRuntime {
		descriptorPath := filepath.Join(opts.networkRoot, "bundled-target.json")
		env = withoutEnvironmentKey(env, "S2D_BUNDLED_TARGET_PATH")
		env = append(env, "S2D_BUNDLED_TARGET_PATH="+descriptorPath)
	}
	// AppImage's normal launch path attempts a FUSE mount. Validation runs in
	// managed/headless environments where FUSE is commonly unavailable; use
	// the AppImage-supported extraction mode instead. This is an environment
	// toggle, not a shell wrapper, so the artifact still receives the exact
	// validation argv and remains owned by this session.
	if strings.EqualFold(filepath.Ext(opts.ArtifactPath), ".AppImage") {
		env = append(env, "APPIMAGE_EXTRACT_AND_RUN=1")
	}
	starter := opts.StartProcess
	if starter == nil {
		starter = startExecProcess
	}
	processPath, processArgs := opts.ArtifactPath, args
	if opts.OfflineNetwork {
		processPath, processArgs, err = isolatedElectronCommand(opts.networkRoot, opts.Launch.UserDataDir, cdpBridge.socketPath, filepath.Join(opts.networkRoot, "renderer.sock"), opts.Launch.CDPPort, opts.ArtifactPath, args)
		if err != nil {
			_ = cdpBridge.Close()
			if ownedNetworkRoot {
				_ = os.RemoveAll(opts.networkRoot)
			}
			return nil, err
		}
	}
	process, err := starter(ctx, processPath, processArgs, env)
	if err != nil {
		if cdpBridge != nil {
			_ = cdpBridge.Close()
		}
		if ownedNetworkRoot {
			_ = os.RemoveAll(opts.networkRoot)
		}
		if ownedProfile {
			_ = os.RemoveAll(opts.Launch.UserDataDir)
		}
		return nil, fmt.Errorf("launch Electron artifact: %w", err)
	}
	session := &ElectronSession{
		options:          opts,
		process:          process,
		waitDone:         make(chan struct{}),
		endpoint:         fmt.Sprintf("http://127.0.0.1:%d", opts.Launch.CDPPort),
		profileDir:       opts.Launch.UserDataDir,
		ownedProfile:     ownedProfile,
		networkRoot:      opts.networkRoot,
		ownedNetworkRoot: ownedNetworkRoot,
		cdpBridge:        cdpBridge,
		shutdownLimit:    opts.ShutdownTimeout,
	}
	if cdpBridge != nil {
		session.endpoint = cdpBridge.Endpoint()
	}
	go func() {
		waitErr := process.Wait()
		session.waitMu.Lock()
		session.waitErr = waitErr
		session.waitMu.Unlock()
		close(session.waitDone)
	}()
	fail := func(cause error) (*ElectronSession, error) {
		if cleanupErr := session.Close(context.Background()); cleanupErr != nil {
			return nil, errors.Join(cause, fmt.Errorf("clean up Electron startup: %w", cleanupErr))
		}
		return nil, cause
	}

	deadline := time.NewTimer(opts.StartupTimeout)
	defer deadline.Stop()
	var lastPages []CDPPage
	for {
		pages, discoverErr := DiscoverElectronRenderers(ctx, session.endpoint)
		if discoverErr == nil {
			lastPages = append(lastPages[:0], pages...)
			page, selectErr := SelectElectronRenderer(pages, opts.Renderer)
			if selectErr == nil {
				var apiBridge *unixCDPBridge
				apiURL := ""
				testDatabaseRoot := ""
				rendererURL := page.URL
				if opts.OfflineNetwork && opts.Launch.BundledRuntime {
					descriptorPath := filepath.Join(opts.networkRoot, "bundled-target.json")
					apiPort, descriptorErr := offlineBundledAPIPort(descriptorPath, opts.networkRoot, page.URL)
					if descriptorErr != nil {
						return fail(descriptorErr)
					}
					// The host-facing API bridge uses an ephemeral loopback port. The
					// namespace relay still targets Electron's exact private API port.
					apiBridge, err = startUnixPortBridge(filepath.Join(opts.networkRoot, "renderer.sock"), 0, apiPort)
					if err != nil {
						return fail(fmt.Errorf("start private offline API bridge: %w", err))
					}
					apiURL = apiBridge.Endpoint()
					session.apiBridge = apiBridge
					testDatabaseRoot = filepath.Join(opts.networkRoot, "test-database")
					if err := validatePrivateTestDatabaseRoot(testDatabaseRoot, opts.networkRoot); err != nil {
						return fail(err)
					}
					if err := verifyOfflineBundledAPI(apiURL, apiPort, filepath.Join(opts.networkRoot, "renderer.sock.errors")); err != nil {
						return fail(fmt.Errorf("private bundled API readiness check failed: %w", err))
					}
				}
				target := &domainv1.AppTarget{
					TargetId:         opts.Launch.TargetID,
					ScenarioName:     opts.Launch.ScenarioName,
					ArtifactDigest:   opts.Launch.ArtifactDigest,
					ContextId:        opts.Launch.ContextID,
					ProcessId:        fmt.Sprintf("%d", process.PID()),
					CdpEndpoint:      session.endpoint,
					CdpTransport:     "loopback-authenticated",
					LoopbackOnly:     true,
					Authenticated:    true,
					RendererId:       page.ID,
					RendererUrl:      rendererURL,
					RendererTitle:    &page.Title,
					ApiUrl:           apiURL,
					TestDatabaseRoot: testDatabaseRoot,
					LaunchedAt:       timestamppb.Now(),
				}
				if err := ValidateAppTarget(target); err != nil {
					return fail(fmt.Errorf("validate launched Electron target: %w", err))
				}
				session.apiBridge = apiBridge
				session.target = target
				return session, nil
			}
			if rendererMayStillBeStarting(opts.Renderer, selectErr) {
				discoverErr = selectErr
			} else {
				// A discovered but ambiguous target is a safety failure, not a
				// readiness condition. Do not attach to a guessed renderer.
				return fail(fmt.Errorf("select Electron renderer: %w", selectErr))
			}
		}

		select {
		case <-session.waitDone:
			return fail(fmt.Errorf("Electron artifact exited before CDP became ready: %w", session.waitError()))
		case <-ctx.Done():
			return fail(ctx.Err())
		case <-deadline.C:
			return fail(fmt.Errorf("Electron CDP endpoint did not become ready: %w (%s)", discoverErr, summarizeRendererPages(lastPages)))
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func offlineRendererPort(rawURL string) (int, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u == nil || u.Scheme != "http" || u.User != nil || u.Opaque != "" {
		return 0, fmt.Errorf("offline renderer bridge requires an HTTP loopback renderer URL")
	}
	ip := net.ParseIP(u.Hostname())
	port, err := strconv.Atoi(u.Port())
	if ip == nil || !ip.IsLoopback() || ip.To4() == nil || err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("offline renderer bridge requires an explicit IPv4 loopback port")
	}
	return port, nil
}

func verifyOfflineBundledAPI(endpoint string, privatePort int, relayErrorsPath string) error {
	return verifyOfflineBundledAPIWithin(endpoint, privatePort, relayErrorsPath, 10*time.Second)
}

func verifyOfflineBundledAPIWithin(endpoint string, privatePort int, relayErrorsPath string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	var lastStatus int
	var lastBody string
	var lastErr error
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/health", nil)
		if err != nil {
			return fmt.Errorf("construct loopback health probe: %w", err)
		}
		response, requestErr := client.Do(request)
		lastErr = requestErr
		if requestErr == nil {
			lastStatus = response.StatusCode
			body, _ := io.ReadAll(io.LimitReader(response.Body, 256))
			_ = response.Body.Close()
			lastBody = strings.Map(func(r rune) rune {
				if r < 0x20 && r != '\t' && r != '\n' {
					return ' '
				}
				return r
			}, strings.TrimSpace(string(body)))
			if lastStatus >= 200 && lastStatus < 300 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("private_port=%d endpoint=%s health_status=%d body=%q transport_error=%v relay_errors=%q: %w", privatePort, endpoint, lastStatus, lastBody, lastErr, boundedRelayErrors(relayErrorsPath), ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func boundedRelayErrors(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(data) > 1024 {
		data = data[len(data)-1024:]
	}
	return strings.TrimSpace(string(data))
}

func offlineAPIPort(rawURL string) (int, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u == nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return 0, fmt.Errorf("offline API endpoint must be a bare loopback HTTP origin")
	}
	return offlineRendererPort(rawURL)
}

func validatePrivateTestDatabaseRoot(path, sessionRoot string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Dir(path) != sessionRoot {
		return fmt.Errorf("test database root escaped the private offline session")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect private test database root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return fmt.Errorf("private test database root must be a real mode-0700 directory")
	}
	return nil
}

func offlineBundledAPIPort(descriptorPath, root, rendererURL string) (int, error) {
	if !pathWithin(root, descriptorPath) {
		return 0, fmt.Errorf("bundled runtime target descriptor escaped the private session root")
	}
	info, err := os.Lstat(descriptorPath)
	if err != nil {
		return 0, fmt.Errorf("bundled runtime did not write a private target descriptor: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("bundled runtime target descriptor is not a regular file")
	}
	data, err := os.ReadFile(descriptorPath)
	if err != nil {
		return 0, fmt.Errorf("read bundled runtime target descriptor: %w", err)
	}
	var descriptor struct {
		RendererURL string `json:"renderer_url"`
		APIURL      string `json:"api_url"`
	}
	if err := json.Unmarshal(data, &descriptor); err != nil {
		return 0, fmt.Errorf("parse bundled runtime target descriptor: %w", err)
	}
	pagePort, err := offlineRendererPort(rendererURL)
	if err != nil {
		return 0, fmt.Errorf("validate selected renderer URL: %w", err)
	}
	descriptorRendererPort, err := offlineRendererPort(descriptor.RendererURL)
	if err != nil || pagePort != descriptorRendererPort {
		return 0, fmt.Errorf("bundled runtime renderer descriptor does not match the selected renderer")
	}
	apiPort, err := offlineAPIPort(descriptor.APIURL)
	if err != nil {
		return 0, fmt.Errorf("bundled runtime API endpoint is not a numeric IPv4 loopback URL: %w", err)
	}
	return apiPort, nil
}

func withoutBundledValidationEndpointOverrides(env []string) []string {
	filtered := env[:0]
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if key == "VROOLI_VALIDATION_API_URL" || key == "VROOLI_VALIDATION_RENDERER_URL" {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func withoutEnvironmentKey(env []string, key string) []string {
	filtered := env[:0]
	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		if name != key {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func summarizeRendererPages(pages []CDPPage) string {
	if len(pages) == 0 {
		return "renderer_pages=0"
	}
	parts := make([]string, 0, len(pages))
	for _, page := range pages {
		parts = append(parts, fmt.Sprintf("type=%s url=%q websocket=%t", page.Type, page.URL, strings.TrimSpace(page.WebSocketDebuggerURL) != ""))
	}
	return "renderer_pages=" + strings.Join(parts, "; ")
}

func rendererMayStillBeStarting(expectation RendererExpectation, err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	if expectation.ID != "" {
		return strings.Contains(message, "was not found or attachable")
	}
	if expectation.URLPrefix != "" || expectation.Title != "" {
		return strings.Contains(message, "matched 0 attachable pages")
	}
	return false
}

// Target returns an immutable copy of the target identity produced at
// attachment time.
func (s *ElectronSession) Target() *domainv1.AppTarget {
	if s == nil || s.target == nil {
		return nil
	}
	return proto.Clone(s.target).(*domainv1.AppTarget)
}

func (s *ElectronSession) Endpoint() string {
	if s == nil {
		return ""
	}
	return s.endpoint
}

func (s *ElectronSession) PID() int {
	if s == nil || s.process == nil {
		return 0
	}
	return s.process.PID()
}

// Restart reopens an exited application with the same owner-created profile.
// It refuses a running process and transfers cleanup ownership only on success.
func (s *ElectronSession) Restart(ctx context.Context) (*ElectronSession, error) {
	if s == nil {
		return nil, fmt.Errorf("Electron validation session required")
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("Electron validation session is closed")
	}
	select {
	case <-s.waitDone:
	default:
		return nil, fmt.Errorf("Electron application must exit before restart")
	}
	opts := s.options
	opts.Launch.UserDataDir = s.profileDir
	opts.Launch.CDPPort = 0
	opts.Renderer.ID = "" // Renderer identity changes across process lifetimes.
	if s.cdpBridge != nil {
		if err := s.cdpBridge.Close(); err != nil {
			return nil, fmt.Errorf("close prior private Electron CDP bridge: %w", err)
		}
		s.cdpBridge = nil
	}
	if s.apiBridge != nil {
		if err := s.apiBridge.Close(); err != nil {
			return nil, fmt.Errorf("close prior private offline API bridge: %w", err)
		}
		s.apiBridge = nil
	}
	next, err := StartElectronSession(ctx, opts)
	if err != nil {
		return nil, err
	}
	next.ownedProfile = s.ownedProfile
	s.ownedProfile = false
	next.ownedNetworkRoot = s.ownedNetworkRoot
	s.ownedNetworkRoot = false
	s.closed = true
	s.closeOnce.Do(func() {}) // Prior process was already reaped; next owns cleanup.
	return next, nil
}

// WaitForExit waits until the managed application process has actually exited.
// Callers must supply a bounded context when waiting for an operator workflow.
func (s *ElectronSession) WaitForExit(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("Electron validation session required")
	}
	select {
	case <-s.waitDone:
		if err := s.waitError(); err != nil && !isExpectedElectronTermination(err) {
			return fmt.Errorf("Electron application exited with an error: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RequestExit asks Chromium to close the Electron browser normally through its
// loopback CDP browser endpoint while preserving restart and cleanup ownership.
func (s *ElectronSession) RequestExit(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("Electron validation session required")
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.closed {
		return fmt.Errorf("Electron validation session is closed")
	}
	select {
	case <-s.waitDone:
		return nil
	default:
	}
	endpoint, err := url.Parse(s.endpoint)
	if err != nil || ValidateAppTargetEndpoint(endpoint) != nil {
		return fmt.Errorf("Electron CDP endpoint is not validated loopback")
	}
	versionURL := *endpoint
	versionURL.Path = "/json/version"
	versionURL.RawQuery = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create Electron browser discovery request: %w", err)
	}
	response, err := (&http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 5 * time.Second}).Do(request)
	if err != nil {
		return fmt.Errorf("discover Electron browser CDP endpoint: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("discover Electron browser CDP endpoint: HTTP %s", response.Status)
	}
	var version struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&version); err != nil {
		return fmt.Errorf("decode Electron browser CDP endpoint: %w", err)
	}
	wsEndpoint, err := url.Parse(version.WebSocketDebuggerURL)
	if err != nil || wsEndpoint.Scheme != "ws" || wsEndpoint.User != nil || wsEndpoint.RawQuery != "" || wsEndpoint.Fragment != "" || !strings.HasPrefix(wsEndpoint.Path, "/devtools/browser/") {
		return fmt.Errorf("Electron browser CDP websocket endpoint is invalid")
	}
	advertisedHost, advertisedPort, err := net.SplitHostPort(wsEndpoint.Host)
	if err != nil || net.ParseIP(advertisedHost) == nil || !net.ParseIP(advertisedHost).IsLoopback() || advertisedPort != endpoint.Port() {
		return fmt.Errorf("Electron browser CDP websocket endpoint does not match validated loopback target")
	}
	wsEndpoint.Host = endpoint.Host
	dialer := websocket.Dialer{Proxy: nil, HandshakeTimeout: 5 * time.Second}
	connection, _, err := dialer.DialContext(ctx, wsEndpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("connect to Electron browser CDP endpoint: %w", err)
	}
	defer connection.Close()
	if err := connection.WriteJSON(struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
	}{ID: 1, Method: "Browser.close"}); err != nil {
		return fmt.Errorf("request Electron browser close through CDP: %w", err)
	}
	return nil
}

// Close performs normal interrupt shutdown, escalates to kill after the
// bounded grace period, waits for the process reaper, and removes only a
// profile directory created by this session.
func (s *ElectronSession) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.closed = true
	s.closeOnce.Do(func() {
		var errs []error
		if err := s.process.Signal(os.Interrupt); err != nil {
			errs = append(errs, fmt.Errorf("signal Electron shutdown: %w", err))
		}
		shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownLimit)
		defer cancel()
		select {
		case <-s.waitDone:
			waitErr := s.waitError()
			if waitErr != nil && !isExpectedElectronTermination(waitErr) {
				errs = append(errs, fmt.Errorf("wait for Electron shutdown: %w", waitErr))
			}
		case <-shutdownCtx.Done():
			if err := s.process.Kill(); err != nil {
				errs = append(errs, fmt.Errorf("kill Electron process: %w", err))
			}
			<-s.waitDone
			if waitErr := s.waitError(); waitErr != nil && !isExpectedElectronTermination(waitErr) {
				errs = append(errs, fmt.Errorf("wait after killing Electron: %w", waitErr))
			}
		}
		if s.cdpBridge != nil {
			if err := s.cdpBridge.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close private Electron CDP bridge: %w", err))
			}
		}
		if s.apiBridge != nil {
			if err := s.apiBridge.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close private offline API bridge: %w", err))
			}
		}
		if s.ownedNetworkRoot {
			if err := os.RemoveAll(s.networkRoot); err != nil {
				errs = append(errs, fmt.Errorf("remove private offline Electron root: %w", err))
			}
		} else if s.ownedProfile {
			if err := os.RemoveAll(s.profileDir); err != nil {
				errs = append(errs, fmt.Errorf("remove isolated Electron profile: %w", err))
			}
		}
		s.closeErr = errors.Join(errs...)
	})
	return s.closeErr
}

// Electron exits with the signal used to stop its process group. That is a
// successful cleanup outcome, not a leaked target. Preserve all other wait
// failures so callers still fail closed on an incomplete teardown.
func isExpectedElectronTermination(err error) bool {
	if err == nil || errors.Is(err, os.ErrProcessDone) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "signal: interrupt") || strings.Contains(message, "signal: killed")
}

func (s *ElectronSession) waitError() error {
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	return s.waitErr
}

func validateArtifactPath(path string) error {
	if path == "" {
		return fmt.Errorf("Electron artifact path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat Electron artifact: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("Electron artifact must be a file")
	}
	return nil
}

func allocateLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

type execElectronProcess struct {
	cmd      *exec.Cmd
	waitDone chan error
	output   *bytes.Buffer
}

func startExecProcess(_ context.Context, path string, args []string, env []string) (ElectronProcess, error) {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	output := &bytes.Buffer{}
	cmd.Stdout = output
	cmd.Stderr = output
	configureElectronCommand(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	process := &execElectronProcess{cmd: cmd, waitDone: make(chan error, 1), output: output}
	go func() { process.waitDone <- cmd.Wait() }()
	return process, nil
}

func (p *execElectronProcess) PID() int { return p.cmd.Process.Pid }
func (p *execElectronProcess) Signal(signal os.Signal) error {
	return signalElectronCommand(p.cmd, signal)
}
func (p *execElectronProcess) Kill() error { return killElectronCommand(p.cmd) }
func (p *execElectronProcess) Wait() error {
	err := <-p.waitDone
	if err == nil || p.output == nil {
		return err
	}
	if output := strings.TrimSpace(p.output.String()); output != "" {
		return fmt.Errorf("%w; artifact output: %s", err, summarizeArtifactOutput(output))
	}
	return err
}

func summarizeArtifactOutput(output string) string {
	// electron-builder/runtime startup can emit one identical staging line per
	// bundled file. Remove that noise first so the retained diagnostic contains
	// the actual runtime failure rather than only the tail of the copy log.
	output = strings.ReplaceAll(output, "File exists and file size matches, skipping\n", "")
	if marker := strings.Index(output, "[Desktop App]"); marker > 0 {
		output = output[marker:]
	}
	const maxOutput = 8192
	if len(output) <= maxOutput {
		return output
	}
	const head = maxOutput / 2
	return output[:head] + "\n…<artifact output elided>…\n" + output[len(output)-(maxOutput-head):]
}
