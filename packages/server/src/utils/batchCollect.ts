import { GqlModelType } from "@local/shared";
import { ObjectMap } from "../models/base";
import { PrismaType } from "../types";

export const DEFAULT_BATCH_SIZE = 100;

export type FindManyArgs = {
    select?: any,
    where?: any,
}

export interface BatchCollectProps<T extends FindManyArgs> {
    batchSize?: number,
    objectType: GqlModelType | `${GqlModelType}`
    prisma: PrismaType,
    processBatch: (batch: any[]) => Promise<void>,
    select: T["select"],
    where?: T["where"],
}

/**
 * Processes data from the database in batches, 
 * as to not overload the database with a large query
 */
export const batchCollect = async <T extends FindManyArgs>({
    batchSize = DEFAULT_BATCH_SIZE,
    objectType,
    prisma,
    processBatch,
    select,
    where,
}: BatchCollectProps<T>) => {
    const delegate = ObjectMap[objectType]!.delegate(prisma);
    let skip = 0;
    let currentBatchSize = 0;

    do {
        // Find all entities according to the given options
        const batch = await delegate.findMany({
            select,
            skip,
            take: batchSize,
            where,
        });
        skip += batchSize;
        currentBatchSize = batch.length;

        await processBatch(batch);
    } while (currentBatchSize === batchSize);
};
