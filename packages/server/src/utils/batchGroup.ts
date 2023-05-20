import { batchCollect, BatchCollectProps, FindManyArgs } from "./batchCollect";

interface BatchGroupProps<T extends FindManyArgs, R> extends Omit<BatchCollectProps<T>, "processBatch"> {
    initialResult: R,
    processBatch: (batch: any[], result: R) => void,
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
    await batchCollect<T>({
        ...props,
        processBatch: async (batch) => processBatch(batch, result),
    });
    return finalizeResult ? finalizeResult(result) : result;
}
