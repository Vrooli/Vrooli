/* eslint-disable no-magic-numbers */
import { customAlphabet } from "nanoid";

// Initialize with browser-compatible fallback by default.
// This allows synchronous use in UI without calling initIdGenerator.
let generator: { next(): bigint } = {
    next: () => {
        const numberString = customAlphabet("0123456789", 12)();
        return BigInt(numberString);
    },
};

const BIGINT_64_MAX = (2n ** 64n) - 1n;

/**
 * Initialise the ID generator, primarily for Node.js environments
 * to attempt using the native 'nodejs-snowflake' module.
 * In browser environments, the generator is already initialized with a fallback.
 * Must be awaited (or called synchronously in `main()`) if used in Node.js.
 */
export async function initIdGenerator(workerId = 1, epoch = 1609459200000) {
    if (typeof process !== "undefined" && process.versions?.node) {
        /* ---------- Node path: Rust/WASM fast generator ---------- */
        try {
            // Using a new Function constructor for dynamic import to bypass Vite's static analysis
            // during client-side builds, preventing resolution errors for 'nodejs-snowflake'.
            const moduleName = "nodejs-snowflake";
            const dynamicImport = new Function("modulePath", "return import(modulePath);");
            const ns = await dynamicImport(moduleName) as any;

            const Snowflake = ns.Snowflake ?? ns.default?.Snowflake ?? ns.default;

            if (!Snowflake) {
                console.warn(
                    "nodejs-snowflake module loaded, but Snowflake constructor not found. Using pure JS ID generator.",
                );
                // Keep the default (already set) pure JS generator
                return;
            }

            const nodeGen = new Snowflake({
                instance_id: workerId,
                custom_epoch: epoch,
            });
            // Successfully initialized Node.js generator
            generator = { next: () => nodeGen.getUniqueID() };
        } catch (error) {
            console.warn(
                "Failed to initialize native Snowflake generator in Node.js environment, using pure JS version. Error:",
                error,
            );
            // Fallback to the pure JS generator is already the default, so no change needed here if an error occurs.
            // If for some reason generator was nullified, re-initialize:
            if (!generator) {
                generator = {
                    next: () => {
                        const numberString = customAlphabet("0123456789", 12)();
                        return BigInt(numberString);
                    },
                };
            }
        }
    }
    // No 'else' block needed, as the generator is already initialized with the browser-compatible version by default.
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
