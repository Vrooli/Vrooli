import { MINUTES_5_MS, SEEDED_PUBLIC_IDS } from "@vrooli/shared";
import pkg from "@prisma/client";
import { logger } from "../events/logger.js";

const { PrismaClient } = pkg;

const DEFAULT_DB_TYPE = "postgres";
const DB_TYPE = process.env.DB_TYPE || DEFAULT_DB_TYPE;

// Seeding retry configuration
const SEED_RETRY_MS = MINUTES_5_MS;
const SEED_MAX_RETRIES = 10;

export interface DatabaseService {
    /** Connects to the database */
    connect(): Promise<void>;
    /** Disconnects from the database */
    disconnect(): Promise<void>;
    /** Returns the database client */
    getClient(): InstanceType<typeof PrismaClient>;
    /** 
     * Runs the initialization script for the database,
     * including creating tables and seeding data.
     */
    seed(): Promise<boolean>;
    /**
     * Drops the entire database.
     * Only available in test mode.
     */
    deleteAll(): Promise<void>;
}

/**
 * Provides a singleton interface for the database service.
 * 
 * Allows for asynchronous initialization of the database service
 * and synchronous access to the database client.
 * 
 * Also allows for the database service to be swapped out easily, 
 * such as using SQLite when running locally and PostgreSQL when 
 * hosted.
 */
export class DbProvider {
    private static dbService: DatabaseService | null = null;
    private static initPromise: Promise<void> | null = null;
    private static isInitialized = false;
    private static seedingSuccessful = false;
    private static seedRetryCount = 0;
    private static seedAttemptCount = 0; // Track total attempts separately from retry count
    private static seedRetryTimeout: NodeJS.Timeout | null = null;
    private static connected = false;
    private static adminId: string | null = null;

    /**
     * Asynchronously initializes the chosen database service.
     */
    public static async init() {
        // If already initializing, wait for the existing promise
        if (DbProvider.initPromise) {
            return DbProvider.initPromise;
        }

        // If already initialized, return immediately
        if (DbProvider.dbService && DbProvider.isInitialized) {
            return;
        }

        // Create the initialization promise
        DbProvider.initPromise = (async () => {
            try {
                // Load the chosen database service
                switch (DB_TYPE) {
                    case "postgres": {
                        const { PostgresDriver } = await import("./postgres.js");
                        DbProvider.dbService = new PostgresDriver();
                        break;
                    }
                    case "sqlite": {
                        const { SQLiteDriver } = await import("./sqlite.js");
                        DbProvider.dbService = new SQLiteDriver();
                        break;
                    }
                    // Add databases as needed
                    default: {
                        throw new Error(`Unsupported database type: ${DB_TYPE}`);
                    }
                }

                if (!DbProvider.dbService) {
                    throw new Error("Database service failed to initialize");
                }

                // Make sure the database client is available
                try {
                    await DbProvider.dbService.connect();
                    DbProvider.connected = true;
                    logger.info("Database connected successfully");

                    // Try to seed the database
                    await DbProvider.attemptSeeding();
                } catch (connectionError) {
                    logger.error("Database connection failed", { trace: "0014", error: connectionError });
                }

                // Mark as initialized only after successful completion
                DbProvider.isInitialized = true;

            } catch (error) {
                // Reset on error so it can be retried
                DbProvider.initPromise = null;
                DbProvider.isInitialized = false;
                logger.error("Caught error in database provider initialization", { trace: "0011", error });
                throw error;
            }
        })();

        return DbProvider.initPromise;
    }

    /**
     * Attempts to seed the database and schedules retries if it fails
     */
    private static async attemptSeeding(): Promise<void> {
        if (!DbProvider.connected || !DbProvider.dbService) {
            logger.error("Cannot seed database: not connected", { trace: "0015" });
            return;
        }

        // Increment attempt counter
        DbProvider.seedAttemptCount++;

        logger.info(`Attempting database seeding (attempt ${DbProvider.seedAttemptCount})`);

        try {
            DbProvider.seedingSuccessful = await DbProvider.dbService.seed();

            if (DbProvider.seedingSuccessful) {
                // If seeding was successful, clear any pending retry
                if (DbProvider.seedRetryTimeout) {
                    clearTimeout(DbProvider.seedRetryTimeout);
                    DbProvider.seedRetryTimeout = null;
                }
                DbProvider.seedRetryCount = 0;
                logger.info(`Database seeding completed successfully after ${DbProvider.seedAttemptCount} attempt(s)`);
            } else {
                logger.warning(`Seeding attempt ${DbProvider.seedAttemptCount} returned false`);
                DbProvider.scheduleRetry("Seeding returned false");
            }
        } catch (seedError) {
            DbProvider.seedingSuccessful = false;
            logger.error(`Seeding attempt ${DbProvider.seedAttemptCount} failed`, {
                trace: "DB-SEED-ERROR",
                attempt: DbProvider.seedAttemptCount,
                name: seedError instanceof Error ? seedError.name : undefined,
                message: seedError instanceof Error ? seedError.message : String(seedError),
                stack: seedError instanceof Error && seedError.stack ? seedError.stack : undefined,
            });
            DbProvider.scheduleRetry(seedError);
        }
    }

    /**
     * Schedules a retry for database seeding
     */
    private static scheduleRetry(error: unknown): void {
        // Check if we've reached maximum retries before incrementing
        if (DbProvider.seedRetryCount >= SEED_MAX_RETRIES) {
            logger.error("Database seeding failed after maximum retry attempts", {
                trace: "DB-SEED-MAX-RETRIES",
                maxRetries: SEED_MAX_RETRIES,
                totalAttempts: DbProvider.seedAttemptCount,
                name: error instanceof Error ? error.name : undefined,
                message: error instanceof Error ? error.message : String(error),
                stack: error instanceof Error && error.stack ? error.stack : undefined,
            });
            return;
        }

        // Increment retry counter
        DbProvider.seedRetryCount++;

        const remainingRetries = SEED_MAX_RETRIES - DbProvider.seedRetryCount;

        logger.warning(`Database seeding failed, scheduling retry ${DbProvider.seedRetryCount} of ${SEED_MAX_RETRIES} in ${SEED_RETRY_MS}ms`, {
            trace: "DB-SEED-RETRY",
            attempt: DbProvider.seedAttemptCount,
            retryCount: DbProvider.seedRetryCount,
            remainingRetries,
            maxRetries: SEED_MAX_RETRIES,
            name: error instanceof Error ? error.name : undefined,
            message: error instanceof Error ? error.message : String(error),
            stack: error instanceof Error && error.stack ? error.stack : undefined,
        });

        // Clear any existing timeout
        if (DbProvider.seedRetryTimeout) {
            clearTimeout(DbProvider.seedRetryTimeout);
        }

        // Schedule the retry
        DbProvider.seedRetryTimeout = setTimeout(() => {
            logger.info(`Executing database seeding retry ${DbProvider.seedRetryCount} of ${SEED_MAX_RETRIES}`);
            DbProvider.attemptSeeding();
        }, SEED_RETRY_MS);
    }

    /**
     * Synchronously returns the database client.
     * Assumes the database service has been initialized.
     */
    public static get(): InstanceType<typeof PrismaClient> {
        if (!DbProvider.dbService) {
            throw new Error("Database service not initialized. Call initDatabaseService first.");
        }
        return DbProvider.dbService.getClient();
    }

    /**
     * Returns whether database seeding was successful.
     */
    public static isSeedingSuccessful(): boolean {
        return DbProvider.seedingSuccessful;
    }

    /**
     * Returns whether the database is connected.
     */
    public static isConnected(): boolean {
        return DbProvider.connected;
    }

    /**
     * Returns the current retry count for seeding.
     */
    public static getSeedRetryCount(): number {
        return DbProvider.seedRetryCount;
    }

    /**
     * Returns the total number of seeding attempts.
     */
    public static getSeedAttemptCount(): number {
        return DbProvider.seedAttemptCount;
    }

    /**
     * Returns the maximum configured retry count.
     */
    public static getMaxRetries(): number {
        return SEED_MAX_RETRIES;
    }

    /**
     * Returns whether seeding retries are still in progress.
     */
    public static isRetryingSeeding(): boolean {
        // Check if we have an active retry timeout OR if we're in the process of retrying
        // but the timeout hasn't been set yet
        return !!DbProvider.seedRetryTimeout ||
            (DbProvider.seedRetryCount > 0 &&
                DbProvider.seedRetryCount < SEED_MAX_RETRIES &&
                !DbProvider.seedingSuccessful);
    }

    /**
     * Manually force a retry of the seeding process.
     * This can be used when the underlying issue has been fixed and an admin
     * wants to restart the seeding process even if max retries were reached.
     * 
     * @param resetCounters If true, resets retry counters
     * @returns Boolean indicating if retry was scheduled
     */
    public static forceRetrySeeding(resetCounters = false): boolean {
        if (!DbProvider.connected || !DbProvider.dbService) {
            logger.error("Cannot force retry seeding: not connected", { trace: "0016" });
            return false;
        }

        if (resetCounters) {
            // Reset counters but keep attempt count to track total attempts
            DbProvider.seedRetryCount = 0;
        }

        // Clear any existing timeout
        if (DbProvider.seedRetryTimeout) {
            clearTimeout(DbProvider.seedRetryTimeout);
            DbProvider.seedRetryTimeout = null;
        }

        // Schedule immediate retry
        logger.info("Manually forcing database seeding retry");

        // Use process.nextTick to ensure it runs after current execution completes
        process.nextTick(() => {
            DbProvider.attemptSeeding();
        });

        return true;
    }

    /**
     * Gracefully shuts down the database connection
     * This should be called during application or test shutdown
     */
    public static async shutdown(): Promise<void> {
        try {
            // Clear any scheduled seeding retry
            if (DbProvider.seedRetryTimeout) {
                clearTimeout(DbProvider.seedRetryTimeout);
                DbProvider.seedRetryTimeout = null;
            }

            // Disconnect from the database if connected
            if (DbProvider.connected && DbProvider.dbService) {
                logger.info("Disconnecting from database");
                await DbProvider.dbService.disconnect();
                DbProvider.connected = false;
            }
        } catch (error) {
            logger.error("Error during database shutdown", { trace: "0720", error });
        }
    }

    /**
     * Drops the entire database.
     * Only available in test mode.
     */
    public static async deleteAll(): Promise<void> {
        if (process.env.NODE_ENV !== "test") {
            throw new Error("dropDatabase is only available in test mode");
        }
        if (!DbProvider.dbService) {
            throw new Error("Database service not initialized. Call init first.");
        }
        await DbProvider.dbService.deleteAll();
        logger.info("Database dropped for test environment");
    }

    /**
     * @returns the ID of the seeded admin user
     */
    public static async getAdminId(): Promise<string> {
        if (DbProvider.adminId) {
            return DbProvider.adminId;
        }
        const client = DbProvider.get();
        const admin = await client.user.findFirst({
            where: { publicId: SEEDED_PUBLIC_IDS.Admin },
        });
        if (!admin) {
            throw new Error("Admin user not found");
        }
        DbProvider.adminId = admin.id.toString();
        return DbProvider.adminId;
    }

    /**
     * Reset singleton instance (for testing)
     * This ensures each test gets a fresh connection
     */
    public static async reset(): Promise<void> {
        if (DbProvider.dbService) {
            try {
                await DbProvider.dbService.disconnect();
            } catch (error) {
                // Ignore disconnect errors
            }
        }
        
        if (DbProvider.seedRetryTimeout) {
            clearTimeout(DbProvider.seedRetryTimeout);
        }
        
        DbProvider.dbService = null;
        DbProvider.initPromise = null;
        DbProvider.isInitialized = false;
        DbProvider.seedingSuccessful = false;
        DbProvider.seedRetryCount = 0;
        DbProvider.seedAttemptCount = 0;
        DbProvider.seedRetryTimeout = null;
        DbProvider.connected = false;
        DbProvider.adminId = null;
    }
}
