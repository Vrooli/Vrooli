import { type ModelType } from "@local/shared";
import { type PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";

export const DEFAULT_BATCH_SIZE = 100;

export type FindManyArgs = {
    select?: unknown,
    where?: unknown,
}

export interface BatchProps<T extends FindManyArgs, R = unknown> {
    batchSize?: number,
    objectType: ModelType | `${ModelType}`
    processBatch: (batch: R[]) => Promise<void>,
    select: T["select"],
    where?: T["where"],
}

/**
 * Processes data from the database in batches, 
 * as to not overload the database with a large query
 */
export async function batch<T extends FindManyArgs, R = unknown>({
    batchSize = DEFAULT_BATCH_SIZE,
    objectType,
    processBatch,
    select,
    where,
}: BatchProps<T, R>) {
    const { ModelMap } = await import("../models/base/index.js");
    const delegate = DbProvider.get()[ModelMap.get(objectType).dbTable] as PrismaDelegate;
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

        await processBatch(batch as R[]);
    } while (currentBatchSize === batchSize);
}

interface BatchGroupProps<T extends FindManyArgs, R> extends Omit<BatchProps<T, R>, "processBatch"> {
    initialResult: R,
    processBatch: (batch: R[], result: R) => void,
    finalizeResult?: (result: R) => R,
}

/**
 * Processes data from the database in batches, while also handling the 
 * accumulation and finalization of the result
 */
export async function batchGroup<T extends FindManyArgs, R>({
    initialResult,
    processBatch,
    finalizeResult,
    ...props
}: BatchGroupProps<T, R>): Promise<R> {
    const result = initialResult;
    await batch<T, R>({
        ...props,
        processBatch: async (batch) => processBatch(batch as R[], result),
    });
    return finalizeResult ? finalizeResult(result) : result;
}
