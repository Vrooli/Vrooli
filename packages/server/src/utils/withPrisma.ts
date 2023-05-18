import pkg from "@prisma/client";
import { logger } from "../events";
import { PrismaType } from "../types";

const { PrismaClient } = pkg;


interface WithPrismaProps {
    process: (prisma: PrismaType) => Promise<void>,
    trace: string,
    traceObject?: Record<string, any>,
}

/**
 * Handles the Prisma connection/disconnection and error logging
 */
export async function withPrisma({
    process,
    trace,
    traceObject,
}: WithPrismaProps): Promise<boolean> {
    let success = false;
    const prisma = new PrismaClient();
    try {
        await process(prisma);
        success = true;
    } catch (error) {
        logger.error("Caught error in withPrisma", { trace, error, ...traceObject });
    } finally {
        await prisma.$disconnect();
    }
    return success;
}