import { GenericContainer, Wait } from 'testcontainers';
import dotenv from 'dotenv';
import path from 'path';

// Load environment variables
dotenv.config({ path: path.resolve(__dirname, '../../../../.env') });

let postgresContainer: any;
let redisContainer: any;

export async function setup() {
    console.log('🚀 Starting test containers...');
    
    // Start PostgreSQL container with pgvector extension
    postgresContainer = await new GenericContainer('pgvector/pgvector:pg16')
        .withEnvironment({
            POSTGRES_DB: 'test-db',
            POSTGRES_USER: 'test-user',
            POSTGRES_PASSWORD: 'test-password',
        })
        .withExposedPorts(5432)
        .withWaitStrategy(Wait.forLogMessage('database system is ready to accept connections', 2))
        .start();

    // Start Redis container
    redisContainer = await new GenericContainer('redis:7-alpine')
        .withExposedPorts(6379)
        .start();

    // Set environment variables for the test run
    const dbUrl = `postgresql://test-user:test-password@localhost:${postgresContainer.getMappedPort(5432)}/test-db`;
    process.env.DATABASE_URL = dbUrl;
    process.env.DB_URL = dbUrl; // Prisma schema uses DB_URL
    process.env.REDIS_URL = `redis://localhost:${redisContainer.getMappedPort(6379)}`;
    
    // Set required environment variables for server initialization
    process.env.JWT_PRIV = 'test-private-key';
    process.env.JWT_PUB = 'test-public-key';
    process.env.PROJECT_DIR = '/root/Vrooli';
    process.env.VITE_SERVER_LOCATION = 'http://localhost:5000';
    process.env.LETSENCRYPT_EMAIL = 'test@example.com';
    process.env.VAPID_PUBLIC_KEY = 'test-vapid-public';
    process.env.VAPID_PRIVATE_KEY = 'test-vapid-private';
    process.env.WORKER_ID = '0';
    process.env.NODE_ENV = 'test';
    
    // Store container references for teardown
    global.__POSTGRES_CONTAINER__ = postgresContainer;
    global.__REDIS_CONTAINER__ = redisContainer;

    console.log('✅ Test containers started successfully');
    console.log(`📊 PostgreSQL: ${dbUrl}`);
    console.log(`📊 Redis: ${process.env.REDIS_URL}`);

    return async () => {
        console.log('🛑 Stopping test containers...');
        await postgresContainer?.stop();
        await redisContainer?.stop();
        console.log('✅ Test containers stopped');
    };
}

export async function teardown() {
    // Cleanup is handled by the function returned from setup
}