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
let initializationPromise: Promise<BcryptService> | null = null;

/**
 * Asynchronously initialize the bcrypt service using dynamic imports
 */
async function initializeBcryptService(): Promise<BcryptService> {
    if (bcryptService) {
        return bcryptService;
    }

    // First try native bcrypt for better performance
    try {
        const bcrypt = await import("bcrypt") as typeof bcryptTypes;
        bcryptService = new BcryptServiceImpl(bcrypt, "bcrypt");
        return bcryptService;
    } catch (bcryptError) {
        // Fall back to bcryptjs if native bcrypt fails
        try {
            const bcryptjs = await import("bcryptjs") as typeof bcryptTypes;
            bcryptService = new BcryptServiceImpl(bcryptjs, "bcryptjs");
            // Fallback to bcryptjs is expected during testing
            return bcryptService;
        } catch (bcryptjsError) {
            // Both failed - this is a critical error
            const errorMessage = `Both bcrypt and bcryptjs failed to load. bcrypt error: ${bcryptError instanceof Error ? bcryptError.message : "Unknown"}, bcryptjs error: ${bcryptjsError instanceof Error ? bcryptjsError.message : "Unknown"}`;
            console.error(`❌ ${errorMessage}`);
            throw new Error(errorMessage);
        }
    }
}

/**
 * Get the bcrypt service instance. This function ensures the service is initialized
 * before returning it. For synchronous contexts where async isn't possible,
 * make sure to call initializeBcryptService() during startup.
 */
function getBcryptService(): BcryptService {
    if (bcryptService) {
        return bcryptService;
    }
    
    // Log the error for debugging
    console.error("BcryptService not initialized when getBcryptService() was called");
    
    // This should not happen if initializeBcryptService() was called during startup
    throw new Error("BcryptService not initialized. Call initializeBcryptService() during application startup.");
}

/**
 * Get the bcrypt service instance asynchronously. This ensures proper initialization
 * with dynamic imports.
 */
async function getBcryptServiceAsync(): Promise<BcryptService> {
    if (bcryptService) {
        return bcryptService;
    }
    
    // Use a single initialization promise to avoid race conditions
    if (!initializationPromise) {
        initializationPromise = initializeBcryptService();
    }
    
    return initializationPromise;
}

export { getBcryptService, getBcryptServiceAsync, initializeBcryptService };
export type { BcryptService };
