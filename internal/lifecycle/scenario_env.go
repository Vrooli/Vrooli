package lifecycle

import (
	"slices"
	"strings"
)

// Scenario process environment boundary.
//
// The lifecycle runs inside whichever process asked for it: an operator shell,
// an agent's terminal, Web Console, the autoheal unit, or the runtime
// supervisor. That process's environment is the caller's, not the scenario's.
// A managed scenario process therefore receives:
//
//  1. the host variables named by scenarioPassThrough (OS user, locale,
//     desktop session, toolchain, network trust, Vrooli installation roots);
//  2. the lifecycle overlay: declared manifest values, resolved ports and
//     discovery, resource environment, and the authentication binding derived
//     from the manifest and operator state.
//
// Everything else the caller carries is dropped. In particular no
// authentication setting, credential, or agent identity crosses this boundary
// from the caller: a scenario restarted from an agent shell that happens to
// export VROOLI_AUTH_MODE=personal_local still runs in the mode its manifest
// and operator state select, and never acts as that agent run.
//
// Adding a variable here is a reviewed decision. A scenario that needs a value
// declares it (service.json environment, resource environment, or schema-backed
// settings) instead of relying on the shell that started it.

// scenarioPassThrough lists the caller variables a scenario process may inherit.
// Names are matched case-insensitively so Windows spellings fold to the same
// policy.
var scenarioPassThrough = map[string]struct{}{}

// scenarioPassThroughPrefixes are variable families that describe the host,
// never a caller's authority. Credential-shaped names inside a family are
// still refused by isSecurityRelevantEnv.
var scenarioPassThroughPrefixes = []string{
	"LC_",         // locale categories
	"XDG_",        // desktop session directories and runtime dir
	"QT_",         // desktop toolkit integration
	"CGO_",        // C toolchain flags for Go builds
	"NPM_CONFIG_", // operator package-manager tuning (e.g. concurrency limits)
}

func init() {
	groups := [][]string{
		// Operating-system user, shell and locale.
		{"PATH", "HOME", "USER", "LOGNAME", "USERNAME", "SHELL", "TMPDIR", "TMP", "TEMP", "TZ", "LANG", "LANGUAGE", "NO_COLOR"},
		// Desktop session channels owned by the OS login, not by a caller:
		// GUI display, session bus (keyring), and the session ssh/gpg agents.
		{
			"DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "DBUS_SESSION_BUS_ADDRESS", "SSH_AUTH_SOCK", "GPG_AGENT_INFO",
			"DESKTOP_SESSION", "GDMSESSION", "SESSION_MANAGER", "WINDOWPATH", "XMODIFIERS", "GTK_MODULES",
		},
		// Go toolchain location and operator build tuning.
		{
			"GOPATH", "GOROOT", "GOFLAGS", "GOMAXPROCS", "GOMEMLIMIT", "GOGC", "GOCACHE", "GOMODCACHE", "GOTMPDIR",
			"GOPROXY", "GOPRIVATE", "GONOPROXY", "GONOSUMDB", "GONOSUMCHECK", "GOSUMDB", "GOINSECURE", "GOTOOLCHAIN",
			"GOTELEMETRY", "GOAMD64", "GOWORK",
		},
		// Native, Node, Rust, Java and Android toolchain locations.
		{
			"CC", "CXX", "PKG_CONFIG_PATH", "LD_LIBRARY_PATH",
			"NVM_DIR", "NVM_BIN", "NVM_INC", "PNPM_HOME", "COREPACK_HOME", "COREPACK_ENABLE_AUTO_PIN",
			"COREPACK_ENABLE_DOWNLOAD_PROMPT", "NODE_EXTRA_CA_CERTS",
			"CARGO_HOME", "RUSTUP_HOME", "JAVA_HOME", "ANDROID_HOME", "ANDROID_SDK_ROOT",
		},
		// Network egress and TLS trust configured for the host.
		{
			"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY",
			"SSL_CERT_FILE", "SSL_CERT_DIR", "REQUESTS_CA_BUNDLE", "CURL_CA_BUNDLE",
		},
		// Accelerator visibility and the host container daemon.
		{"CUDA_VISIBLE_DEVICES", "NVIDIA_VISIBLE_DEVICES", "HIP_VISIBLE_DEVICES", "ROCR_VISIBLE_DEVICES", "CUDA_HOME", "CUDA_PATH", "DOCKER_HOST"},
		// Windows system variables a process cannot start without.
		{
			"SYSTEMROOT", "SYSTEMDRIVE", "WINDIR", "COMSPEC", "PATHEXT", "USERPROFILE", "APPDATA", "LOCALAPPDATA",
			"PROGRAMDATA", "PROGRAMFILES", "PROGRAMFILES(X86)", "PROGRAMW6432", "COMPUTERNAME", "NUMBER_OF_PROCESSORS",
			"PROCESSOR_ARCHITECTURE", "OS",
		},
		// Vrooli installation location. These say where this installation keeps
		// its source, binaries, storage and artifacts; isolated test
		// environments redirect them, and the scenario must follow the same
		// redirection as the control plane that started it.
		{
			"VROOLI_ROOT", "VROOLI_SOURCE_ROOT", "VROOLI_HOME", "VROOLI_BIN", "VROOLI_DATA", "VROOLI_DATA_ROOT",
			"VROOLI_STATE_ROOT", "VROOLI_CONFIG_ROOT", "VROOLI_CACHE_ROOT", "VROOLI_LOGS_ROOT", "VROOLI_STORAGE_ROOT",
			"VROOLI_STORAGE_PLATFORM", "VROOLI_RESOURCE_STORAGE_ROOT", "VROOLI_RESOURCE_ARTIFACT_DIR",
			"VROOLI_RCL_ARTIFACT_ROOT", "VROOLI_PROTO_ARTIFACT_ROOT", "VROOLI_CLI_ARTIFACT_MODE", "VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED",
			"VROOLI_SANDBOX_SCOPE", "VROOLI_HOST_LIFECYCLE_BASE", "VROOLI_API_PORT",
		},
	}
	for _, group := range groups {
		for _, name := range group {
			scenarioPassThrough[name] = struct{}{}
		}
	}
}

// scenarioEnvDrop classifies one caller variable that did not reach a
// scenario process. Security drops are reported by name in the lifecycle log;
// values are never recorded.
type scenarioEnvDrop struct {
	Name     string
	Security bool
}

// hostSessionChannels are allowlisted sockets whose names happen to look
// credential-shaped. They are OS-login channels, not caller credentials.
var hostSessionChannels = map[string]struct{}{"SSH_AUTH_SOCK": {}, "XAUTHORITY": {}, "GPG_AGENT_INFO": {}}

// isSecurityRelevantEnv reports variables that configure authentication,
// carry a credential, or identify an agent run or session. None of them may be
// inherited by a scenario process from the caller.
func isSecurityRelevantEnv(name string) bool {
	key := strings.ToUpper(name)
	if _, ok := hostSessionChannels[key]; ok {
		return false
	}
	for _, prefix := range []string{"VROOLI_AUTH_", "VROOLI_CLOUDFLARE_ACCESS_", "VROOLI_AGENT_", "VROOLI_OPERATOR_SESSION", "VROOLI_BREAK_GLASS"} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	switch key {
	case "VROOLI_RUN_ID", "VROOLI_HARNESS_SESSION_ID", "VROOLI_AGENT_MANAGER_STORE_ID", "VROOLI_TEST_MODE_FORCE_ENABLE",
		"CLAUDE_CODE_SESSION_ID", "CODEX_THREAD_ID", "CLAUDE_CODE_MESSAGING_TOKEN", "CLAUDE_CODE_MESSAGING_SOCKET",
		"WC_WEB_CONSOLE_SESSION_ID", "WC_SESSION_STATE_ROOT":
		return true
	}
	for _, marker := range []string{"TOKEN", "SECRET", "PASSWORD", "PASSWD", "API_KEY", "APIKEY", "PRIVATE_KEY", "CREDENTIAL", "_AUTH_", "SESSION_KEY"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return strings.HasSuffix(key, "_AUTH") || strings.HasSuffix(key, "_KEY")
}

func scenarioMayInherit(name string) bool {
	key := strings.ToUpper(name)
	if isSecurityRelevantEnv(key) {
		return false
	}
	if _, ok := scenarioPassThrough[key]; ok {
		return true
	}
	for _, prefix := range scenarioPassThroughPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// scenarioInheritedEnv filters the caller environment down to the documented
// pass-through set. It returns the retained entries in their original order
// and every dropped name, sorted, with its security classification.
func scenarioInheritedEnv(caller []string) ([]string, []scenarioEnvDrop) {
	kept := make([]string, 0, len(caller))
	dropped := make([]scenarioEnvDrop, 0)
	seen := make(map[string]struct{})
	for _, entry := range caller {
		name, _, ok := strings.Cut(entry, "=")
		if !ok || name == "" {
			continue
		}
		if scenarioMayInherit(name) {
			kept = append(kept, entry)
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		dropped = append(dropped, scenarioEnvDrop{Name: name, Security: isSecurityRelevantEnv(name)})
	}
	slices.SortFunc(dropped, func(a, b scenarioEnvDrop) int { return strings.Compare(a.Name, b.Name) })
	return kept, dropped
}

// droppedSecurityEnvNames returns the security-relevant caller variable names
// that scenarioInheritedEnv refuses, for the start log.
func droppedSecurityEnvNames(caller []string) []string {
	_, dropped := scenarioInheritedEnv(caller)
	names := make([]string, 0)
	for _, drop := range dropped {
		if drop.Security {
			names = append(names, drop.Name)
		}
	}
	return names
}
