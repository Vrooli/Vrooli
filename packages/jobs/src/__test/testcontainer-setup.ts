/**
 * Testcontainer setup for integration tests
 * Use this for tests that need real database access
 */

import { DbProvider, initSingletons } from "@vrooli/server";
import { initIdGenerator } from "@vrooli/shared";
import { execSync } from "child_process";
import { generateKeyPairSync } from "crypto";
import { vi } from "vitest";
import { GenericContainer, type StartedTestContainer } from "testcontainers";

let redisContainer: StartedTestContainer;
let postgresContainer: StartedTestContainer;

export async function setupTestContainers() {
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
    const postgresHost = postgresContainer.getHost();
    const postgresPort = postgresContainer.getMappedPort(5432);
    process.env.DB_URL = `postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${postgresHost}:${postgresPort}/${POSTGRES_DB}`;

    // Apply Prisma migrations and generate client
    try {
        console.info("Applying Prisma migrations...");
        execSync("pnpm prisma migrate deploy", { stdio: "inherit" });
        console.info("Database setup complete.");
    } catch (error) {
        console.error("Failed to set up database:", error);
        throw error;
    }

    // Set snowflake worker ID
    initIdGenerator(parseInt(process.env.WORKER_ID ?? "0"));

    // Initialize singletons
    await initSingletons();
    
    // Setup databases
    await DbProvider.init();
}

export async function teardownTestContainers() {
    // Close database connection
    await DbProvider.shutdown();

    // Stop containers
    try {
        if (redisContainer) {
            await redisContainer.stop({
                timeout: 10000,
                removeVolumes: true,
            });
        }
    } catch (error) {
        console.error("Error stopping Redis container:", error);
    }

    try {
        if (postgresContainer) {
            await postgresContainer.stop({
                timeout: 10000,
                removeVolumes: true,
            });
        }
    } catch (error) {
        console.error("Error stopping Postgres container:", error);
    }
}