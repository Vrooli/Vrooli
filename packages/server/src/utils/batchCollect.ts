import { GqlModelType } from "@local/shared";
import { PrismaClient } from "@prisma/client";
import { ObjectMap } from "../models";

export const DEFAULT_BATCH_SIZE = 100;

type FindManyArgs = {
    select?: any,
    where?: any,
}

type BatchCollectProps<T extends FindManyArgs> = {
    batchSize?: number,
    objectType: GqlModelType | `${GqlModelType}`
    prisma: PrismaClient,
    processData: (batch: any[]) => Promise<void>,
    select: T["select"],
    where?: T["where"],
}

export const batchCollect = async <T extends FindManyArgs>({
    batchSize = DEFAULT_BATCH_SIZE,
    objectType,
    prisma,
    processData,
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

        await processData(batch);
    } while (currentBatchSize === batchSize);
};
