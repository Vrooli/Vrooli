import pkg from "@prisma/client";
import { logger } from "../events/logger.js";

const { PrismaClient } = pkg;

const DEFAULT_DB_TYPE = "postgres";
const DB_TYPE = process.env.DB_TYPE || DEFAULT_DB_TYPE;

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
    seed(): Promise<void>;
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

    /**
     * Asynchronously initializes the chosen database service.
     */
    public static async init() {
        // If already initialized, just return
        if (DbProvider.dbService) {
            return;
        }

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

            // Make sure the database client is available and seeded
            await DbProvider.dbService.connect();
            await DbProvider.dbService.seed();
        } catch (error) {
            logger.error("Caught error in setupDatabase", { trace: "0011", error });
            // Don't let the app start if the database setup fails
            if (process.env.NODE_ENV !== "test") {
                process.exit(1);
            }
        }
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
}
