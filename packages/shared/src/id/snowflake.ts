/**
 * Snowflake ID implementation
 *
 * A 64‑bit, time‑ordered, distributed‑friendly ID generator
 * inspired by Twitter Snowflake.
 *
 * Layout (left → right):
 * ┌──────────41 bits───────────┬────10 bits────┬───12 bits───┐
 * │  milliseconds since epoch  │   worker ID   │   sequence   │
 * └────────────────────────────┴───────────────┴──────────────┘
 *
 * • Epoch: 2021‑01‑01 00:00:00 UTC
 * • 1024 distinct workers per deployment
 * • 4096 IDs / ms / worker
 *
 * In production you **must** call `setWorkerId()` with a stable
 * value (0‑1023) unique to the node / process / shard.
 */

// ─────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────
const EPOCH = 1609459200000;                     // Jan 1 2021 UTC
const WORKER_ID_BITS = 10;
const SEQUENCE_BITS = 12;

const MAX_WORKER_ID = (1 << WORKER_ID_BITS) - 1; // 1023
const MAX_SEQUENCE = (1 << SEQUENCE_BITS) - 1; // 4095

const TIMESTAMP_LEFT_SHIFT = WORKER_ID_BITS + SEQUENCE_BITS; // 22
const WORKER_ID_SHIFT = SEQUENCE_BITS;                  // 12

const BIGINT_64_MAX = (1n << 64n) - 1n;                      // 2⁶⁴−1

// ─────────────────────────────────────────────────────────────
// Worker‑ID initialisation (safe for Node *and* browser)
// ─────────────────────────────────────────────────────────────

// Helper: pseudo‑PID when `process.pid` is unavailable (browser)
function getDefaultPid(): number {
    if (typeof process !== "undefined" && typeof process.pid === "number") {
        return process.pid;
    }
    // Browser / non‑Node: pick a random 0‑999 for dev convenience
    return Math.floor(Math.random() * 1_000);
}

/**
 * Default worker ID:
 *   • low 2 digits of (real PID OR pseudo‑PID)  +
 *   • random 0‑899  → gives an easy‑to‑spot dev pattern
 *
 * *Do not rely on this in production.*
 */
let workerId =
    ((getDefaultPid() % 100) + Math.floor(Math.random() * 900)) & MAX_WORKER_ID;

// Runtime variables
let sequence = 0;
let lastTimestamp = -1;

// ─────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────

/**
 * Set a stable worker ID (0 – 1023) for this node/process.
 */
export function setWorkerId(id: number): void {
    if (id < 0 || id > MAX_WORKER_ID || !Number.isInteger(id)) {
        throw new Error(`Worker ID must be an integer between 0 and ${MAX_WORKER_ID}`);
    }
    workerId = id;
}

/**
 * Generate next Snowflake ID as a `bigint`.
 */
export function nextBigInt(): bigint {
    let timestamp = Date.now();

    // Clock moved backwards → refuse to generate IDs
    if (timestamp < lastTimestamp) {
        throw new Error(
            `Clock moved backwards by ${lastTimestamp - timestamp} ms — ID generation halted`
        );
    }

    if (timestamp === lastTimestamp) {
        // Same millisecond → bump sequence
        sequence = (sequence + 1) & MAX_SEQUENCE;
        if (sequence === 0) {
            // Sequence overflow → wait for next millisecond
            timestamp = waitNextMillis(lastTimestamp);
        }
    } else {
        // New millisecond → reset sequence
        sequence = 0;
    }

    lastTimestamp = timestamp;

    // Compose the 64‑bit ID
    return (BigInt(timestamp - EPOCH) << BigInt(TIMESTAMP_LEFT_SHIFT)) |
        (BigInt(workerId) << BigInt(WORKER_ID_SHIFT)) |
        BigInt(sequence);
}

/**
 * Validate that `id` is a positive 64‑bit integer.
 */
export function validatePK(id: unknown): boolean {
    if (typeof id !== "string" && typeof id !== "bigint") return false;
    try {
        const snowflake = typeof id === "string" ? BigInt(id) : id;
        return snowflake > 0n && snowflake <= BIGINT_64_MAX;
    } catch {
        return false;
    }
}

/**
 * Extract the original timestamp as a `Date`.
 */
export function getTimestampFromId(id: string | bigint): Date {
    const snowflake = typeof id === "string" ? BigInt(id) : id;
    const ts = Number((snowflake >> BigInt(TIMESTAMP_LEFT_SHIFT)) + BigInt(EPOCH));
    return new Date(ts);
}

/**
 * Convenience helpers mirroring your earlier API
 */
export const generatePK = nextBigInt;
export const generatePKString = () => nextBigInt().toString();

/**
 * Dummy placeholder ID for tests / schema defaults.
 * Be sure to replace before persisting to the database.
 */
export const DUMMY_ID = "0";

// ─────────────────────────────────────────────────────────────
// Internals
// ─────────────────────────────────────────────────────────────

function waitNextMillis(last: number): number {
    let ts = Date.now();
    while (ts <= last) ts = Date.now();
    return ts;
}
