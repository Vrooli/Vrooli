package env

// BundledEnvironmentReasons is the reviewable contract for inherited names.
var BundledEnvironmentReasons = map[string]string{
	"DBUS_SESSION_BUS_ADDRESS":  "native credential authority needs the selected Secret Service session",
	"DISPLAY":                   "native Linux children need the assigned display",
	"HTTP_PROXY":                "HTTP clients need the configured network proxy",
	"HTTPS_PROXY":               "HTTPS clients need the configured network proxy",
	"HOME":                      "libraries need the user-scoped home for non-secret caches",
	"LANG":                      "children need the requested locale",
	"LPBS_AI_DOOR_URL":          "the configured app endpoint connects embedded Gateway to the public AI door",
	"LC_ALL":                    "operators can force a deterministic locale for evidence",
	"NO_PROXY":                  "local services must bypass a configured network proxy",
	"PATH":                      "executables need platform tools and bundled binaries",
	"SSL_CERT_DIR":              "TLS clients need the platform certificate directory",
	"SSL_CERT_FILE":             "TLS clients need the platform certificate file",
	"TMPDIR":                    "tools need the runtime temporary workspace",
	"TZ":                        "time formatting needs the configured timezone",
	"USER":                      "tools use the logical account name for ownership",
	"XDG_RUNTIME_DIR":           "native credential authority needs the selected session directory",
	"http_proxy":                "HTTP clients need the configured network proxy",
	"https_proxy":               "HTTPS clients need the configured network proxy",
	"no_proxy":                  "local services must bypass a configured network proxy",
	"VROOLI_CREDENTIAL_BACKEND": "desktop runtime selects native OS credential storage without repository setup state",
	"XAUTHORITY":                "native Linux children need display credentials",
}

// BundledEnvironmentAllowlist is the stable sorted list used by callers.
var BundledEnvironmentAllowlist = []string{
	"DBUS_SESSION_BUS_ADDRESS", "DISPLAY", "HOME", "HTTPS_PROXY", "HTTP_PROXY",
	"LANG", "LC_ALL", "LPBS_AI_DOOR_URL", "NO_PROXY", "PATH", "SSL_CERT_DIR",
	"SSL_CERT_FILE", "TMPDIR", "TZ", "USER", "VROOLI_CREDENTIAL_BACKEND", "XAUTHORITY", "XDG_RUNTIME_DIR",
	"http_proxy", "https_proxy", "no_proxy",
}
