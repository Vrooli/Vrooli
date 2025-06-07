import { initIdGenerator } from "@vrooli/shared";
import { beforeAll, afterAll, vi } from "vitest";
import { execSync } from "child_process";
import { generateKeyPairSync } from "crypto";
import * as http from "http";
import * as https from "https";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import { DbProvider } from "../../../db/provider.js";
import { closeRedis, initializeRedis } from "../../../redisConn.js";
import { AnthropicService } from "../../../tasks/llm/services/anthropic.js";
import { MistralService } from "../../../tasks/llm/services/mistral.js";
import { OpenAIService } from "../../../tasks/llm/services/openai.js";
import { QueueService } from "../../../tasks/queues.js";
import { setupTaskQueues } from "../../../tasks/setup.js";
import { initSingletons } from "../../../utils/singletons.js";

const SETUP_TIMEOUT_MS = 120_000;
const TEARDOWN_TIMEOUT_MS = 60_000;

let redisContainer: StartedTestContainer;
let postgresContainer: StartedTestContainer;

beforeAll(async () => {
    // Set up environment variables
    const { publicKey, privateKey } = generateKeyPairSync("rsa", {
        modulusLength: 2048,
        publicKeyEncoding: {
            type: "spki",
            format: "pem",
        },
        privateKeyEncoding: {
            type: "pkcs8",
            format: "pem",
        },
    });
    process.env.JWT_PRIV = privateKey;
    process.env.JWT_PUB = publicKey;
    process.env.ANTHROPIC_API_KEY = "dummy";
    process.env.MISTRAL_API_KEY = "dummy";
    process.env.OPENAI_API_KEY = "dummy";
    process.env.VITE_SERVER_LOCATION = "local";

    // Start the Redis container
    redisContainer = await new GenericContainer("redis")
        .withExposedPorts(6379)
        .start();
    // Set the REDIS_URL environment variable
    const redisHost = redisContainer.getHost();
    const redisPort = redisContainer.getMappedPort(6379);
    process.env.REDIS_URL = `redis://${redisHost}:${redisPort}`;

    // Start the PostgreSQL container
    const POSTGRES_USER = "testuser";
    const POSTGRES_PASSWORD = "testpassword";
    const POSTGRES_DB = "testdb";
    postgresContainer = await new GenericContainer("pgvector/pgvector:pg15")
        .withExposedPorts(5432)
        .withEnvironment({
            POSTGRES_USER,
            POSTGRES_PASSWORD,
            POSTGRES_DB,
        })
        .start();
    // Set the POSTGRES_URL environment variable
    const postgresHost = postgresContainer.getHost();
    const postgresPort = postgresContainer.getMappedPort(5432);
    process.env.DB_URL = `postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${postgresHost}:${postgresPort}/${POSTGRES_DB}`;

    // Apply Prisma migrations and generate client
    try {
        console.info("Applying Prisma migrations...");
        execSync("pnpm prisma migrate deploy", { stdio: "inherit" });
        console.info("Generating Prisma client...");
        execSync("pnpm prisma generate", { stdio: "inherit" });
        console.info("Database setup complete.");
    } catch (error) {
        console.error("Failed to set up database:", error);
        throw error; // Fail the test setup if migrations fail
    }

    // Set snowflake worker ID
    initIdGenerator(parseInt(process.env.WORKER_ID ?? "0"));

    // Initialize singletons
    await initSingletons();
    // Setup queues
    await setupTaskQueues();
    // Setup databases
    await initializeRedis();
    await DbProvider.init();

    // Add vitest mocks for LLM services
    setupLlmServiceMocks();
}, SETUP_TIMEOUT_MS);

function setupLlmServiceMocks() {
    // Mock OpenAI service
    vi.spyOn(OpenAIService.prototype, "generateResponse").mockResolvedValue({
        attempts: 1,
        message: "Mocked OpenAI response",
        cost: 0.001,
    });

    // Mock Anthropic service
    vi.spyOn(AnthropicService.prototype, "generateResponse").mockResolvedValue({
        attempts: 1,
        message: "Mocked Anthropic response",
        cost: 0.001,
    });

    // Mock Mistral service
    vi.spyOn(MistralService.prototype, "generateResponse").mockResolvedValue({
        attempts: 1,
        message: "Mocked Mistral response",
        cost: 0.001,
    });

    // Ensure all methods that might make network requests are properly mocked
    const mockEstimateTokens = vi.fn().mockReturnValue({ model: "default", tokens: 10 });
    vi.spyOn(OpenAIService.prototype, "estimateTokens").mockImplementation(mockEstimateTokens);
    vi.spyOn(AnthropicService.prototype, "estimateTokens").mockImplementation(mockEstimateTokens);
    vi.spyOn(MistralService.prototype, "estimateTokens").mockImplementation(mockEstimateTokens);
}

afterAll(async () => {
    // Restore all vitest mocks
    vi.restoreAllMocks();

    // Properly close all task queues and their Redis connections
    await QueueService.get().shutdown();

    // Close the Redis client connection
    await closeRedis();

    // Close database connection
    await DbProvider.shutdown();

    // Destroy HTTP/HTTPS agents more aggressively
    if (http.globalAgent) {
        http.globalAgent.destroy();
    }
    if (https.globalAgent) {
        https.globalAgent.destroy();
    }

    // Force close all sockets in the global agents
    function forceCloseAgentSockets(agent: any) {
        if (!agent || !agent.sockets) return;
        Object.keys(agent.sockets).forEach(key => {
            agent.sockets[key].forEach((socket: any) => {
                try {
                    socket.destroy();
                } catch (err) {
                    console.error("Error destroying socket:", err);
                }
            });
        });
    }

    forceCloseAgentSockets(http.globalAgent);
    forceCloseAgentSockets(https.globalAgent);

    // Stop containers with more forceful options
    try {
        // Stop the Redis container
        if (redisContainer) {
            // Use a shorter timeout and remove volumes
            await redisContainer.stop({
                timeout: 10000, // 10 seconds timeout
                removeVolumes: true,
            });
        }
    } catch (error) {
        console.error("Error stopping Redis container:", error);
    }

    try {
        // Stop the Postgres container
        if (postgresContainer) {
            // Use a shorter timeout and remove volumes
            await postgresContainer.stop({
                timeout: 10000, // 10 seconds timeout
                removeVolumes: true,
            });
        }
    } catch (error) {
        console.error("Error stopping Postgres container:", error);
    }
}, TEARDOWN_TIMEOUT_MS);