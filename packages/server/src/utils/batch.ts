import { GqlModelType } from "@local/shared";
import { prismaInstance } from "../db/instance";
import { ModelMap } from "../models/base";

export const DEFAULT_BATCH_SIZE = 100;

export type FindManyArgs = {
    select?: unknown,
    where?: unknown,
}

export interface BatchProps<T extends FindManyArgs> {
    batchSize?: number,
    objectType: GqlModelType | `${GqlModelType}`
    processBatch: (batch: any[]) => Promise<void>,
    select: T["select"],
    where?: T["where"],
}

/**
 * Processes data from the database in batches, 
 * as to not overload the database with a large query
 */
export const batch = async <T extends FindManyArgs>({
    batchSize = DEFAULT_BATCH_SIZE,
    objectType,
    processBatch,
    select,
    where,
}: BatchProps<T>) => {
    const delegate = ModelMap.get(objectType).delegate(prismaInstance);
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

interface BatchGroupProps<T extends FindManyArgs, R> extends Omit<BatchProps<T>, "processBatch"> {
    initialResult: R,
    processBatch: (batch: any[], result: R) => void,
    finalizeResult?: (result: R) => R,
}

/**
 * Processes data from the database in batches, while also handling the 
 * accumulation and finalization of the result
 */
export const batchGroup = async <T extends FindManyArgs, R>({
    initialResult,
    processBatch,
    finalizeResult,
    ...props
}: BatchGroupProps<T, R>): Promise<R> => {
    const result = initialResult;
    await batch<T>({
        ...props,
        processBatch: async (batch) => processBatch(batch, result),
    });
    return finalizeResult ? finalizeResult(result) : result;
};
