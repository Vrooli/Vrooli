package env

// BundledEnvironmentReasons is the reviewable contract for inherited names.
var BundledEnvironmentReasons = map[string]string{
	"DISPLAY":       "native Linux children need the assigned display",
	"HOME":          "libraries need the user-scoped home for non-secret caches",
	"LANG":          "children need the requested locale",
	"LC_ALL":        "operators can force a deterministic locale for evidence",
	"PATH":          "executables need platform tools and bundled binaries",
	"SSL_CERT_DIR":  "TLS clients need the platform certificate directory",
	"SSL_CERT_FILE": "TLS clients need the platform certificate file",
	"TMPDIR":        "tools need the runtime temporary workspace",
	"TZ":            "time formatting needs the configured timezone",
	"USER":          "tools use the logical account name for ownership",
	"XAUTHORITY":    "native Linux children need display credentials",
}

// BundledEnvironmentAllowlist is the stable sorted list used by callers.
var BundledEnvironmentAllowlist = []string{
	"DISPLAY", "HOME", "LANG", "LC_ALL", "PATH", "SSL_CERT_DIR",
	"SSL_CERT_FILE", "TMPDIR", "TZ", "USER", "XAUTHORITY",
}
