import { ErrorTrace } from "../events";
import { PrismaType } from "../types";
import { batchCollect, BatchCollectProps, FindManyArgs } from "./batchCollect";
import { withPrisma } from "./withPrisma";


export interface BatchProps<T extends FindManyArgs> extends Omit<BatchCollectProps<T>, "prisma" | "processBatch"> {
    processBatch: (batch: any[], prisma: PrismaType) => Promise<void>,
    trace: string,
    traceObject?: ErrorTrace,
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
    await withPrisma({
        process: async (prisma) => {
            await batchCollect<T>({
                prisma,
                processBatch: async (batch) => await processBatch(batch, prisma),
                ...props,
            });
        },
        trace,
        traceObject,
    });
}
