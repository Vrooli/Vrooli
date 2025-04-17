import pkg from "@prisma/client";
import { logger } from "../events/logger.js";
import { type DatabaseService } from "./provider.js";

const { PrismaClient } = pkg;

const debug = process.env.NODE_ENV === "development";

export class PostgresDriver implements DatabaseService {
    private prisma: InstanceType<typeof PrismaClient>;

    constructor() {
        // Prisma automatically picks up the DB_URL from your .env
        // e.g., DB_URL="postgresql://user:password@localhost:5432/mydb"
        // If you need custom configuration, pass it into the PrismaClient constructor
        this.prisma = new PrismaClient({
            log: debug ? ["info", "warn", "error"] : undefined,
        });
    }

    async connect(): Promise<void> {
        // Prisma doesn't require an explicit 'connect' to start using it.
        // But you can force a connection to check if the DB is reachable.
        await this.prisma.$connect();
        logger.info("Connected to PostgreSQL via Prisma");
    }

    async disconnect(): Promise<void> {
        await this.prisma.$disconnect();
        logger.info("Disconnected from PostgreSQL");
    }

    // If you'd like a standard way to access the Prisma client from outside:
    public getClient(): InstanceType<typeof PrismaClient> {
        return this.prisma;
    }

    public async seed(): Promise<boolean> {
        try {
            // For now, skip during testing
            if (process.env.NODE_ENV === "test") {
                console.info("skipping seed in test environment");
                return true;
            }
            const { init } = await import("./seeds/init.js");
            await init(this.prisma);
            return true;
        } catch (error) {
            logger.error("PostgreSQL seeding failed", { error });
            return false;
        }
    }
}
