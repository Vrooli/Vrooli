/** Every update installation shares this validation and durable apply boundary. */
export type UpdateTarget = { applicationId: string; serverURL: string; channel: string; platform: string; arch: string };
export type VerifiedUpdate = { version: string; artifactRef: string };
export type UpdateVerification = { ok: true; update: VerifiedUpdate } | { ok: false; reason: string };
type ApplyState =
    | { phase: "idle" }
    | { phase: "failed"; completion?: Promise<void> }
    | { phase: "pending"; update: VerifiedUpdate }
    | { phase: "applying"; completion: Promise<void> }
    | { phase: "installing" };

export function validateUpdateBinding(info: unknown, config: UpdateTarget): { ok: boolean; reason?: string } {
    if (typeof info !== "object" || info === null) return { ok: false, reason: "update metadata is not an object" };
    const binding = (info as { vrooli?: unknown }).vrooli;
    if (binding === undefined) return { ok: true };
    if (typeof binding !== "object" || binding === null) return { ok: false, reason: "update binding is malformed" };
    const values = binding as { applicationId?: unknown; applicationKey?: unknown; channel?: unknown; platform?: unknown; arch?: unknown; architecture?: unknown; artifactRef?: unknown };
    if (typeof values.applicationId === "string" && values.applicationId !== config.applicationId) return { ok: false, reason: "application identity mismatch" };
    if (typeof values.applicationKey === "string") {
        try {
            const segments = new URL(config.serverURL).pathname.split("/").filter(Boolean);
            const updatesIndex = segments.lastIndexOf("updates");
            if (updatesIndex < 0 || !segments[updatesIndex + 1] || segments[updatesIndex + 1] !== values.applicationKey) return { ok: false, reason: "application key mismatch" };
        } catch {
            return { ok: false, reason: "update server URL is invalid" };
        }
    }
    if (typeof values.channel === "string" && values.channel !== config.channel) return { ok: false, reason: "update channel mismatch" };
    const expectedPlatform = config.platform === "darwin" ? "mac" : config.platform === "win32" ? "win" : config.platform;
    if (typeof values.platform === "string" && values.platform !== expectedPlatform && !(expectedPlatform === "win" && values.platform === "windows")) return { ok: false, reason: "target platform mismatch" };
    // LPBS owns `architecture` (Go amd64); Electron calls that machine x64.
    // Check both fields when a legacy feed also carries `arch`, so one cannot
    // override an incompatible or conflicting canonical identity.
    const normalizeArchitecture = (value: unknown): string | undefined => {
        if (typeof value !== "string" || !value.trim()) return undefined;
        return value.trim() === "amd64" ? "x64" : value.trim();
    };
    for (const architecture of [values.architecture, values.arch]) {
        if (architecture !== undefined && normalizeArchitecture(architecture) !== normalizeArchitecture(config.arch)) return { ok: false, reason: "target architecture mismatch" };
    }
    const advertisedDigest = (info as { sha512?: unknown }).sha512;
    if (typeof values.artifactRef === "string" && (typeof advertisedDigest !== "string" || !advertisedDigest.trim() || values.artifactRef !== `sha512:${advertisedDigest.trim()}`)) return { ok: false, reason: "artifact digest mismatch" };
    return { ok: true };
}


export function verifyDownloadedUpdate(info: unknown, config: UpdateTarget): UpdateVerification {
    const candidate = info as { version?: unknown; sha512?: unknown } | null;
    if (!candidate || typeof candidate.version !== "string" || !candidate.version.trim()) return { ok: false, reason: "successor version is missing" };
    if (typeof candidate.sha512 !== "string" || !candidate.sha512.trim()) return { ok: false, reason: "successor artifact digest is missing" };
    const binding = validateUpdateBinding(info, config);
    if (!binding.ok) return { ok: false, reason: binding.reason ?? "update binding mismatch" };
    return { ok: true, update: { version: candidate.version, artifactRef: `sha512:${candidate.sha512.trim()}` } };
}

export function createUpdateApplyTransition(options: {
    updater: { autoInstallOnAppQuit: boolean };
    target: UpdateTarget;
    prepare(update: VerifiedUpdate): Promise<void>;
    install(restart: boolean): void;
    onFailure(error: unknown): void;
}) {
    // electron-updater's quit hook does not consult application validation or
    // recovery markers. It must never own an install in this shell.
    options.updater.autoInstallOnAppQuit = false;
    let state: ApplyState = { phase: "idle" };
    const apply = (restart: boolean): Promise<void> => {
        if (state.phase === "applying") return state.completion;
        if (state.phase !== "pending") return Promise.resolve();
        const update = state.update;
        // Schedule preparation after publishing the applying state, including
        // when the preparation callback throws synchronously.
        const completion = Promise.resolve().then(async () => {
            await options.prepare(update);
            if (state.phase !== "applying" || state.completion !== completion) throw new Error("update application was cancelled by an updater failure");
            state = { phase: "installing" };
            options.install(restart);
        }).catch(error => {
            state = { phase: "failed" };
            throw error;
        });
        state = { phase: "applying", completion };
        return completion;
    };
    return {
        get phase() { return state.phase; },
        failed() { state = { phase: "failed", completion: state.phase === "applying" || state.phase === "failed" ? state.completion : undefined }; },
        downloaded(info: unknown): UpdateVerification {
            if (state.phase === "applying" || state.phase === "installing" || (state.phase === "failed" && state.completion)) return { ok: false, reason: "update installation is already in progress" };
            const result = verifyDownloadedUpdate(info, options.target);
            state = result.ok ? { phase: "pending", update: result.update } : { phase: "failed" };
            return result;
        },
        apply,
        beforeQuit(event: { preventDefault(): void }): boolean {
            if (state.phase !== "pending" && state.phase !== "applying") return false;
            event.preventDefault();
            void apply(false).catch(options.onFailure);
            return true;
        },
    };
}
