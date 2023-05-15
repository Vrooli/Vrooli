import pkg from "@prisma/client";
import { logger } from "../events";
import { PrismaType } from "../types";
import { batchCollect, BatchCollectProps, FindManyArgs } from "./batchCollect";

const { PrismaClient } = pkg;


interface BatchProps<T extends FindManyArgs> extends Omit<BatchCollectProps<T>, "prisma" | "processBatch"> {
    processBatch: (batch: any[], prisma: PrismaType) => Promise<void>,
    trace: string,
    traceObject?: Record<string, any>,
}

/**
 * Processes data from the database in batches, 
 * while also handling the connection/disconnection to the database
 */
export async function batch<T extends FindManyArgs>({
    processBatch,
    trace,
    traceObject,
    ...props
}: BatchProps<T>) {
    const prisma = new PrismaClient();
    try {
        await batchCollect<T>({
            prisma,
            processBatch: async (batch) => await processBatch(batch, prisma),
            ...props,
        });
    } catch (error) {
        logger.error("Caught error in batch", { trace, error, ...traceObject });
    } finally {
        await prisma.$disconnect();
    }
}
