/**
 * Test validation utilities for detecting incomplete cleanup and data leakage
 * 
 * These utilities help ensure tests properly clean up after themselves
 * and don't leave orphaned data that could affect other tests.
 */

import { type PrismaClient } from "@prisma/client";

/**
 * Information about orphaned records found in a table
 */
export interface OrphanedRecords {
    table: string;
    count: number;
}

/**
 * Comprehensive cleanup validation options
 */
export interface CleanupValidationOptions {
    /** Tables to check for orphaned records. If not provided, checks critical tables */
    tables?: string[];
    /** Whether to throw an error if orphans are found (default: false) */
    throwOnOrphans?: boolean;
    /** Whether to log orphaned records to console (default: true in test env) */
    logOrphans?: boolean;
    /** Custom error message prefix */
    errorPrefix?: string;
}

/**
 * Tables that are critical for test isolation
 * These should typically be empty between tests
 */
export const CRITICAL_TABLES = [
    'user',
    'user_auth', 
    'session',
    'team',
    'member',
    'chat',
    'chat_message',
    'chat_participants',
    'resource',
    'resource_version',
    'routine',
    'routine_version',
    'run',
    'run_step',
    'run_io',
    'comment',
    'notification',
    'email',
    'credit_account',
    'api_key',
] as const;

/**
 * Tables that may legitimately have seed data or persistent records
 * These are checked but warnings are less critical
 */
export const PERSISTENT_TABLES = [
    'tag',
    'stats_site',
    'award', // May have system awards
] as const;

/**
 * Validates that specified tables are properly cleaned up
 * 
 * @param prisma - The Prisma client
 * @param options - Validation options
 * @returns Array of orphaned record information
 */
export async function validateCleanup(
    prisma: PrismaClient, 
    options: CleanupValidationOptions = {}
): Promise<OrphanedRecords[]> {
    // Only run validation in test environment
    if (process.env.NODE_ENV !== 'test') {
        return [];
    }
    
    const {
        tables = CRITICAL_TABLES,
        throwOnOrphans = false,
        logOrphans = true,
        errorPrefix = 'Test cleanup validation failed'
    } = options;
    
    // Check each table for remaining records
    const orphanChecks = await Promise.allSettled(
        tables.map(async (table): Promise<OrphanedRecords> => {
            try {
                const count = await (prisma as any)[table].count();
                return { table, count };
            } catch (error) {
                // Table might not exist or be accessible
                if (logOrphans) {
                    console.warn(`Could not check table '${table}':`, error instanceof Error ? error.message : error);
                }
                return { table, count: 0 };
            }
        })
    );
    
    // Extract successful results and orphaned records
    const results = orphanChecks
        .filter((result): result is PromiseFulfilledResult<OrphanedRecords> => result.status === 'fulfilled')
        .map(result => result.value);
        
    const orphans = results.filter(({ count }) => count > 0);
    
    // Log orphaned records if requested
    if (orphans.length > 0 && logOrphans) {
        console.warn(`${errorPrefix}:`, orphans.map(({ table, count }) => `${table}(${count})`).join(', '));
        
        // In debug mode, show more details
        if (process.env.TEST_LOG_LEVEL === 'DEBUG') {
            for (const { table, count } of orphans) {
                console.debug(`Table '${table}' has ${count} orphaned records`);
            }
        }
    }
    
    // Throw error if requested
    if (orphans.length > 0 && throwOnOrphans) {
        const message = `${errorPrefix}: Found orphaned records in ${orphans.length} tables: ${
            orphans.map(({ table, count }) => `${table}(${count})`).join(', ')
        }`;
        throw new Error(message);
    }
    
    return orphans;
}

/**
 * Validates cleanup for critical tables only
 * Useful for most tests that should have complete isolation
 */
export async function validateCriticalCleanup(prisma: PrismaClient): Promise<OrphanedRecords[]> {
    return validateCleanup(prisma, {
        tables: [...CRITICAL_TABLES],
        logOrphans: true,
        throwOnOrphans: false,
    });
}

/**
 * Strict cleanup validation that throws on any orphaned records
 * Use for tests that require perfect isolation
 */
export async function validateStrictCleanup(prisma: PrismaClient): Promise<void> {
    await validateCleanup(prisma, {
        tables: [...CRITICAL_TABLES],
        throwOnOrphans: true,
        errorPrefix: 'Strict cleanup validation failed',
    });
}

/**
 * Quick check for common tables that cause test interference
 * Returns true if cleanup is complete, false if orphans detected
 */
export async function isCleanupComplete(prisma: PrismaClient): Promise<boolean> {
    const orphans = await validateCleanup(prisma, {
        tables: ['user', 'session', 'chat', 'team', 'run'],
        logOrphans: false,
        throwOnOrphans: false,
    });
    
    return orphans.length === 0;
}

/**
 * Get summary of current database state for debugging
 * Useful for understanding what data exists before/after tests
 */
export async function getDatabaseSummary(prisma: PrismaClient): Promise<Record<string, number>> {
    const summary: Record<string, number> = {};
    
    const allTables = [...CRITICAL_TABLES, ...PERSISTENT_TABLES];
    
    await Promise.all(
        allTables.map(async (table) => {
            try {
                summary[table] = await (prisma as any)[table].count();
            } catch (error) {
                summary[table] = -1; // Indicates error
            }
        })
    );
    
    return summary;
}

/**
 * Assert that specific tables are empty (for use in tests)
 * Throws descriptive error if tables contain data
 */
export async function assertTablesEmpty(prisma: PrismaClient, tables: string[]): Promise<void> {
    const nonEmptyTables: OrphanedRecords[] = [];
    
    for (const table of tables) {
        try {
            const count = await (prisma as any)[table].count();
            if (count > 0) {
                nonEmptyTables.push({ table, count });
            }
        } catch (error) {
            throw new Error(`Failed to check table '${table}': ${error instanceof Error ? error.message : error}`);
        }
    }
    
    if (nonEmptyTables.length > 0) {
        const details = nonEmptyTables.map(({ table, count }) => `${table}: ${count} records`).join(', ');
        throw new Error(`Expected tables to be empty but found data in: ${details}`);
    }
}

/**
 * Wait for cleanup to complete (useful for eventual consistency scenarios)
 * Polls until tables are clean or timeout is reached
 */
export async function waitForCleanup(
    prisma: PrismaClient,
    options: {
        tables?: string[];
        timeoutMs?: number;
        pollIntervalMs?: number;
    } = {}
): Promise<boolean> {
    const {
        tables = ['user', 'session'],
        timeoutMs = 5000,
        pollIntervalMs = 100,
    } = options;
    
    const startTime = Date.now();
    
    while (Date.now() - startTime < timeoutMs) {
        const orphans = await validateCleanup(prisma, {
            tables,
            logOrphans: false,
            throwOnOrphans: false,
        });
        
        if (orphans.length === 0) {
            return true;
        }
        
        await new Promise(resolve => setTimeout(resolve, pollIntervalMs));
    }
    
    return false;
}