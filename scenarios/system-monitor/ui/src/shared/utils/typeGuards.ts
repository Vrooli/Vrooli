/**
 * Shared type-narrowing helpers for safely reading unknown payloads.
 */

/** Narrow an unknown value to string, or undefined. */
export const str = (v: unknown): string | undefined => (typeof v === 'string' ? v : undefined);

/** Narrow an unknown value to number, or undefined. */
export const num = (v: unknown): number | undefined => (typeof v === 'number' ? v : undefined);

/** Narrow an unknown value to boolean, or undefined. */
export const bool = (v: unknown): boolean | undefined => (typeof v === 'boolean' ? v : undefined);
