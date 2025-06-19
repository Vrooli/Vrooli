/**
 * Helper for running tests in database transactions
 * This is MUCH faster than truncating tables
 */

import { type PrismaClient } from "@prisma/client";
import { DbProvider } from "../../db/provider.js";

export interface TransactionTestContext {
    prisma: PrismaClient;
}

/**
 * Wraps a test in a database transaction that gets rolled back
 * This provides test isolation without expensive TRUNCATE operations
 * 
 * IMPORTANT: This works by temporarily overriding DbProvider.get() to return
 * the transaction client, so all code using DbProvider.get() will automatically
 * use the transaction.
 * 
 * WARNING: This helper does NOT work with code that creates its own transactions!
 * Prisma does not support nested transactions, so if your test code internally
 * uses $transaction(), it will fail. In such cases, you must test without this
 * wrapper and handle cleanup manually.
 * 
 * Usage:
 * ```ts
 * it("should create a user", withDbTransaction(async () => {
 *     // All database operations through DbProvider.get() will use the transaction
 *     const user = await DbProvider.get().user.create({ data: { ... } });
 *     expect(user).toBeDefined();
 *     // Transaction automatically rolls back after test
 * }));
 * ```
 */
export function withDbTransaction<T>(
    testFn: () => Promise<T>
): () => Promise<T> {
    return async () => {
        // Make sure DbProvider is available
        if (!DbProvider || !DbProvider.get) {
            throw new Error("DbProvider not initialized. Make sure to call DbProvider.init() in beforeAll()");
        }
        
        // Store reference to original get method on the class itself
        const DbProviderClass = DbProvider.constructor;
        const originalGet = DbProviderClass.prototype.get || DbProvider.get;
        
        let client;
        try {
            client = DbProvider.get();
        } catch (error) {
            throw new Error(`Failed to get database client: ${error}`);
        }
        
        // Use interactive transactions with a long timeout
        return await client.$transaction(
            async (txClient) => {
                // Override DbProvider.get() to return the transaction client
                // This needs to work even if DbProvider is re-imported
                const override = () => txClient as any;
                DbProvider.get = override;
                if (DbProviderClass.prototype.get) {
                    DbProviderClass.prototype.get = override;
                }
                // Also try to override on the class itself
                if (DbProviderClass.get) {
                    DbProviderClass.get = override;
                }
                
                try {
                    // Run the test with the overridden DbProvider
                    const result = await testFn();
                    
                    // Throw a special error to trigger rollback
                    throw new RollbackError(result);
                } finally {
                    // Always restore the original get method
                    DbProvider.get = originalGet;
                    if (DbProviderClass.prototype.get) {
                        DbProviderClass.prototype.get = originalGet;
                    }
                    if (DbProviderClass.get) {
                        DbProviderClass.get = originalGet;
                    }
                }
            },
            {
                maxWait: 30000, // Max time to wait for transaction slot
                timeout: 60000, // Max time for transaction to complete
            }
        ).catch(error => {
            // If it's our rollback error, return the result
            if (error instanceof RollbackError) {
                return error.result;
            }
            // Otherwise, it's a real error
            throw error;
        });
    };
}

/**
 * Legacy wrapper that provides the transaction client directly
 * Use withDbTransaction instead for better integration with existing code
 */
export function withTransaction<T>(
    testFn: (ctx: TransactionTestContext) => Promise<T>
): () => Promise<T> {
    return async () => {
        const client = DbProvider.get();
        
        // Use interactive transactions with a long timeout
        return await client.$transaction(
            async (prisma) => {
                // Run the test with the transaction client
                const result = await testFn({ prisma });
                
                // Throw a special error to trigger rollback
                throw new RollbackError(result);
            },
            {
                maxWait: 30000, // Max time to wait for transaction slot
                timeout: 60000, // Max time for transaction to complete
            }
        ).catch(error => {
            // If it's our rollback error, return the result
            if (error instanceof RollbackError) {
                return error.result;
            }
            // Otherwise, it's a real error
            throw error;
        });
    };
}

/**
 * Special error class to trigger transaction rollback
 * while preserving the test result
 */
class RollbackError<T> extends Error {
    constructor(public result: T) {
        super("Transaction rollback");
        this.name = "RollbackError";
    }
}

/**
 * Alternative: Use savepoints for nested test isolation
 * This is useful when you need multiple isolated operations in one test
 */
export async function withSavepoint<T>(
    prisma: PrismaClient,
    name: string,
    fn: () => Promise<T>
): Promise<T> {
    await prisma.$executeRawUnsafe(`SAVEPOINT ${name}`);
    try {
        const result = await fn();
        await prisma.$executeRawUnsafe(`RELEASE SAVEPOINT ${name}`);
        return result;
    } catch (error) {
        await prisma.$executeRawUnsafe(`ROLLBACK TO SAVEPOINT ${name}`);
        throw error;
    }
}