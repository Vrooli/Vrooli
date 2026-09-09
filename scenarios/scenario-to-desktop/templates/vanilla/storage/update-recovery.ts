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
    applicationId?: string;
    channel?: string;
    platform?: string;
    arch?: string;
    fromArtifactRef?: string;
    toArtifactRef?: string;
    state?: "apply_pending" | "applying";
};

export type InstalledUpdateReceipt = {
    schemaVersion: 1;
    outcome: "applied" | "interrupted" | "unknown" | "corrupt" | "failed";
    applicationId?: string;
    channel?: string;
    platform?: string;
    arch?: string;
    fromVersion: string;
    toVersion: string;
    fromArtifactRef?: string;
    toArtifactRef?: string;
    installedExecutable: string;
    installedExecutableDigest?: string;
    health: "passed" | "failed";
    observedAt: string;
    reason?: string;
};

export type UpdateStateName = "offered" | "downloading" | "verified" | "apply_pending" | "applying" | "recovering" | "relaunched" | "healthy" | "failed";

export type UpdateState = {
    schemaVersion: 1;
    state: UpdateStateName;
    applicationId?: string;
    channel?: string;
    platform?: string;
    arch?: string;
    fromVersion?: string;
    toVersion?: string;
    artifactRef?: string;
    observedAt: string;
    reason?: string;
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
    const optionalString = (candidateValue: unknown, maxLength: number): string | undefined =>
        typeof candidateValue === "string" && candidateValue.length > 0 && candidateValue.length <= maxLength
            ? candidateValue
            : undefined;
    const state = candidate.state === "apply_pending" || candidate.state === "applying" ? candidate.state : undefined;
    const parsed: UpdateIntent = {
        schemaVersion: 1,
        fromVersion: candidate.fromVersion,
        toVersion: candidate.toVersion,
        requestedAt: candidate.requestedAt,
        nonce: candidate.nonce,
    };
    const optionalFields: Array<[keyof UpdateIntent, string | undefined]> = [
        ["applicationId", optionalString(candidate.applicationId, 256)],
        ["channel", optionalString(candidate.channel, 64)],
        ["platform", optionalString(candidate.platform, 32)],
        ["arch", optionalString(candidate.arch, 32)],
        ["fromArtifactRef", optionalString(candidate.fromArtifactRef, 512)],
        ["toArtifactRef", optionalString(candidate.toArtifactRef, 512)],
    ];
    for (const [key, value] of optionalFields) if (value !== undefined) parsed[key] = value as never;
    if (state !== undefined) parsed.state = state;
    return parsed;
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

export async function readUpdateIntent(fs: UpdateRecoveryFileSystem, markerPath: string): Promise<UpdateIntent | null> {
    try {
        const encoded = await fs.readFile(markerPath, "utf-8");
        if (Buffer.byteLength(encoded, "utf-8") > MAX_MARKER_BYTES) return null;
        try { return parseIntent(JSON.parse(encoded)); } catch { return null; }
    } catch (error: unknown) {
        const code = error instanceof Error && "code" in error ? (error as NodeJS.ErrnoException).code : undefined;
        if (code === "ENOENT") return null;
        return null;
    }
}

export async function markUpdateIntentApplying(fs: UpdateRecoveryFileSystem, markerPath: string): Promise<void> {
    const intent = await readUpdateIntent(fs, markerPath);
    if (!intent) throw new Error("update intent is missing or corrupt");
    await writeUpdateIntent(fs, markerPath, { ...intent, state: "applying" });
}

export async function clearUpdateIntent(fs: UpdateRecoveryFileSystem, markerPath: string): Promise<void> {
    try { await fs.unlink?.(markerPath); } catch { /* cleanup is best effort */ }
}

export async function writeInstalledUpdateReceipt(fs: UpdateRecoveryFileSystem, receiptPath: string, receipt: InstalledUpdateReceipt): Promise<void> {
    if (!receipt.fromVersion || !receipt.toVersion || !receipt.installedExecutable || !receipt.observedAt ||
        !["applied", "interrupted", "unknown", "corrupt", "failed"].includes(receipt.outcome) ||
        !["passed", "failed"].includes(receipt.health)) {
        throw new Error("invalid installed update receipt");
    }
    const encoded = JSON.stringify(receipt);
    if (Buffer.byteLength(encoded, "utf-8") > MAX_MARKER_BYTES) throw new Error("installed update receipt exceeds size limit");
    if (typeof fs.rename !== "function") {
        await fs.writeFile(receiptPath, encoded, "utf-8");
        return;
    }
    const temporary = `${receiptPath}.tmp-${++markerSequence}`;
    try {
        await fs.writeFile(temporary, encoded, "utf-8");
        await fs.rename(temporary, receiptPath);
    } catch (error) {
        try { await fs.unlink?.(temporary); } catch { /* preserve the original failure */ }
        throw error;
    }
}

export async function writeUpdateState(fs: UpdateRecoveryFileSystem, statePath: string, state: UpdateState): Promise<void> {
    if (!state.observedAt || !["offered", "downloading", "verified", "apply_pending", "applying", "recovering", "relaunched", "healthy", "failed"].includes(state.state)) {
        throw new Error("invalid update state");
    }
    const encoded = JSON.stringify({ ...state, schemaVersion: 1, state: state.state, observedAt: state.observedAt });
    if (Buffer.byteLength(encoded, "utf-8") > MAX_MARKER_BYTES) throw new Error("update state exceeds size limit");
    if (typeof fs.rename !== "function") {
        await fs.writeFile(statePath, encoded, "utf-8");
        return;
    }
    const temporary = `${statePath}.tmp-${++markerSequence}`;
    try {
        await fs.writeFile(temporary, encoded, "utf-8");
        await fs.rename(temporary, statePath);
    } catch (error) {
        try { await fs.unlink?.(temporary); } catch { /* preserve the original failure */ }
        throw error;
    }
}

export async function consumeUpdateIntent(fs: UpdateRecoveryFileSystem, markerPath: string, currentVersion: string): Promise<UpdateRecovery> {
    const intent = await readUpdateIntent(fs, markerPath);
    if (!intent) {
        let exists = true;
        try { await fs.readFile(markerPath, "utf-8"); } catch (error: unknown) {
            const code = error instanceof Error && "code" in error ? (error as NodeJS.ErrnoException).code : undefined;
            exists = code !== "ENOENT";
        }
        await clearUpdateIntent(fs, markerPath);
        return { disposition: exists ? "corrupt" : "none" };
    }
    const disposition = currentVersion === intent.toVersion
        ? "applied"
        : currentVersion === intent.fromVersion ? "interrupted" : "unknown";
    await clearUpdateIntent(fs, markerPath);
    return { disposition, intent };
}
