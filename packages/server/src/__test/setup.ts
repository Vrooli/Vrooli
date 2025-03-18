import { i18nConfig } from "@local/shared";
import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { execSync } from "child_process";
import { generateKeyPairSync } from "crypto";
import i18next from "i18next";
import { GenericContainer, StartedTestContainer } from "testcontainers";
import { DbProvider } from "../db/provider.js";
import { ModelMap } from "../models/base/index.js";
import { LlmServiceRegistry } from "../tasks/llm/registry.js";
import { TokenEstimationRegistry } from "../tasks/llm/tokenEstimator.js";

// Enable chai-as-promised for async tests
chai.use(chaiAsPromised);

let redisContainer: StartedTestContainer;
let postgresContainer: StartedTestContainer;

before(async function beforeAllTests() {
    this.timeout(60_000); // Allow extra time for container startup

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
    postgresContainer = await new GenericContainer("ankane/pgvector:v0.4.4")
        .withExposedPorts(5432)
        .withEnvironment({
            "POSTGRES_USER": POSTGRES_USER,
            "POSTGRES_PASSWORD": POSTGRES_PASSWORD,
            "POSTGRES_DB": POSTGRES_DB
        })
        .start();
    // Set the POSTGRES_URL environment variable
    const postgresHost = postgresContainer.getHost();
    const postgresPort = postgresContainer.getMappedPort(5432);
    process.env.DB_URL = `postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${postgresHost}:${postgresPort}/${POSTGRES_DB}`;

    // Apply Prisma migrations and generate client
    try {
        console.info('Applying Prisma migrations...');
        execSync('yarn prisma migrate deploy', { stdio: 'inherit' });
        console.info('Generating Prisma client...');
        execSync('yarn prisma generate', { stdio: 'inherit' });
        console.info('Database setup complete.');
    } catch (error) {
        console.error('Failed to set up database:', error);
        throw error; // Fail the test setup if migrations fail
    }

    // Initialize singletons
    await i18next.init(i18nConfig(true));
    await ModelMap.init();
    await LlmServiceRegistry.init();
    await TokenEstimationRegistry.init();
    await DbProvider.init();
});

after(async function afterAllTests() {
    this.timeout(60_000); // Allow extra time for container shutdown

    // Stop the Redis container
    if (redisContainer) {
        await redisContainer.stop();
    }

    // Stop the Postgres container
    if (postgresContainer) {
        await postgresContainer.stop();
    }
});
