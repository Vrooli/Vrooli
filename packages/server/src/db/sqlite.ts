import pkg from "@prisma/client";
import { logger } from "../events/logger.js";
import { type DatabaseService } from "./provider.js";

const { PrismaClient } = pkg;

const debug = process.env.NODE_ENV === "development";

export class SQLiteDriver implements DatabaseService {
    private prisma: InstanceType<typeof PrismaClient>;

    constructor() {
        // Prisma automatically picks up the DB_URL from your .env
        // e.g., DB_URL="postgresql://user:password@localhost:5432/mydb"
        // If you need custom configuration, pass it into the PrismaClient constructor
        this.prisma = new PrismaClient({
            //TODO probably need to specify the file path here
            log: debug ? ["info", "warn", "error"] : undefined,
        });
    }

    async connect(): Promise<void> {
        // Prisma doesn't require an explicit 'connect' to start using it.
        // But you can force a connection to check if the DB is reachable.
        await this.prisma.$connect();
        logger.info("Connected to SQLite via Prisma");
    }

    async disconnect(): Promise<void> {
        await this.prisma.$disconnect();
        logger.info("Disconnected from SQLite");
    }

    // If you'd like a standard way to access the Prisma client from outside:
    public getClient(): InstanceType<typeof PrismaClient> {
        return this.prisma;
    }

    public async seed(): Promise<boolean> {
        try {
            logger.info("Starting SQLite seeding");
            // Could use a different seed script here if the default data is different when running SQLite (i.e. local)
            const { init } = await import("./seeds/init.js");
            await init(this.prisma);
            logger.info("SQLite seeding completed successfully");
            return true;
        } catch (error) {
            // Check if this is a "table does not exist" error
            const errorMessage = error instanceof Error ? error.message : String(error);
            const isTableMissing = errorMessage.includes("no such table") || 
                                   errorMessage.includes("does not exist") ||
                                   errorMessage.includes("doesn't exist") ||
                                   errorMessage.includes("P2021"); // Prisma error code for missing table
            
            if (isTableMissing) {
                logger.error("SQLite seeding failed: Database tables not found. Please run migrations first.", {
                    trace: "SQLITE-SEED-NO-TABLES",
                    message: "Database migrations have not been applied. Run 'pnpm prisma migrate deploy' before starting the server.",
                    originalError: errorMessage,
                });
            } else {
                logger.error("SQLite seeding failed", {
                    trace: "SQLITE-SEED",
                    name: error instanceof Error ? error.name : undefined,
                    message: error instanceof Error ? error.message : String(error),
                    stack: error instanceof Error && error.stack ? error.stack : undefined,
                });
            }
            return false;
        }
    }

    /**
     * Clears all rows from every table in the database while preserving table definitions.
     */
    public async deleteAll(): Promise<void> {
        // Disable foreign key constraints to allow deletion in any order
        await this.prisma.$executeRawUnsafe("PRAGMA foreign_keys = OFF;");
        // Retrieve all user-defined tables in SQLite
        const tables: Array<{ name: string }> = await this.prisma.$queryRawUnsafe(
            "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';",
        );
        // Delete all rows from each table
        for (const { name } of tables) {
            await this.prisma.$executeRawUnsafe(`DELETE FROM "${name}";`);
        }
        // Reset SQLite autoincrement sequences
        await this.prisma.$executeRawUnsafe("DELETE FROM sqlite_sequence;");
        // Re-enable foreign key constraints
        await this.prisma.$executeRawUnsafe("PRAGMA foreign_keys = ON;");
        logger.info("Cleared all data from SQLite database tables");
    }
}
