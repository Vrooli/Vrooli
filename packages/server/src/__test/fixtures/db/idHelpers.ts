/**
 * Helper file to provide ID generation functions
 * This works around TypeScript resolution issues with @vrooli/shared
 * 
 * NOTE: These implementations match the logic from @vrooli/shared/id
 * but are duplicated here to avoid TypeScript module resolution issues
 * during test compilation.
 */

// Constants matching @vrooli/shared/id/publicId.ts
const PUBLIC_ID_LENGTH = 12;
const ALPHABET = "0123456789abcdefghijklmnopqrstuvwxyz";

/**
 * Generate a snowflake-like ID (BigInt)
 * Matches the behavior of @vrooli/shared generatePK()
 */
export function generatePK(): bigint {
    // Generate a snowflake-like ID using timestamp + random component
    const timestamp = BigInt(Date.now());
    const random = BigInt(Math.floor(Math.random() * 1000));
    return (timestamp << BigInt(10)) | random;
}

/**
 * Generate a human-friendly public ID for URLs
 * Matches the behavior of @vrooli/shared generatePublicId()
 */
export function generatePublicId(length: number = PUBLIC_ID_LENGTH): string {
    const chars = ALPHABET;
    let result = "";
    for (let i = 0; i < length; i++) {
        result += chars[Math.floor(Math.random() * chars.length)];
    }
    return result;
}

/**
 * Generate a nanoid-like string
 * Matches the behavior of @vrooli/shared nanoid()
 */
export function nanoid(size: number = PUBLIC_ID_LENGTH): string {
    return generatePublicId(size);
}

/**
 * Dynamic import helper for runtime access to @vrooli/shared
 * Use this when you need the actual implementation at runtime
 */
export async function getSharedIdFunctions() {
    try {
        // Try to import the actual functions at runtime
        const sharedModule = await import("@vrooli/shared") as any;
        return {
            generatePK: (sharedModule?.generatePK && typeof sharedModule.generatePK === "function") ? sharedModule.generatePK : generatePK,
            generatePublicId: (sharedModule?.generatePublicId && typeof sharedModule.generatePublicId === "function") ? sharedModule.generatePublicId : generatePublicId,
            nanoid: (sharedModule?.nanoid && typeof sharedModule.nanoid === "function") ? sharedModule.nanoid : nanoid,
        };
    } catch (error) {
        // Fallback to local implementations
        console.warn("Failed to import @vrooli/shared, using local implementations");
        return {
            generatePK,
            generatePublicId,
            nanoid,
        };
    }
}
