const BUNDLED_ENVIRONMENT_ALLOWLIST = [
    "HOME", "USER", "LANG", "LC_ALL", "TZ", "TMPDIR", "DISPLAY",
    "XAUTHORITY", "PATH", "SSL_CERT_FILE", "SSL_CERT_DIR",
    // Process-lifetime attribution is needed to reap children even if a host
    // supervisor changes their process group. Host agent routing is omitted.
    "S2D_PROCESS_SESSION",
] as const;

export function bundledEnvironmentBase(hostEnvironment: NodeJS.ProcessEnv): Record<string, string | undefined> {
    return Object.fromEntries(
        BUNDLED_ENVIRONMENT_ALLOWLIST
            .filter((name) => hostEnvironment[name] !== undefined)
            .map((name) => [name, hostEnvironment[name]]),
    );
}
