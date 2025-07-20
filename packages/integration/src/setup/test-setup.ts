// Mock browser APIs that don't exist in Node environment
// This must be done before any imports that might use these APIs
(global as any).File = class File {
    public bits: any[];
    public name: string;
    public type: string;
    public lastModified: number;
    public size: number;

    constructor(bits: any[], name: string, options: { type?: string; lastModified?: number } = {}) {
        this.bits = bits;
        this.name = name;
        this.type = options.type || "";
        this.lastModified = options.lastModified || Date.now();
        // Calculate size from bits
        this.size = bits.reduce((acc, bit) => {
            if (typeof bit === "string") return acc + bit.length;
            if (bit instanceof ArrayBuffer) return acc + bit.byteLength;
            if (bit instanceof Uint8Array) return acc + bit.length;
            return acc;
        }, 0);
    }

    // Mock some File methods
    async text() {
        return this.bits.map(bit => {
            if (typeof bit === "string") return bit;
            return new TextDecoder().decode(bit);
        }).join("");
    }

    slice(start?: number, end?: number, contentType?: string) {
        // Simple slice implementation
        const text = this.bits.join("");
        const sliced = text.slice(start, end);
        return new File([sliced], this.name, { type: contentType || this.type });
    }
};

// Mock Blob as well since File extends Blob
(global as any).Blob = class Blob {
    public bits: any[];
    public type: string;
    public size: number;

    constructor(bits: any[] = [], options: { type?: string } = {}) {
        this.bits = bits;
        this.type = options.type || "";
        this.size = bits.reduce((acc, bit) => {
            if (typeof bit === "string") return acc + bit.length;
            if (bit instanceof ArrayBuffer) return acc + bit.byteLength;
            return acc;
        }, 0);
    }
};

import { exec } from "child_process";
import { promisify } from "util";
import { afterAll, afterEach, beforeAll, beforeEach, expect } from "vitest";

const _execAsync = promisify(exec);

// Services will be imported when needed
let ModelMap: any;
let DbProvider: any;
let CacheService: any;
let QueueService: any;
let BusService: any;
let prisma: any;

const componentsInitialized = {
    modelMap: false,
    dbProvider: false,
    idGenerator: false,
    migrations: false,
};

beforeAll(async function setup() {
    console.log("=== Integration Test Setup Starting ===");

    try {
        // Debug environment variables
        console.log("Environment check:", {
            REDIS_URL: process.env.REDIS_URL ? "SET" : "NOT SET",
            DB_URL: process.env.DB_URL ? "SET" : "NOT SET",
            DATABASE_URL: process.env.DATABASE_URL ? "SET" : "NOT SET",
            NODE_ENV: process.env.NODE_ENV,
            VITEST: process.env.VITEST,
        });

        // Step 1: Skip ID generator initialization to avoid memory issues
        // As noted in server setup, vitest workers have 2GB heap limit and
        // importing @vrooli/shared after full setup causes crashes.
        // Tests that need ID generation should initialize it individually.
        console.log("Step 1: Skipping ID generator (will be initialized per-test as needed)");
        console.log("⚠ ID generator not initialized globally - tests should init as needed");

        // Step 2: Initialize ModelMap (required by DbProvider)
        console.log("Step 2: Initializing ModelMap...");
        try {
            ({ ModelMap } = await import("@vrooli/server"));
            await ModelMap.init();
            componentsInitialized.modelMap = true;
            console.log("✓ ModelMap ready");
        } catch (error) {
            console.error("ModelMap initialization failed:", error);
            // Don't throw - continue without it
        }

        // Step 3: Initialize DbProvider
        console.log("Step 3: Initializing DbProvider...");
        try {
            ({ DbProvider } = await import("@vrooli/server"));
            await DbProvider.init();
            componentsInitialized.dbProvider = true;
            console.log("✓ DbProvider ready");
        } catch (error) {
            console.error("DbProvider initialization failed:", error);
            // Don't throw - continue without it
        }

        // Step 4: Migrations are already run by global setup
        console.log("Step 4: Migrations already applied by global setup");
        componentsInitialized.migrations = true;

        // Step 5: Initialize other services (non-critical)
        console.log("Step 5: Initializing optional services...");
        try {
            ({ CacheService } = await import("@vrooli/server"));
            CacheService.get();
            console.log("✓ CacheService ready");
        } catch (e) {
            console.log("⚠ CacheService initialization skipped");
        }

        try {
            ({ QueueService } = await import("@vrooli/server"));
            console.log("✓ QueueService imported");
        } catch (e) {
            console.log("⚠ QueueService import skipped");
        }

        try {
            ({ BusService } = await import("@vrooli/server"));
            console.log("✓ BusService imported");
        } catch (e) {
            console.log("⚠ BusService import skipped");
        }

        console.log("=== Integration Test Setup Complete ===");
        console.log("Initialized components:", Object.entries(componentsInitialized)
            .filter(([, enabled]) => enabled)
            .map(([name]) => name)
            .join(", "));

    } catch (error) {
        console.error("=== Setup Failed ===");
        console.error(error);
        await cleanup();
        throw error;
    }
}, 300000); // 5 minute timeout

beforeEach(async () => {
    // Clean database before each test
    // This ensures test isolation
    if (prisma) {
        await cleanDatabase();
    }
});

afterEach(async () => {
    // Additional cleanup if needed
});

afterAll(async () => {
    const testFile = expect.getState().testPath?.split("/").pop() || "unknown";
    console.log(`[${new Date().toISOString()}] afterAll cleanup starting for ${testFile}`);
    const timeoutId = setTimeout(() => {
        console.error(`[${new Date().toISOString()}] CLEANUP TIMEOUT WARNING for ${testFile} - cleanup taking too long!`);
    }, 30000);
    try {
        await cleanup();
        clearTimeout(timeoutId);
        console.log(`[${new Date().toISOString()}] afterAll cleanup completed for ${testFile}`);
    } catch (error) {
        clearTimeout(timeoutId);
        console.error(`[${new Date().toISOString()}] afterAll cleanup error for ${testFile}:`, error);
        throw error;
    }
});

async function cleanup() {
    console.log("\n=== Integration Test Cleanup Starting ===");

    // Reset singleton services to prevent stale connections
    console.log("Resetting services...");
    try {
        if (CacheService && CacheService.reset) {
            const timeoutPromise = new Promise((_, reject) =>
                setTimeout(() => reject(new Error("CacheService reset timeout")), 10000),
            );
            await Promise.race([CacheService.reset(), timeoutPromise]);
            console.log("✓ CacheService reset");
        }
    } catch (e) {
        console.log("⚠ CacheService not available or failed to reset:", e.message);
    }

    try {
        if (QueueService && QueueService.reset) {
            await QueueService.reset();
            console.log("✓ QueueService reset");
        }
    } catch (e) {
        console.log("⚠ QueueService not available or failed to reset:", e.message);
    }

    try {
        if (BusService && BusService.reset) {
            await BusService.reset();
            console.log("✓ BusService reset");
        }
    } catch (e) {
        console.log("⚠ BusService not available or failed to reset:", e.message);
    }

    // Clean up DbProvider
    if (componentsInitialized.dbProvider) {
        try {
            if (DbProvider && DbProvider.reset) {
                await DbProvider.reset();
                console.log("✓ DbProvider reset");
            }
        } catch (e) {
            console.log("⚠ DbProvider not available or failed to reset:", e.message);
        }
    }

    // Note: Containers are managed by global teardown
    console.log("=== Integration Test Cleanup Complete ===");
}

async function cleanDatabase() {
    if (!prisma) {
        console.warn("Cannot clean database: Prisma client not available");
        return;
    }

    try {
        // Get all table names except migrations
        const tables = await prisma.$queryRaw<Array<{ tablename: string }>>`
            SELECT tablename FROM pg_tables 
            WHERE schemaname = 'public' 
            AND tablename != '_prisma_migrations'
        `;

        // Truncate all tables
        for (const { tablename } of tables) {
            await prisma.$executeRawUnsafe(
                `TRUNCATE TABLE "public"."${tablename}" CASCADE`,
            );
        }
    } catch (error) {
        console.error("Database cleanup failed:", error);
        // Don't throw - test might still work
    }
}

export { cleanDatabase };

// Export status for tests to check
export function getSetupStatus() {
    return { ...componentsInitialized };
}
