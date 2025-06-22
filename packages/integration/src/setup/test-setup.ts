import { beforeAll, beforeEach, afterEach, afterAll } from 'vitest';
import { PrismaClient } from '@prisma/client';
import { exec } from 'child_process';
import { promisify } from 'util';
import { DbProvider, CacheService } from '@vrooli/server';
import { initIdGenerator } from '@vrooli/shared';

const execAsync = promisify(exec);

// Create a single Prisma client for all tests
let prisma: PrismaClient;

beforeAll(async () => {
    // Initialize ID generator (required for Snowflake IDs)
    initIdGenerator(0);

    // Initialize Prisma client
    prisma = new PrismaClient({
        datasources: {
            db: {
                url: process.env.DB_URL || process.env.DATABASE_URL,
            },
        },
    });

    // Initialize the server's DbProvider
    await DbProvider.init();

    // Initialize CacheService (required by many endpoints)
    try {
        CacheService.get();
    } catch (e) {
        // CacheService initialization might fail in test environment, that's okay
        console.log('⚠️ CacheService initialization skipped in test environment');
    }

    // Run migrations
    console.log('🔄 Running database migrations...');
    await execAsync('cd ../server && pnpm prisma migrate deploy');
    console.log('✅ Migrations completed');

    // Export for use in tests
    (global as any).__PRISMA__ = prisma;
});

beforeEach(async () => {
    // Clean database before each test
    // This ensures test isolation
    await cleanDatabase();
});

afterEach(async () => {
    // Additional cleanup if needed
});

afterAll(async () => {
    // Disconnect Prisma
    await prisma?.$disconnect();
});

async function cleanDatabase() {
    // Get all table names except migrations
    const tables = await prisma.$queryRaw<Array<{ tablename: string }>>`
        SELECT tablename FROM pg_tables 
        WHERE schemaname = 'public' 
        AND tablename != '_prisma_migrations'
    `;

    // Truncate all tables
    for (const { tablename } of tables) {
        await prisma.$executeRawUnsafe(
            `TRUNCATE TABLE "public"."${tablename}" CASCADE`
        );
    }
}

// Export utilities for tests
export function getPrisma(): PrismaClient {
    return (global as any).__PRISMA__ || prisma;
}

export { cleanDatabase };