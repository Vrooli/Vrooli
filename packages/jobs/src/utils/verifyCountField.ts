import type { Prisma } from "@prisma/client";
import pkg from "@prisma/client";
import { DbProvider, batch, logger } from "@vrooli/server";
import { camelCase, uppercaseFirstLetter, type ModelType } from "@vrooli/shared";

const { PrismaClient } = pkg;

/**
 * Generic interface for count verification payload
 */
interface CountPayload {
    id: bigint;
    _count: Record<string, number>;
}

/**
 * Configuration for count verification
 */
interface CountVerificationConfig<TSelect extends Prisma.SelectSubset<any, any>> {
    /** Array of table names to process */
    tableNames: readonly string[];
    /** Field name containing the cached count */
    countField: string;
    /** Relation name for the actual count (_count.{relationName}) */
    relationName: string;
    /** Prisma select shape */
    select: TSelect;
    /** Trace ID for logging */
    traceId: string;
}

/**
 * Generic function to verify count fields across multiple tables
 * Compares cached count fields with actual relation counts and updates mismatches
 */
export async function verifyCountField<
    TFindManyArgs extends Record<string, any>,
    TPayload extends CountPayload,
    TSelect extends Prisma.SelectSubset<any, any>
>(config: CountVerificationConfig<TSelect>): Promise<void> {
    const { tableNames, countField, relationName, select, traceId } = config;

    async function processTableInBatches(
        tableName: keyof InstanceType<typeof PrismaClient>
    ): Promise<void> {
        try {
            await batch<TFindManyArgs, TPayload>({
                objectType: uppercaseFirstLetter(camelCase(tableName as string)) as ModelType,
                processBatch: async (batchItems) => {
                    for (const item of batchItems) {
                        const actualCount = item._count[relationName];
                        const cachedCount = (item as any)[countField];
                        
                        if (cachedCount !== actualCount) {
                            logger.warning(
                                `Updating ${tableName as string} ${item.id} ${countField} from ${cachedCount} to ${actualCount}.`,
                                { trace: traceId }
                            );
                            
                            await (DbProvider.get()[tableName] as { update: any }).update({
                                where: { id: item.id },
                                data: { [countField]: actualCount },
                            });
                        }
                    }
                },
                select,
            });
        } catch (error) {
            logger.error(`verifyCountField processTableInBatches caught error for ${tableName as string}`, { 
                error, 
                trace: `${traceId}_batch_error`,
                tableName: tableName as string 
            });
        }
    }

    for (const tableName of tableNames) {
        await processTableInBatches(tableName);
    }
}