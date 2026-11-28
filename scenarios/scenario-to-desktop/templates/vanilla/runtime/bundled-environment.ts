const BUNDLED_ENVIRONMENT_ALLOWLIST = [
    "HOME", "USER", "LANG", "LC_ALL", "TZ", "TMPDIR", "DISPLAY",
    "XAUTHORITY", "PATH", "SSL_CERT_FILE", "SSL_CERT_DIR",
    "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy",
    "DBUS_SESSION_BUS_ADDRESS", "XDG_RUNTIME_DIR",
    // Process-lifetime attribution is needed to reap children even if a host
    // supervisor changes their process group. Host agent routing is omitted.
    "S2D_PROCESS_SESSION",
] as const;

export function bundledEnvironmentBase(hostEnvironment: NodeJS.ProcessEnv, lpbsURL?: string): Record<string, string | undefined> {
    const environment = Object.fromEntries(
        BUNDLED_ENVIRONMENT_ALLOWLIST
            .filter((name) => hostEnvironment[name] !== undefined)
            .map((name) => [name, hostEnvironment[name]]),
    );
    // The app configuration owns the public account/AI endpoint. Host routing
    // variables are intentionally excluded from this environment.
    if (lpbsURL) environment.LPBS_AI_DOOR_URL = lpbsURL;
    return environment;
}
