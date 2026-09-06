/**
 * Crash-safe update intent persistence.
 *
 * The updater cannot make an external application replacement transactional.
 * This marker records the intended transition before quitAndInstall and is
 * consumed exactly once on the following launch. It contains versions and a
 * nonce only; it never contains credentials or user content.
 */

export type UpdateIntent = {
    schemaVersion: 1;
    fromVersion: string;
    toVersion: string;
    requestedAt: string;
    nonce: string;
};

export type UpdateRecovery =
    | { disposition: "none" }
    | { disposition: "applied"; intent: UpdateIntent }
    | { disposition: "interrupted"; intent: UpdateIntent }
    | { disposition: "unknown"; intent: UpdateIntent }
    | { disposition: "corrupt" };

export interface UpdateRecoveryFileSystem {
    readFile(path: string, encoding: "utf-8"): Promise<string>;
    writeFile(path: string, content: string, encoding?: "utf-8"): Promise<void>;
    rename?(from: string, to: string): Promise<void>;
    unlink?(path: string): Promise<void>;
}

const MAX_MARKER_BYTES = 4096;
let markerSequence = 0;

function validVersion(value: unknown): value is string {
    return typeof value === "string" && value.length > 0 && value.length <= 128 && /^[0-9A-Za-z.+-]+$/.test(value);
}

function parseIntent(value: unknown): UpdateIntent | null {
    if (typeof value !== "object" || value === null) return null;
    const candidate = value as Partial<UpdateIntent>;
    if (candidate.schemaVersion !== 1 || !validVersion(candidate.fromVersion) || !validVersion(candidate.toVersion) ||
        typeof candidate.requestedAt !== "string" || candidate.requestedAt.length > 64 ||
        typeof candidate.nonce !== "string" || candidate.nonce.length < 8 || candidate.nonce.length > 128) return null;
    return {
        schemaVersion: 1,
        fromVersion: candidate.fromVersion,
        toVersion: candidate.toVersion,
        requestedAt: candidate.requestedAt,
        nonce: candidate.nonce,
    };
}

export async function writeUpdateIntent(fs: UpdateRecoveryFileSystem, markerPath: string, intent: UpdateIntent): Promise<void> {
    const parsed = parseIntent(intent);
    if (!parsed) throw new Error("invalid update intent");
    const encoded = JSON.stringify(parsed);
    if (Buffer.byteLength(encoded, "utf-8") > MAX_MARKER_BYTES) throw new Error("update intent exceeds size limit");
    if (typeof fs.rename !== "function") {
        await fs.writeFile(markerPath, encoded, "utf-8");
        return;
    }
    const temporary = `${markerPath}.tmp-${++markerSequence}`;
    try {
        await fs.writeFile(temporary, encoded, "utf-8");
        await fs.rename(temporary, markerPath);
    } catch (error) {
        try { await fs.unlink?.(temporary); } catch { /* preserve the original failure */ }
        throw error;
    }
}

export async function consumeUpdateIntent(fs: UpdateRecoveryFileSystem, markerPath: string, currentVersion: string): Promise<UpdateRecovery> {
    let intent: UpdateIntent | null = null;
    try {
        const encoded = await fs.readFile(markerPath, "utf-8");
        if (Buffer.byteLength(encoded, "utf-8") > MAX_MARKER_BYTES) return { disposition: "corrupt" };
        try { intent = parseIntent(JSON.parse(encoded)); } catch { intent = null; }
        if (!intent) return { disposition: "corrupt" };
        if (currentVersion === intent.toVersion) return { disposition: "applied", intent };
        if (currentVersion === intent.fromVersion) return { disposition: "interrupted", intent };
        return { disposition: "unknown", intent };
    } catch (error: unknown) {
        const code = error instanceof Error && "code" in error ? (error as NodeJS.ErrnoException).code : undefined;
        if (code === "ENOENT") return { disposition: "none" };
        return { disposition: "corrupt" };
    } finally {
        try { await fs.unlink?.(markerPath); } catch { /* marker cleanup is best effort */ }
    }
}
