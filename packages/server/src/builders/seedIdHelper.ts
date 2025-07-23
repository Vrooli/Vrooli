import { generatePK } from "@vrooli/shared";

/**
 * Handles ID generation during seeding operations.
 * 
 * When seeding data, IDs in source files are often nanoid placeholders 
 * (e.g., "ud0ay7uh4h6d") since generatePK() can't be called at module level.
 * This helper detects these placeholders and replaces them with proper snowflake IDs.
 * 
 * @param id - The ID from the seed data (could be nanoid string or numeric string)
 * @param isSeeding - Whether this is a seeding operation
 * @returns A BigInt ID suitable for database insertion
 */
export function seedId(id: string | number | bigint, isSeeding: boolean): bigint {
    // If not seeding, just convert to BigInt normally
    if (!isSeeding) {
        return BigInt(id);
    }

    // If already a bigint, return as-is
    if (typeof id === "bigint") {
        return id;
    }

    // During seeding, if it's a string that's not purely numeric, 
    // it's likely a placeholder that needs replacing
    if (typeof id === "string") {
        // Check if it's a numeric string (can be converted to BigInt)
        try {
            // If it successfully converts, it's a numeric string
            return BigInt(id);
        } catch {
            // If conversion fails, it's a placeholder - generate new ID
            console.log("[seedId] Replacing placeholder ID:", id, "with new snowflake ID");
            return generatePK();
        }
    }

    // For numbers, convert directly
    return BigInt(id);
}

