import pkg from "@prisma/client";
import { logger } from "../events/logger.js";
import { type DatabaseService } from "./provider.js";

const { PrismaClient } = pkg;

export class SQLiteDriver implements DatabaseService {
    private prisma: InstanceType<typeof PrismaClient>;

    constructor() {
        // Prisma automatically picks up the DATABASE_URL from your .env
        // e.g., DATABASE_URL="postgresql://user:password@localhost:5432/mydb"
        // If you need custom configuration, pass it into the PrismaClient constructor
        this.prisma = new PrismaClient(
            //TODO probably need to specify the file path here
        );
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

    public async seed(): Promise<void> {
        // Could use a different seed script here if the default data is different when running SQLite (i.e. local)
        const { init } = await import("./seeds/init.js");
        await init(this.prisma);
    }
}
