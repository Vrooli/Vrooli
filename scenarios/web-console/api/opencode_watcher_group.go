package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"web-console/backends/opencode"
)

// openCodeCaptureTarget identifies one concrete OpenCode executable to capture
// from. V1 and V2 expose different session stores and HTTP APIs, so a host that
// has both needs one capture loop per major.
type openCodeCaptureTarget struct {
	Path    string
	Major   int
	Version string
}

// OpenCodeWatcherGroup runs one capture loop per installed OpenCode major. A
// V1 pane stays on V1 after an upgrade, so a single capture server built from
// the globally selected binary would silently hide it.
type OpenCodeWatcherGroup struct {
	server   *Server
	captures []*OpenCodeWatcher
}

func NewOpenCodeWatcherGroup(server *Server) *OpenCodeWatcherGroup {
	targets := discoverOpenCodeCaptureTargets()
	group := &OpenCodeWatcherGroup{server: server}
	if len(targets) == 0 {
		// Preserve the historical PATH-based fallback when no managed install
		// is discoverable (for example a source-only test host).
		group.captures = append(group.captures, NewOpenCodeWatcher(server))
		return group
	}
	for _, target := range targets {
		target := target
		w := NewOpenCodeWatcher(server)
		w.major = target.Major
		w.startServer = func() (opencode.Server, func(), error) {
			return spawnOpenCodeServeForPath(target.Path)
		}
		group.captures = append(group.captures, w)
	}
	return group
}

func (g *OpenCodeWatcherGroup) Start() {
	if g == nil {
		return
	}
	for _, capture := range g.captures {
		capture.Start()
	}
}

func (g *OpenCodeWatcherGroup) Stop() {
	if g == nil {
		return
	}
	for _, capture := range g.captures {
		capture.Stop()
	}
}

func (g *OpenCodeWatcherGroup) nativeClient() opencode.Client {
	if g == nil || len(g.captures) == 0 {
		return nil
	}
	if len(g.captures) == 1 {
		return g.captures[0].nativeClient()
	}
	return &routingOpenCodeClient{group: g}
}

func (g *OpenCodeWatcherGroup) reconcileNativeSession(ctx context.Context, nativeSessionID, webSessionID string) {
	if g == nil {
		return
	}
	for _, capture := range g.captures {
		if capture.ownsNativeSession(nativeSessionID) || capture.ownsWebSession(webSessionID) {
			capture.reconcileNativeSession(ctx, nativeSessionID, webSessionID)
			return
		}
	}
	// Attribution may not have reached this pane yet. Try each capture; the
	// server that owns the native session answers, the other logs a miss.
	for _, capture := range g.captures {
		capture.reconcileNativeSession(ctx, nativeSessionID, webSessionID)
	}
}

func (g *OpenCodeWatcherGroup) ReplyPermission(ctx context.Context, sessionID, reply string) error {
	if g == nil || len(g.captures) == 0 {
		return errors.New("OpenCode capture is not running")
	}
	var firstErr error
	for _, capture := range g.captures {
		err := capture.ReplyPermission(ctx, sessionID, reply)
		if err == nil {
			return nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("no OpenCode permission request is pending for session %s", sessionID)
	}
	return firstErr
}

func (g *OpenCodeWatcherGroup) captureForNativeSession(nativeSessionID string) *OpenCodeWatcher {
	if g == nil {
		return nil
	}
	for _, capture := range g.captures {
		if capture.ownsNativeSession(nativeSessionID) {
			return capture
		}
	}
	return nil
}

// routingOpenCodeClient sends native control calls to the capture that owns the
// session. V1 and V2 session IDs are distinct; a single native client cannot
// reach both servers.
type routingOpenCodeClient struct {
	group *OpenCodeWatcherGroup
}

func (c *routingOpenCodeClient) firstClient() opencode.Client {
	if c == nil || c.group == nil {
		return nil
	}
	for _, capture := range c.group.captures {
		if client := capture.nativeClient(); client != nil {
			return client
		}
	}
	return nil
}

func (c *routingOpenCodeClient) ListSessions(ctx context.Context, directory string) ([]opencode.Session, error) {
	client := c.firstClient()
	if client == nil {
		return nil, errors.New("the OpenCode server is not connected")
	}
	return client.ListSessions(ctx, directory)
}

func (c *routingOpenCodeClient) SessionMessages(ctx context.Context, sessionID string) ([]opencode.MessageWithParts, error) {
	if owner := c.group.captureForNativeSession(sessionID); owner != nil {
		if client := owner.nativeClient(); client != nil {
			return client.SessionMessages(ctx, sessionID)
		}
	}
	var firstErr error
	for _, capture := range c.group.captures {
		client := capture.nativeClient()
		if client == nil {
			continue
		}
		messages, err := client.SessionMessages(ctx, sessionID)
		if err == nil && len(messages) > 0 {
			return messages, nil
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, nil
}

func (c *routingOpenCodeClient) SessionStatus(ctx context.Context) (map[string]opencode.SessionStatus, error) {
	client := c.firstClient()
	if client == nil {
		return nil, errors.New("the OpenCode server is not connected")
	}
	return client.SessionStatus(ctx)
}

func (c *routingOpenCodeClient) AbortSession(ctx context.Context, sessionID string) error {
	return c.routeBySession(sessionID, func(client opencode.Client) error {
		return client.AbortSession(ctx, sessionID)
	})
}

func (c *routingOpenCodeClient) RevertMessage(ctx context.Context, sessionID, messageID, partID string) error {
	return c.routeBySession(sessionID, func(client opencode.Client) error {
		return client.RevertMessage(ctx, sessionID, messageID, partID)
	})
}

func (c *routingOpenCodeClient) Events(ctx context.Context, onEvent func(opencode.Event)) error {
	client := c.firstClient()
	if client == nil {
		return errors.New("the OpenCode server is not connected")
	}
	return client.Events(ctx, onEvent)
}

func (c *routingOpenCodeClient) ReplyPermission(ctx context.Context, sessionID, requestID, reply string) error {
	if c == nil || c.group == nil {
		return errors.New("OpenCode capture is not running")
	}
	return c.group.ReplyPermission(ctx, sessionID, reply)
}

func (c *routingOpenCodeClient) routeBySession(sessionID string, run func(opencode.Client) error) error {
	if c == nil || c.group == nil {
		return errors.New("OpenCode capture is not running")
	}
	if owner := c.group.captureForNativeSession(sessionID); owner != nil {
		if client := owner.nativeClient(); client != nil {
			return run(client)
		}
	}
	var firstErr error
	for _, capture := range c.group.captures {
		client := capture.nativeClient()
		if client == nil {
			continue
		}
		err := run(client)
		if err == nil {
			return nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = errors.New("the OpenCode server is not connected")
	}
	return firstErr
}

// discoverOpenCodeCaptureTargets finds the newest managed install for each
// major. It understands both the historical flat cli/<version>-<os>-<arch>
// layout and the channel-scoped cli/<channel>/<version>-<os>-<arch> layout.
func discoverOpenCodeCaptureTargets() []openCodeCaptureTarget {
	byMajor := map[int]openCodeCaptureTarget{}
	for _, dataDir := range openCodeDataDirs() {
		cliRoot := filepath.Join(dataDir, "cli")
		for _, installDir := range openCodeInstallDirs(cliRoot) {
			version, ok := installDirVersion(installDir)
			if !ok {
				continue
			}
			major := majorFromOpenCodeVersion(version)
			if major < 1 {
				continue
			}
			binary := findOpenCodeExecutable(installDir)
			if binary == "" {
				continue
			}
			current, exists := byMajor[major]
			if !exists || versionGreater(version, current.Version) {
				byMajor[major] = openCodeCaptureTarget{Path: binary, Major: major, Version: version}
			}
		}
	}
	targets := make([]openCodeCaptureTarget, 0, len(byMajor))
	for _, target := range byMajor {
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		if path, err := exec.LookPath("opencode"); err == nil {
			if major := openCodeMajorForPath(path); major > 0 {
				targets = append(targets, openCodeCaptureTarget{Path: path, Major: major})
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Major < targets[j].Major })
	return targets
}

func openCodeDataDirs() []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return
		}
		dir = filepath.Clean(dir)
		if seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	add(os.Getenv("OPENCODE_DATA_DIR"))
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		add(filepath.Join(xdg, "vrooli", "resources", "opencode"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".local", "share", "vrooli", "resources", "opencode"))
	}
	return dirs
}

func openCodeInstallDirs(cliRoot string) []string {
	entries, err := os.ReadDir(cliRoot)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(cliRoot, entry.Name())
		if _, ok := installDirVersion(path); ok {
			dirs = append(dirs, path)
			continue
		}
		// A channel directory groups its versions one level deeper.
		children, err := os.ReadDir(path)
		if err != nil {
			continue
		}
		for _, child := range children {
			if !child.IsDir() {
				continue
			}
			candidate := filepath.Join(path, child.Name())
			if _, ok := installDirVersion(candidate); ok {
				dirs = append(dirs, candidate)
			}
		}
	}
	return dirs
}

func installDirVersion(dir string) (string, bool) {
	suffix := "-" + runtime.GOOS + "-" + runtime.GOARCH
	base := filepath.Base(filepath.Clean(dir))
	if !strings.HasSuffix(base, suffix) {
		return "", false
	}
	version := strings.TrimSuffix(base, suffix)
	if version == "" || version == base {
		return "", false
	}
	if majorFromOpenCodeVersion(version) < 1 {
		return "", false
	}
	return version, true
}

func majorFromOpenCodeVersion(version string) int {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	head, _, ok := strings.Cut(version, ".")
	if !ok {
		return 0
	}
	major, err := strconv.Atoi(head)
	if err != nil || major < 1 {
		return 0
	}
	return major
}

func providerMajor(version string) int {
	match := opencodeVersionRE.FindStringSubmatch(strings.TrimSpace(version))
	if match == nil {
		return 0
	}
	major, err := strconv.Atoi(match[1])
	if err != nil || major < 1 {
		return 0
	}
	return major
}

func openCodeMajorForPath(path string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return 0
	}
	match := opencodeVersionRE.FindStringSubmatch(string(out))
	if match == nil {
		return 0
	}
	major, err := strconv.Atoi(match[1])
	if err != nil || major < 1 {
		return 0
	}
	return major
}

func versionGreater(a, b string) bool {
	ap, bp := versionParts(a), versionParts(b)
	for i := 0; i < len(ap) || i < len(bp); i++ {
		var av, bv int
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av != bv {
			return av > bv
		}
	}
	return false
}

func versionParts(version string) []int {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(version), "v"), ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		if digits := leadingDigits(part); digits != "" {
			value, _ := strconv.Atoi(digits)
			out = append(out, value)
		} else {
			out = append(out, 0)
		}
	}
	return out
}

func leadingDigits(value string) string {
	for i, r := range value {
		if r < '0' || r > '9' {
			return value[:i]
		}
	}
	return value
}

// findOpenCodeExecutable locates the native executable inside a managed npm
// prefix without running lifecycle scripts. The V1 and V2 main packages both
// place the real binary in a platform optional package under node_modules.
func findOpenCodeExecutable(prefix string) string {
	root := filepath.Join(prefix, "node_modules")
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return ""
	}
	found := ""
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			rel, relErr := filepath.Rel(root, path)
			if relErr == nil && len(strings.Split(rel, string(filepath.Separator))) > 5 {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "opencode" || filepath.Base(filepath.Dir(path)) != "bin" {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			return nil
		}
		found = path
		return fs.SkipAll
	})
	return found
}
