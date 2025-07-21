/**
 * Type-safe ID conversion utilities for handling bigint/string conversions
 * across the codebase, particularly for database operations.
 */

/**
 * Convert a bigint ID to string for serialization/transport
 */
export function idToString(id: bigint): string {
    return id.toString();
}

/**
 * Convert a string ID back to bigint for database operations
 */
export function stringToId(id: string): bigint {
    return BigInt(id);
}

/**
 * Type guard to check if a value is a valid bigint ID
 */
export function isBigIntId(value: unknown): value is bigint {
    return typeof value === "bigint" && value >= 0n;
}

/**
 * Type guard to check if a value can be converted to a bigint ID
 */
export function isValidIdString(value: unknown): value is string {
    if (typeof value !== "string") return false;
    try {
        const id = BigInt(value);
        return id >= 0n;
    } catch {
        return false;
    }
}

/**
 * Convert ID to appropriate type based on context
 * Useful for handling mixed ID types in legacy code
 */
export function normalizeId(id: string | number | bigint): bigint {
    if (typeof id === "bigint") return id;
    if (typeof id === "string") return BigInt(id);
    if (typeof id === "number") return BigInt(id);
    throw new Error(`Invalid ID type: ${typeof id}`);
}

/**
 * Create a Prisma connect object with proper ID type
 */
export function connectById(id: string | bigint): { connect: { id: bigint } } {
    return {
        connect: { id: typeof id === "string" ? BigInt(id) : id },
    };
}

/**
 * Create a Prisma connect object for optional relationships
 */
export function connectByIdOptional(id?: string | bigint | null): { connect: { id: bigint } } | undefined {
    if (!id) return undefined;
    return connectById(id);
}

/**
 * Helper for handling IDs in test fixtures
 * Ensures consistent ID generation and type conversion
 */
export function fixtureId(id?: bigint | string | null): bigint | undefined {
    if (!id) return undefined;
    return typeof id === "string" ? BigInt(id) : id;
}
