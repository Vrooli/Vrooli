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

export type UpdateStateName = "offered" | "downloading" | "verified" | "apply_pending" | "applying" | "installing" | "recovering" | "relaunched" | "healthy" | "failed";

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
    transitions?: Array<{ state: UpdateStateName; observedAt: string }>;
};

// Recovery may observe either side of the external installer boundary. The
// intent remains authoritative even if the last state write was interrupted.
const updateTransitions: Record<UpdateStateName, readonly UpdateStateName[]> = {
    offered: ["offered", "downloading", "verified", "failed", "recovering", "relaunched"],
    downloading: ["downloading", "offered", "verified", "failed", "recovering", "relaunched"],
    verified: ["offered", "verified", "apply_pending", "failed", "recovering", "relaunched"],
    apply_pending: ["applying", "failed", "recovering", "relaunched"],
    applying: ["installing", "failed", "recovering", "relaunched"],
    installing: ["failed", "recovering", "relaunched"],
    recovering: ["offered", "failed", "recovering", "relaunched"],
    relaunched: ["healthy", "failed", "relaunched"],
    healthy: ["offered", "failed"],
    failed: ["offered", "failed", "recovering", "relaunched"],
};
export const updateStateNames = Object.keys(updateTransitions) as UpdateStateName[];

export function transitionUpdateState(current: UpdateStateName | "idle", requested: UpdateStateName): UpdateStateName {
    const allowed = current === "idle" ? ["offered", "recovering", "relaunched", "failed"] : updateTransitions[current];
    if (!allowed?.includes(requested)) throw new Error(`invalid update transition: ${current} -> ${requested}`);
    return requested;
}

/** Serialize event callbacks into the existing atomic state file. */
export function createUpdateStateWriter(fs: UpdateRecoveryFileSystem, statePath: string) {
    let tail = Promise.resolve();
    return (next: UpdateState): Promise<void> => {
        const operation = tail.then(async () => {
            let previous: UpdateState | undefined;
            try {
                previous = JSON.parse(await fs.readFile(statePath, "utf-8")) as UpdateState;
                if (!previous || !Object.prototype.hasOwnProperty.call(updateTransitions, previous.state)) throw new Error("invalid persisted update state");
            } catch (error) {
                if ((error as NodeJS.ErrnoException).code !== "ENOENT") throw error;
            }
            transitionUpdateState(previous?.state ?? "idle", next.state);
            const history = next.state === "offered" && previous?.state !== "offered" ? [] : previous?.transitions ?? [];
            // Progress callbacks do not consume trace capacity. A new offer
            // starts a new attempt; retain only this bounded lifecycle history.
            const transitions = previous?.state === next.state ? history : [...history, { state: next.state, observedAt: next.observedAt }].slice(-16);
            await writeUpdateState(fs, statePath, { ...(next.state === "offered" ? {} : previous), ...next, transitions });
        });
        tail = operation.catch(() => undefined);
        return operation;
    };
}

export type UpdateRecovery =
    | { disposition: "none" }
    | { disposition: "applied"; intent: UpdateIntent }
    | { disposition: "interrupted"; intent: UpdateIntent }
    | { disposition: "unknown"; intent: UpdateIntent }
    | { disposition: "corrupt" };

export function classifyUpdateRecovery(intent: UpdateIntent, currentVersion: string): Exclude<UpdateRecovery["disposition"], "none" | "corrupt"> {
    // A matching version is insufficient for a release-bound receipt. The
    // successor must carry the immutable artifact identity verified before
    // quitAndInstall.
    if (currentVersion === intent.toVersion && intent.toArtifactRef) return "applied";
    if (currentVersion === intent.fromVersion) return "interrupted";
    return "unknown";
}

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
    if (!state.observedAt || !Object.prototype.hasOwnProperty.call(updateTransitions, state.state)) {
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
    const disposition = classifyUpdateRecovery(intent, currentVersion);
    await clearUpdateIntent(fs, markerPath);
    return { disposition, intent };
}
