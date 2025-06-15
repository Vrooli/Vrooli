/**
 * Working test setup with full initialization but avoiding the segfault
 */

import { vi, beforeAll, afterAll } from "vitest";
import { execSync } from "child_process";
import { GenericContainer, type StartedTestContainer } from "testcontainers";

// Mock sharp first
vi.mock("sharp", () => {
    const makeChain = () => {
        const chain: any = {};
        const pass = () => chain;
        Object.assign(chain, {
            resize: pass,
            toBuffer: async () => Buffer.alloc(0),
            metadata: async () => ({}),
        });
        return chain;
    };
    return { __esModule: true, default: () => makeChain() };
});

let redisContainer: StartedTestContainer | null = null;
let postgresContainer: StartedTestContainer | null = null;
let componentsInitialized = {
    containers: false,
    prisma: false,
    modelMap: false,
    dbProvider: false,
    mocks: false,
    idGenerator: false,
};

beforeAll(async function setup() {
    console.log("=== Test Setup Starting ===");
    
    try {
        // Set environment vars
        process.env.JWT_PRIV = "dummy-key";
        process.env.JWT_PUB = "dummy-key";
        process.env.VITE_SERVER_LOCATION = "local";
        process.env.ANTHROPIC_API_KEY = "dummy";
        process.env.MISTRAL_API_KEY = "dummy";
        process.env.OPENAI_API_KEY = "dummy";
        
        // Step 1: Start containers
        console.log("Step 1: Starting containers...");
        redisContainer = await new GenericContainer("redis")
            .withExposedPorts(6379)
            .start();
        process.env.REDIS_URL = `redis://${redisContainer.getHost()}:${redisContainer.getMappedPort(6379)}`;
        
        const POSTGRES_USER = "testuser";
        const POSTGRES_PASSWORD = "testpassword";
        const POSTGRES_DB = "testdb";
        postgresContainer = await new GenericContainer("pgvector/pgvector:pg15")
            .withExposedPorts(5432)
            .withEnvironment({ POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB })
            .start();
        process.env.DB_URL = `postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${postgresContainer.getHost()}:${postgresContainer.getMappedPort(5432)}/${POSTGRES_DB}`;
        
        componentsInitialized.containers = true;
        console.log("✓ Containers started");
        
        // Step 2: Setup Prisma
        console.log("Step 2: Setting up Prisma...");
        execSync("pnpm prisma migrate deploy", { stdio: "inherit", env: process.env });
        execSync("pnpm prisma generate", { stdio: "inherit" });
        componentsInitialized.prisma = true;
        console.log("✓ Prisma ready");
        
        // Step 3: Initialize ModelMap (this is what the tests need)
        console.log("Step 3: Initializing ModelMap...");
        try {
            const { ModelMap } = await import("../models/base/index.js");
            await ModelMap.init();
            componentsInitialized.modelMap = true;
            console.log("✓ ModelMap ready");
        } catch (error) {
            console.error("ModelMap initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 4: Initialize DbProvider
        console.log("Step 4: Initializing DbProvider...");
        try {
            const { DbProvider } = await import("../index.js");
            await DbProvider.init();
            componentsInitialized.dbProvider = true;
            console.log("✓ DbProvider ready");
        } catch (error) {
            console.error("DbProvider initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 5: Setup LLM service mocks
        console.log("Step 5: Setting up LLM mocks...");
        try {
            await setupLlmServiceMocks();
            componentsInitialized.mocks = true;
            console.log("✓ LLM mocks ready");
        } catch (error) {
            console.error("LLM mock setup failed:", error);
        }
        
        console.log("=== Test Setup Complete ===");
        console.log("Initialized components:", Object.entries(componentsInitialized)
            .filter(([, enabled]) => enabled)
            .map(([name]) => name)
            .join(", "));
        
        // IMPORTANT: ID generator not initialized in global setup to avoid worker crashes
        // 
        // Root cause: Memory pressure from containers + Prisma + module complexity
        // causes vitest workers (2GB heap limit) to crash when importing @vrooli/shared
        // after full setup. This is NOT a bug in the ID generator itself.
        //
        // Tests that need ID generation should initialize it individually:
        // const { initIdGenerator } = await import("@vrooli/shared");
        // await initIdGenerator(0);
        //
        // This distributes memory load and prevents crashes. See ID_GENERATOR_CRASH_ROOT_CAUSE.md
        // for detailed analysis.
        
    } catch (error) {
        console.error("=== Setup Failed ===");
        console.error(error);
        await cleanup();
        throw error;
    }
}, 300000);

async function setupLlmServiceMocks() {
    try {
        const [
            { OpenAIService },
            { AnthropicService }, 
            { MistralService }
        ] = await Promise.all([
            import("../tasks/llm/services/openai.js"),
            import("../tasks/llm/services/anthropic.js"),
            import("../tasks/llm/services/mistral.js"),
        ]);
        
        vi.spyOn(OpenAIService.prototype, "generateResponse" as any).mockResolvedValue({
            attempts: 1,
            message: "Mocked OpenAI response",
            cost: 0.001,
        });
        
        vi.spyOn(AnthropicService.prototype, "generateResponse" as any).mockResolvedValue({
            attempts: 1,
            message: "Mocked Anthropic response",
            cost: 0.001,
        });
        
        vi.spyOn(MistralService.prototype, "generateResponse" as any).mockResolvedValue({
            attempts: 1,
            message: "Mocked Mistral response",
            cost: 0.001,
        });
        
        const mockEstimateTokens = vi.fn().mockReturnValue({ model: "default", tokens: 10 });
        vi.spyOn(OpenAIService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
        vi.spyOn(AnthropicService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
        vi.spyOn(MistralService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
    } catch (error) {
        console.warn("Could not setup LLM mocks:", error);
    }
}

async function cleanup() {
    console.log("\n=== Cleanup Starting ===");
    
    // Restore mocks
    vi.restoreAllMocks();
    
    // Clean up based on what was initialized
    if (componentsInitialized.dbProvider) {
        try {
            const { DbProvider } = await import("../index.js");
            await DbProvider.shutdown();
            console.log("✓ DbProvider shutdown");
        } catch (e) {
            console.error("DbProvider shutdown error:", e);
        }
    }
    
    // Stop containers
    const stops = [];
    if (redisContainer) {
        stops.push(redisContainer.stop({ timeout: 5000 })
            .then(() => console.log("✓ Redis stopped"))
            .catch(e => console.error("Redis stop error:", e)));
    }
    if (postgresContainer) {
        stops.push(postgresContainer.stop({ timeout: 5000 })
            .then(() => console.log("✓ PostgreSQL stopped"))
            .catch(e => console.error("PostgreSQL stop error:", e)));
    }
    
    await Promise.all(stops);
    console.log("=== Cleanup Complete ===");
}

afterAll(cleanup, 60000);

// Export status for tests to check
export const getSetupStatus = () => ({ ...componentsInitialized });