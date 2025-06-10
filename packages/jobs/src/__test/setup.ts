/**
 * Vitest global test‑bed
 * --------------------------------------------------------------------------
 * This file is loaded once per worker before any test files execute.  Besides
 * spinning up Docker containers, seeding the database, and configuring stubs, it also **globally mocks the `sharp` module** (see below).  Placing
 * the mock at the *very top* guarantees it is applied before *any* import that
 * might transitively require `sharp`, preventing the native addon from ever
 * being loaded in the test environment.
 */

// ---------------------------------------------------------------------------
// 1.  GLOBAL MOCK FOR `sharp`
// ---------------------------------------------------------------------------
// Why we mock it:
//   • Vitest runs test files in isolated worker processes (threads or forks).
//   • `sharp` is a native Node‑API addon; loading it more than once inside a
//     single process, or inside Vitest’s thread pool, often fails with
//       "Cannot find module '../build/Release/sharp-*.node'"  or
//       "The module did not self‑register".
//   • Our unit tests don’t need real image manipulation—returning empty
//     buffers is enough to satisfy the code paths under test.
//   • If an individual test *does* need to exercise `sharp` you can opt out
//     locally by calling `vi.unmock('sharp')` *before* importing the module
//     under test.
//
// The stub below re‑creates the fluent API most code relies on (`resize()`,
// `jpeg()`, `toBuffer()`, …) but performs no work.
// ---------------------------------------------------------------------------

// Import from built package but avoid problematic barrel exports
import { DbProvider } from "@vrooli/server/db/provider.js";
import { initIdGenerator } from "@vrooli/shared";
import { execSync } from "child_process";
import { generateKeyPairSync } from "crypto";
import * as http from "http";
import * as https from "https";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import { vi, beforeAll, afterAll } from "vitest";

const SETUP_TIMEOUT_MS = 120_000;
const TEARDOWN_TIMEOUT_MS = 60_000;

vi.mock("sharp", () => {
    const makeChain = () => {
        const chain: any = {};
        const pass = () => chain; // keeps the chainable API intact

        Object.assign(chain, {
            resize: pass,
            rotate: pass,
            flatten: pass,
            jpeg: pass,
            png: pass,
            webp: pass,
            avif: pass,
            toBuffer: async () => Buffer.alloc(0),
            toFile: async () => ({ size: 0 }),
            metadata: async () => ({}),
        });

        return chain;
    };

    const mockedSharp = () => makeChain();

    // Support both `import sharp from 'sharp'` *and*
    // `import * as sharp from 'sharp'` styles.
    return { __esModule: true, default: mockedSharp };
});

// Mock CacheService to prevent initialization issues during module loading
vi.mock("@vrooli/server/redisConn.js", () => ({
    CacheService: {
        get: vi.fn(() => ({
            get: vi.fn(),
            set: vi.fn(),
            delete: vi.fn(),
            clear: vi.fn(),
        })),
        init: vi.fn(),
    },
}));

// Mock the entire conversation response engine to avoid circular dependencies
vi.mock("@vrooli/server/services/conversation/responseEngine.js", () => ({
    ResponseEngine: vi.fn().mockImplementation(() => ({
        generateResponse: vi.fn().mockResolvedValue("Mock response"),
    })),
}));

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
        execSync("cd ../server && pnpm prisma migrate deploy", { stdio: "inherit", shell: "/bin/bash" });
        console.info("Generating Prisma client...");
        execSync("cd ../server && pnpm prisma generate", { stdio: "inherit", shell: "/bin/bash" });
        console.info("Database setup complete.");
    } catch (error) {
        console.error("Failed to set up database:", error);
        throw error; // Fail the test setup if migrations fail
    }

    // Set snowflake worker ID
    initIdGenerator(parseInt(process.env.WORKER_ID ?? "0"));

    // Initialize only essential services to avoid circular dependencies
    // CacheService is mocked, so we skip its initialization
    
    // Setup databases - this is what tests need
    await DbProvider.init();
});

// Removed LLM service mocks to avoid circular dependencies

afterAll(async () => {

    // Restore all vitest mocks
    vi.restoreAllMocks();

    // Skip queue shutdown since we didn't initialize queues

    // Skip Redis connection close since we're using minimal setup

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

    // Final check for remaining connections
    console.info("\nFinal check for remaining connections...");
    const remainingConnections = checkActiveHandles();

    if (remainingConnections > 0) {
        console.warn(`\n⚠️ WARNING: ${remainingConnections} connections still active after cleanup`);
        console.error("\nexiting with code 1");

        setTimeout(() => {
            process.exit(1);
        }, 5000);
    } else {
        console.info("\n✅ All connections properly closed");

        setTimeout(() => {
            process.exit(0);
        }, 5000);
    }
});

/**
 * Custom function to check for hanging connections
 * This analyzes Node's internal handle tracking directly without whyIsNodeRunning
 * @returns The number of active handles
 */
function checkActiveHandles(): number {
    try {
        // Get the active handles directly from process
        // TypeScript doesn't know about this internal method, so we use type assertion
        const activeHandles = (process as any)._getActiveHandles();

        // Filter out expected handles (like TTY streams for stdout/stderr)
        const significantHandles = activeHandles.filter(handle => {
            if (!handle) return false;

            // TTY streams for console output are expected and not a problem
            if (handle.constructor && handle.constructor.name === "WriteStream" &&
                (handle.fd === 1 || handle.fd === 2)) {
                return false;
            }
            if (handle.constructor && handle.constructor.name === "ReadStream" && handle.fd === 0) {
                return false;
            }

            // Otherwise consider it significant
            return true;
        });

        // Group handles by type
        const handlesByType: Record<string, any[]> = {};

        for (const handle of activeHandles) {
            if (!handle) continue;

            let type = "unknown";

            if (handle.constructor && handle.constructor.name) {
                type = handle.constructor.name;
            }

            if (!handlesByType[type]) {
                handlesByType[type] = [];
            }
            handlesByType[type].push(handle);
        }

        // Print summary
        console.info("Active handles summary:");

        if (Object.keys(handlesByType).length === 0) {
            console.info("  No active handles detected");
        } else {
            // Print counts and details for each type
            Object.entries(handlesByType)
                .sort((a, b) => b[1].length - a[1].length) // Sort by count (descending)
                .forEach(([type, handles]) => {
                    console.info(`  - ${type}: ${handles.length}`);

                    // For each type, add extra info based on handle type
                    if (type === "Socket") {
                        handles.forEach((socket, i) => {
                            try {
                                // Basic socket info
                                const addrInfo = socket._address ?
                                    `${socket._address.address || ""}:${socket._address.port || ""}` :
                                    "unknown address";
                                const connInfo = socket.remoteAddress ?
                                    `${socket.remoteAddress}:${socket.remotePort}` :
                                    "no remote";
                                const serverInfo = socket.server ? "server socket" : "client socket";

                                // Check socket state 
                                const destroyed = socket.destroyed ? "destroyed" : "active";
                                const connecting = socket.connecting ? "connecting" : "connected";
                                const readable = socket.readable ? "readable" : "not readable";
                                const writable = socket.writable ? "writable" : "not writable";

                                // Check for pending operations
                                const pendingOps = socket._pendingData ? "has pending data" : "no pending data";

                                // Get more detailed information
                                let socketDetails = "";
                                if (socket._handle) {
                                    socketDetails = `(type: ${socket._handle.constructor?.name || "unknown"})`;
                                }

                                // Look for traces of what created this socket in stack
                                const creation = socket.creation || socket._creation || "";
                                const creationInfo = creation ? `\n      Created at: ${creation}` : "";

                                console.info(
                                    `    #${i + 1}: ${addrInfo} → ${connInfo} (${serverInfo}) ${socketDetails}\n` +
                                    `      State: ${destroyed}, ${connecting}, ${readable}, ${writable}\n` +
                                    `      Operations: ${pendingOps}${creationInfo}`,
                                );

                                // If it's a server socket, check for linked server details
                                if (socket.server) {
                                    const serverAddr = socket.server.address && socket.server.address();
                                    const serverAddrStr = serverAddr ?
                                        (typeof serverAddr === "string" ?
                                            serverAddr :
                                            `${serverAddr.address || ""}:${serverAddr.port || ""}`) :
                                        "not listening";
                                    console.info(`      Server: ${serverAddrStr}`);
                                }

                                // Check event listeners
                                const events = Object.keys(socket._events || {}).join(", ");
                                if (events) {
                                    console.info(`      Events: ${events}`);
                                }
                            } catch (err) {
                                console.info(`    #${i + 1}: Error getting socket details: ${err.message}`);
                            }
                        });
                    } else if (type === "Server") {
                        handles.forEach((server, i) => {
                            const address = server.address ?
                                (typeof server.address() === "string" ?
                                    server.address() :
                                    `${server.address()?.address || ""}:${server.address()?.port || ""}`) :
                                "not listening";
                            console.info(`    #${i + 1}: ${address}`);

                            // Check event listeners
                            const events = Object.keys(server._events || {}).join(", ");
                            if (events) {
                                console.info(`      Events: ${events}`);
                            }
                        });
                    } else if (type === "Timer" || type === "Timeout" || type === "Immediate") {
                        // Show info about timer (how long until it fires)
                        handles.forEach((timer, i) => {
                            if (i < 3) { // Only show first 3 timers to avoid clutter
                                const msFuture = timer._idleTimeout > 0 ?
                                    `fires in ${timer._idleTimeout}ms` :
                                    "repeating/unknown";
                                // Try to get callback info
                                let callbackInfo = "";
                                try {
                                    const callbackStr = timer._onTimeout ? timer._onTimeout.toString().substring(0, 100) : "";
                                    callbackInfo = callbackStr ? `\n      Callback: ${callbackStr}...` : "";
                                } catch (err) {
                                    // Ignore errors getting callback info
                                }

                                console.info(`    #${i + 1}: ${msFuture}${callbackInfo}`);
                            }
                        });
                        if (handles.length > 3) {
                            console.info(`    ... and ${handles.length - 3} more timers`);
                        }
                    } else if (type === "ReadStream") {
                        // Show info about read streams
                        handles.forEach((stream, i) => {
                            let details = "unknown stream";

                            // Try different ways to get path info
                            if (stream.path) {
                                details = `path: ${stream.path}`;
                            } else if (stream.getPath) {
                                details = `path: ${stream.getPath()}`;
                            } else if (stream.fd !== undefined) {
                                details = `fd: ${stream.fd}`;
                            }

                            // Check if stream is a TTY
                            if (stream.isTTY) {
                                details += " (TTY)";
                            }

                            // Show readability status
                            const readable = stream.readable ? "readable" : "not readable";

                            console.info(`    #${i + 1}: ${details} - ${readable}`);
                        });
                    } else if (type === "WriteStream") {
                        // Show info about write streams
                        handles.forEach((stream, i) => {
                            let details = "unknown stream";

                            // Try different ways to get path info
                            if (stream.path) {
                                details = `path: ${stream.path}`;
                            } else if (stream.getPath) {
                                details = `path: ${stream.getPath()}`;
                            } else if (stream.fd !== undefined) {
                                details = `fd: ${stream.fd}`;
                            }

                            // Check if stream is a TTY
                            if (stream.isTTY) {
                                details += " (TTY)";
                            }

                            // Show writability status
                            const writable = stream.writable ? "writable" : "not writable";

                            console.info(`    #${i + 1}: ${details} - ${writable}`);
                        });
                    } else {
                        handles.forEach((handle, i) => {
                            console.info(`    #${i + 1}: ${handle.constructor.name}`);
                        });
                    }
                });

            console.info(`Total active handles: ${activeHandles.length}`);
        }

        // Check for pending callbacks
        // TypeScript doesn't know about this internal method, so we use type assertion
        const pendingTimers = (process as any)._getActiveHandles ? (process as any)._getActiveHandles().length : "unknown";
        const pendingCallbacks = (process as any)._getAsyncIdCount ? (process as any)._getAsyncIdCount() : "unknown";

        console.info("\nEvent loop state:");
        console.info(`  - Active timers: ${pendingTimers}`);
        console.info(`  - Pending callbacks: ${pendingCallbacks}`);

        return significantHandles.length;
    } catch (error) {
        console.error("Error checking for hanging connections:", error);
        return 0;
    }
}
