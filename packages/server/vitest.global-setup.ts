/**
 * Global setup for vitest - runs ONCE before all test files
 * This handles expensive operations like starting containers and running migrations
 */

import { execSync } from "child_process";
import { GenericContainer, type StartedTestContainer } from "testcontainers";

let redisContainer: StartedTestContainer | null = null;
let postgresContainer: StartedTestContainer | null = null;

export async function setup() {
    console.log("\n=== GLOBAL TEST SETUP STARTING ===");
    console.log("This runs ONCE before all test files");
    
    try {
        // Set required environment variables
        process.env.JWT_PRIV = "dummy-key";
        process.env.JWT_PUB = "dummy-key";
        process.env.VITE_SERVER_LOCATION = "local";
        process.env.ANTHROPIC_API_KEY = "dummy";
        process.env.MISTRAL_API_KEY = "dummy";
        process.env.OPENAI_API_KEY = "dummy";
        
        // Start Redis container
        console.log("\n1. Starting Redis container...");
        redisContainer = await new GenericContainer("redis:alpine")
            .withExposedPorts(6379)
            .withStartupTimeout(120000)
            .start();
        const redisUrl = `redis://${redisContainer.getHost()}:${redisContainer.getMappedPort(6379)}`;
        process.env.REDIS_URL = redisUrl;
        console.log(`✓ Redis started at ${redisUrl}`);
        
        // Start PostgreSQL container
        console.log("\n2. Starting PostgreSQL container...");
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
            .withStartupTimeout(120000)
            .start();
        const dbUrl = `postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${postgresContainer.getHost()}:${postgresContainer.getMappedPort(5432)}/${POSTGRES_DB}`;
        process.env.DB_URL = dbUrl;
        console.log(`✓ PostgreSQL started at ${dbUrl}`);
        
        // Run migrations ONCE
        console.log("\n3. Running Prisma migrations...");
        execSync("npx prisma migrate deploy", { 
            stdio: "inherit",
            env: process.env,
            cwd: __dirname, // Ensure we run from the server directory
        });
        console.log("✓ Migrations applied");
        
        // Generate Prisma client ONCE
        console.log("\n4. Generating Prisma client...");
        execSync("npx prisma generate", { 
            stdio: "inherit",
            env: process.env,
            cwd: __dirname, // Ensure we run from the server directory
        });
        console.log("✓ Prisma client generated");
        
        // Store container references globally so teardown can access them
        (global as any).__TEST_REDIS_CONTAINER__ = redisContainer;
        (global as any).__TEST_POSTGRES_CONTAINER__ = postgresContainer;
        
        console.log("\n=== GLOBAL TEST SETUP COMPLETE ===");
        console.log(`Redis URL: ${redisUrl}`);
        console.log(`Database URL: ${dbUrl}`);
        console.log("Containers and migrations are ready for all tests\n");
        
    } catch (error) {
        console.error("\n=== GLOBAL SETUP FAILED ===");
        console.error(error);
        // Clean up on failure
        if (redisContainer) await redisContainer.stop();
        if (postgresContainer) await postgresContainer.stop();
        throw error;
    }
}

export async function teardown() {
    console.log("\n=== GLOBAL TEST TEARDOWN STARTING ===");
    
    // Retrieve containers from global
    const redis = (global as any).__TEST_REDIS_CONTAINER__;
    const postgres = (global as any).__TEST_POSTGRES_CONTAINER__;
    
    const stops = [];
    
    if (redis) {
        stops.push(redis.stop({ timeout: 10000 })
            .then(() => console.log("✓ Redis container stopped"))
            .catch((e: any) => console.error("Redis stop error:", e)));
    }
    
    if (postgres) {
        stops.push(postgres.stop({ timeout: 10000 })
            .then(() => console.log("✓ PostgreSQL container stopped"))
            .catch((e: any) => console.error("PostgreSQL stop error:", e)));
    }
    
    await Promise.all(stops);
    
    console.log("=== GLOBAL TEST TEARDOWN COMPLETE ===\n");
}