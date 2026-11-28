package resources

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
	activitynetwork "github.com/vrooli/vrooli/internal/resources/activitynetwork"
	"github.com/vrooli/vrooli/internal/values"
)

const activityNetworkJoinedEnv = "VROOLI_ACTIVITY_NETWORK_JOINED"

type activityNetworkLaunchKind uint8

const (
	activityNetworkLaunchUnknown activityNetworkLaunchKind = iota
	activityNetworkLaunchRootlessKit
	activityNetworkLaunchLegacyDirect
)

type activityNetworkLaunchFacts struct {
	PID              int
	Executable       string
	CommandLine      string
	RootlessKitPath  string
	RootlessKitArgs  []string
	ArtifactPath     string
	StateDir         string
	ChildPID         int
	ChildParentPID   int
	ChildExecutable  string
	ChildCommandLine string
	ChildArgs        []string
	LegacyServiceArg []string
}

// classifyActivityNetworkLaunch validates the launcher boundary recorded by
// the managed-service supervisor. A live direct service is recognized only
// when both its artifact executable and complete expected argv match; a
// RootlessKit owner additionally requires its declared state directory and a
// child whose parent is the recorded owner PID.
func classifyActivityNetworkLaunch(f activityNetworkLaunchFacts) (activityNetworkLaunchKind, error) {
	if f.PID <= 1 || strings.TrimSpace(f.Executable) == "" || strings.TrimSpace(f.CommandLine) == "" {
		return activityNetworkLaunchUnknown, fmt.Errorf("activity-network launch facts are incomplete")
	}
	if filepath.Clean(strings.TrimSuffix(f.Executable, " (deleted)")) == filepath.Clean(f.RootlessKitPath) {
		if !processCommandLineMatches(f.CommandLine, f.RootlessKitPath, f.RootlessKitArgs) {
			return activityNetworkLaunchUnknown, fmt.Errorf("RootlessKit owner does not match its declared launcher arguments")
		}
		if f.ChildPID <= 1 || f.ChildParentPID != f.PID ||
			filepath.Clean(strings.TrimSuffix(f.ChildExecutable, " (deleted)")) != filepath.Clean(f.ArtifactPath) ||
			!processCommandLineMatches(f.ChildCommandLine, f.ArtifactPath, f.ChildArgs) {
			return activityNetworkLaunchUnknown, fmt.Errorf("RootlessKit child is not proven to belong to its managed owner")
		}
		return activityNetworkLaunchRootlessKit, nil
	}
	if filepath.Clean(strings.TrimSuffix(f.Executable, " (deleted)")) == filepath.Clean(f.ArtifactPath) &&
		processCommandLineMatches(f.CommandLine, f.ArtifactPath, f.LegacyServiceArg) {
		return activityNetworkLaunchLegacyDirect, nil
	}
	return activityNetworkLaunchUnknown, fmt.Errorf("managed-service process does not match either declared activity-network launch contract")
}

// processCommandLineMatches uses the host's argv projection only for commands
// whose arguments contain no whitespace. It fails closed when an argument
// would be ambiguous in that representation.
func processCommandLineMatches(commandLine, executable string, args []string) bool {
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\r\n") {
			return false
		}
	}
	want := append([]string{executable}, args...)
	return strings.Join(strings.Fields(commandLine), " ") == strings.Join(want, " ")
}

// RunActivityNetworkOwnerCommand joins the exact RootlessKit namespace for a
// resource-owned command that must address its private server listener. It is
// a no-op when the resource is stopped or the caller already joined the same
// namespace. Workload traffic uses the host activity edge and owner-authorized
// Unix socket bridge instead of a published backend TCP port.
func RunActivityNetworkOwnerCommand(resource string, command []string) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, nil
	}
	if !isActivityNetworkResource(resource) {
		return false, fmt.Errorf("resource %q has no private activity network owner", resource)
	}
	if len(command) == 0 || !filepath.IsAbs(command[0]) {
		return false, fmt.Errorf("activity network owner command must start with an absolute executable path")
	}
	supervisor, _, err := managedServiceSupervisorFor(resource)
	if err != nil {
		return false, err
	}
	state, running, err := supervisor.Status()
	if err != nil {
		return false, fmt.Errorf("read %s managed-service owner: %w", resource, err)
	}
	if !running {
		return false, nil
	}
	rootlesskit, err := exec.LookPath("rootlesskit")
	if err != nil {
		return false, fmt.Errorf("resolve RootlessKit owner for %s: %w", resource, err)
	}
	if !filepath.IsAbs(rootlesskit) {
		rootlesskit, err = filepath.Abs(rootlesskit)
		if err != nil {
			return false, err
		}
	}
	actualExecutable, err := platform.ProcessExecutablePath(state.PID)
	if err != nil || filepath.Clean(strings.TrimSuffix(actualExecutable, " (deleted)")) != filepath.Clean(rootlesskit) {
		return false, fmt.Errorf("%s managed-service owner is not the declared RootlessKit process", resource)
	}
	stateDir := filepath.Join(os.TempDir(), "vra-"+resource)
	commandLine, err := platform.ProcessCommandLine(state.PID)
	if err != nil || !strings.Contains(commandLine, "--state-dir "+stateDir) {
		return false, fmt.Errorf("%s RootlessKit process does not own its declared private state directory", resource)
	}
	childPID, err := activitynetwork.ReadChildPID(stateDir)
	if err != nil {
		return false, err
	}
	if cliutil.ProcessParent(childPID) != state.PID {
		return false, fmt.Errorf("%s RootlessKit child PID is not owned by its managed service", resource)
	}
	if strings.TrimSpace(os.Getenv(activityNetworkJoinedEnv)) == "1" {
		if !sameNetworkNamespace(os.Getpid(), childPID) {
			return false, fmt.Errorf("%s activity-network joined marker is not backed by the managed owner's namespace", resource)
		}
		ownerEnv, err := verifiedActivityNetworkOwnerEnvironment(state)
		if err != nil {
			return false, err
		}
		commandEnv, err := activityNetworkOwnerCommandEnvironment(resource, ownerEnv, os.Environ())
		if err != nil {
			return false, err
		}
		for _, entry := range commandEnv {
			key, value, ok := strings.Cut(entry, "=")
			if ok && (key == "OLLAMA_BASE_URL" || key == activityNetworkJoinedEnv) {
				if err := os.Setenv(key, value); err != nil {
					return false, fmt.Errorf("set verified %s owner command environment: %w", key, err)
				}
			}
		}
		return false, nil
	}
	ownerEnv, err := verifiedActivityNetworkOwnerEnvironment(state)
	if err != nil {
		return false, err
	}
	commandEnv, err := activityNetworkOwnerCommandEnvironment(resource, ownerEnv, os.Environ())
	if err != nil {
		return false, err
	}
	nsArgs, err := activitynetwork.OwnerNamespaceArgs(childPID, command)
	if err != nil {
		return false, err
	}
	cmd := exec.Command("nsenter", nsArgs...)
	cmd.Env = commandEnv
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return true, err
	}
	return true, nil
}

func verifiedActivityNetworkOwnerEnvironment(state ManagedServiceState) (map[string]string, error) {
	env, err := readManagedServiceEnvironment(state.PID)
	if err != nil {
		return nil, fmt.Errorf("read managed-service owner environment: %w", err)
	}
	token := env[managedServiceOwnershipTokenEnv]
	if token == "" || managedServiceTokenHash(token) != state.OwnershipTokenHash {
		return nil, fmt.Errorf("managed-service owner environment token does not match the supervisor record")
	}
	incarnation, err := managedServiceProcessIncarnation(state.PID)
	if err != nil || incarnation == "" || incarnation != state.Incarnation {
		return nil, fmt.Errorf("managed-service owner incarnation changed while reading its environment")
	}
	return env, nil
}

func activityNetworkOwnerCommandEnvironment(resource string, ownerEnv map[string]string, ambient []string) ([]string, error) {
	env := values.SetEnv(ambient, activityNetworkJoinedEnv, "1")
	if resource != "ollama" {
		return env, nil
	}
	baseURL, err := ollamaOwnerBaseURL(ownerEnv["OLLAMA_HOST"])
	if err != nil {
		return nil, err
	}
	// Remove all ambient copies before appending the owner-derived endpoint so
	// duplicate OLLAMA_BASE_URL entries cannot override this trusted value.
	result := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, "OLLAMA_BASE_URL=") {
			result = append(result, entry)
		}
	}
	return append(result, "OLLAMA_BASE_URL="+baseURL), nil
}

func ollamaOwnerBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("verified Ollama owner environment has no OLLAMA_HOST listener")
	}
	var host, port string
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "http" || u.User != nil || u.Host == "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
			return "", fmt.Errorf("verified Ollama OLLAMA_HOST must be an HTTP loopback host:port")
		}
		host, port = u.Hostname(), u.Port()
	} else {
		var err error
		host, port, err = net.SplitHostPort(raw)
		if err != nil {
			return "", fmt.Errorf("verified Ollama OLLAMA_HOST must be a loopback host:port: %w", err)
		}
	}
	ip := net.ParseIP(host)
	portNumber, portErr := strconv.Atoi(port)
	if ip == nil || !ip.IsLoopback() || portErr != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("verified Ollama OLLAMA_HOST must use a literal loopback IP and valid port")
	}
	return "http://" + net.JoinHostPort(ip.String(), strconv.Itoa(portNumber)), nil
}

func isActivityNetworkResource(resource string) bool {
	switch strings.TrimSpace(resource) {
	case "whisper", "reranker", "ollama", "kyutai-stt":
		return true
	default:
		return false
	}
}
