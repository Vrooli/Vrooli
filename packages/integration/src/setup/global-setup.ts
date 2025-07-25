import { GenericContainer, Wait } from "testcontainers";
import { generateKeyPairSync } from "crypto";
import { execSync } from "child_process";
import dotenv from "dotenv";
import path from "path";

// Load environment variables
dotenv.config({ path: path.resolve(__dirname, "../../../../.env") });

let postgresContainer: any;
let redisContainer: any;

export async function setup() {
    console.log("\n=== INTEGRATION GLOBAL TEST SETUP STARTING ===");
    console.log("This runs ONCE before all test files");
    
    try {
        // Generate proper RSA keys for JWT testing
        const { privateKey, publicKey } = generateKeyPairSync("rsa", {
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

        // Set required environment variables
        process.env.NODE_ENV = "test";
        process.env.JWT_PRIV = privateKey;
        process.env.JWT_PUB = publicKey;
        process.env.PROJECT_DIR = "/home/matthalloran8/Vrooli";
        process.env.VITE_SERVER_LOCATION = "local";
        process.env.SITE_EMAIL_USERNAME = "test@example.com";
        process.env.SITE_EMAIL_ALIAS = "noreply@example.com";
        process.env.VAPID_PUBLIC_KEY = "test-vapid-public";
        process.env.VAPID_PRIVATE_KEY = "test-vapid-private";
        process.env.WORKER_ID = "0";
        process.env.ANTHROPIC_API_KEY = "dummy";
        process.env.MISTRAL_API_KEY = "dummy";
        process.env.OPENAI_API_KEY = "dummy";
        
        // Start PostgreSQL container with pgvector extension
        console.log("\n1. Starting PostgreSQL container...");
        const POSTGRES_USER = "testuser";
        const POSTGRES_PASSWORD = "testpassword";
        const POSTGRES_DB = "testdb";
        postgresContainer = await new GenericContainer("pgvector/pgvector:pg16")
            .withExposedPorts(5432)
            .withEnvironment({
                POSTGRES_USER,
                POSTGRES_PASSWORD,
                POSTGRES_DB,
            })
            .withWaitStrategy(Wait.forLogMessage("database system is ready to accept connections", 2))
            .withStartupTimeout(120000)
            .start();

        const postgresPort = postgresContainer.getMappedPort(5432);
        const dbUrl = `postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${postgresContainer.getHost()}:${postgresPort}/${POSTGRES_DB}`;
        process.env.DATABASE_URL = dbUrl;
        process.env.DB_URL = dbUrl; // Prisma schema uses DB_URL
        console.log(`✓ PostgreSQL started at ${dbUrl} (container:5432 -> host:${postgresPort})`);

        // Start Redis container
        console.log("\n2. Starting Redis container...");
        redisContainer = await new GenericContainer("redis:7-alpine")
            .withExposedPorts(6379)
            .withStartupTimeout(120000)
            .start();

        const redisPort = redisContainer.getMappedPort(6379);
        const redisUrl = `redis://${redisContainer.getHost()}:${redisPort}`;
        process.env.REDIS_URL = redisUrl;
        console.log(`✓ Redis started at ${redisUrl} (container:6379 -> host:${redisPort})`);
        
        // Run migrations ONCE (from server directory)
        console.log("\n3. Running Prisma migrations...");
        const serverPath = path.resolve(__dirname, "../../../server");
        // Set PRISMA_SCHEMA_PATH for the migration
        const envWithSchema = {
            ...process.env,
            PRISMA_SCHEMA_PATH: path.join(serverPath, "src/db/schema.prisma"),
        };
        execSync("npx prisma migrate deploy --schema=./src/db/schema.prisma", {
            stdio: "inherit",
            env: envWithSchema,
            cwd: serverPath,
        });
        console.log("✓ Migrations applied");

        // Generate Prisma client ONCE (from server directory)
        console.log("\n4. Generating Prisma client...");
        execSync("npx prisma generate --schema=./src/db/schema.prisma", {
            stdio: "inherit",
            env: envWithSchema,
            cwd: serverPath,
        });
        console.log("✓ Prisma client generated");
        
        // Store container references for teardown
        global.__POSTGRES_CONTAINER__ = postgresContainer;
        global.__REDIS_CONTAINER__ = redisContainer;

        console.log("\n=== INTEGRATION GLOBAL TEST SETUP COMPLETE ===");
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
    console.log("\n=== INTEGRATION GLOBAL TEST TEARDOWN STARTING ===");

    // Retrieve containers from global
    const redis = global.__REDIS_CONTAINER__;
    const postgres = global.__POSTGRES_CONTAINER__;

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

    console.log("=== INTEGRATION GLOBAL TEST TEARDOWN COMPLETE ===\n");
}
