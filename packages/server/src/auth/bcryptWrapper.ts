/**
 * Hybrid bcrypt wrapper that falls back to bcryptjs if native bcrypt fails
 * This ensures authentication always works while maintaining performance when possible
 */
// AI_CHECK: TYPE_SAFETY=2 | LAST: 2025-07-03
import type * as bcryptTypes from "bcrypt";

interface BcryptService {
    hash(data: string, rounds: number): Promise<string>;
    hashSync(data: string, rounds: number): string;
    compare(data: string, encrypted: string): Promise<boolean>;
    compareSync(data: string, encrypted: string): boolean;
    isAvailable: boolean;
    implementation: "bcrypt" | "bcryptjs";
}

class BcryptServiceImpl implements BcryptService {
    public readonly isAvailable = true;
    public readonly implementation: "bcrypt" | "bcryptjs";
    private bcryptLib: typeof bcryptTypes;

    constructor(bcryptLib: typeof bcryptTypes, implementation: "bcrypt" | "bcryptjs") {
        this.bcryptLib = bcryptLib;
        this.implementation = implementation;
    }

    async hash(data: string, rounds: number): Promise<string> {
        return this.bcryptLib.hash(data, rounds);
    }

    hashSync(data: string, rounds: number): string {
        return this.bcryptLib.hashSync(data, rounds);
    }

    async compare(data: string, encrypted: string): Promise<boolean> {
        return this.bcryptLib.compare(data, encrypted);
    }

    compareSync(data: string, encrypted: string): boolean {
        return this.bcryptLib.compareSync(data, encrypted);
    }
}

let bcryptService: BcryptService | null = null;

function getBcryptService(): BcryptService {
    if (bcryptService) {
        return bcryptService;
    }

    // First try native bcrypt for better performance
    try {
        const bcrypt = require("bcrypt") as typeof bcryptTypes;
        bcryptService = new BcryptServiceImpl(bcrypt, "bcrypt");
        console.info("✅ Using native bcrypt for optimal performance");
        return bcryptService;
    } catch (bcryptError) {
        // Fall back to bcryptjs if native bcrypt fails
        try {
            const bcryptjs = require("bcryptjs") as typeof bcryptTypes;
            bcryptService = new BcryptServiceImpl(bcryptjs, "bcryptjs");
            console.warn("⚠️  Native bcrypt failed to load, using bcryptjs fallback. Performance may be reduced.");
            console.warn(`   Original error: ${bcryptError instanceof Error ? bcryptError.message : "Unknown error"}`);
            console.warn("   NOTE: This is expected during testing and does not affect functionality.");
            return bcryptService;
        } catch (bcryptjsError) {
            // Both failed - this is a critical error
            const errorMessage = `Both bcrypt and bcryptjs failed to load. bcrypt error: ${bcryptError instanceof Error ? bcryptError.message : "Unknown"}, bcryptjs error: ${bcryptjsError instanceof Error ? bcryptjsError.message : "Unknown"}`;
            console.error(`❌ ${errorMessage}`);
            throw new Error(errorMessage);
        }
    }
}

export { getBcryptService };
export type { BcryptService };
