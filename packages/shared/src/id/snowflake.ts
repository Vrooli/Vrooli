/* eslint-disable no-magic-numbers */
import { customAlphabet } from "nanoid";

let generator: { next(): bigint } | null = null;
const BIGINT_64_MAX = (2n ** 64n) - 1n;

/**
 * Initialise the ID generator once per process.
 * Must be awaited (or called synchronously in `main()`).
 */
export async function initIdGenerator(workerId = 1, epoch = 1609459200000) {
    if (typeof process !== "undefined" && process.versions?.node) {
        /* ---------- Node path: Rust/WASM fast generator ---------- */
        const ns = (await import("nodejs-snowflake")) as any;
        const Snowflake = ns.Snowflake ?? ns.default?.Snowflake ?? ns.default;

        const nodeGen = new Snowflake({
            instance_id: workerId,
            custom_epoch: epoch,
        });

        generator = { next: () => nodeGen.getUniqueID() };
    } else {
        /** Browser / JSDOM – pure‑JS fallback */
        generator = {
            next: () => {
                const numberString = customAlphabet("0123456789", 12)();
                return BigInt(numberString);
            },
        };
    }
}

/** Placeholder used only in fixtures; replace before persist */
export const DUMMY_ID = "0";

/** Generate the next Snowflake ID (BigInt).  */
export function generatePK(): bigint {
    if (!generator) {
        throw new Error("Snowflake generator not initialised - call initIdGenerator() first");
    }
    return generator.next();
}

/** True iff `id` is a positive unsigned‑64 BigInt. */
export function validatePK(id: unknown): boolean {
    if (typeof id !== "string" && typeof id !== "bigint") return false;
    try {
        const val = typeof id === "string" ? BigInt(id) : id;
        return val > 0n && val <= BIGINT_64_MAX;
    } catch {
        return false;
    }
}
