// AI_CHECK: TEST_COVERAGE=2 | LAST: 2025-06-24
import { DbProvider, batch, logger } from "@vrooli/server";
import { camelCase, uppercaseFirstLetter, type ModelType, ModelType as ModelTypeEnum } from "@vrooli/shared";

/**
 * Type guard to check if a value is a valid ModelType
 */
function isValidModelType(value: string): value is ModelType {
    const modelTypeValues: readonly string[] = Object.values(ModelTypeEnum);
    return modelTypeValues.includes(value);
}


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
interface CountVerificationConfig<TSelect> {
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
    TFindManyArgs extends { select?: unknown },
    TPayload extends CountPayload,
    TSelect extends TFindManyArgs["select"]
>(config: CountVerificationConfig<TSelect>): Promise<void> {
    const { tableNames, countField, relationName, select, traceId } = config;

    async function processTableInBatches(
        tableName: string,
    ): Promise<void> {
        try {
            const modelTypeCandidate = uppercaseFirstLetter(camelCase(tableName));
            
            // Validate that the constructed string is a valid ModelType
            if (!isValidModelType(modelTypeCandidate)) {
                logger.error("Invalid ModelType constructed from table name", {
                    tableName,
                    modelTypeCandidate,
                    trace: traceId,
                });
                return;
            }
            
            await batch<TFindManyArgs, TPayload>({
                objectType: modelTypeCandidate,
                processBatch: async (batchItems) => {
                    // Type guards for safely accessing count fields
                    function hasCountField(obj: unknown, field: string): obj is Record<string, unknown> {
                        return obj !== null && typeof obj === "object" && field in obj;
                    }
                    
                    function isValidNumber(value: unknown): value is number {
                        return typeof value === "number";
                    }
                    
                    // Type guard to check if dbProvider has the table property
                    function hasTable(provider: unknown, table: string): provider is Record<string, unknown> {
                        return provider !== null && 
                               typeof provider === "object" && 
                               table in provider;
                    }
                    
                    // Type guard to check if dbModel has update method
                    function hasUpdateMethod(model: unknown): model is { update: (args: { where: { id: bigint }; data: Record<string, unknown> }) => Promise<unknown> } {
                        return model !== null &&
                               typeof model === "object" &&
                               "update" in model &&
                               typeof model.update === "function";
                    }

                    for (const item of batchItems) {
                        const actualCount = item._count[relationName];
                        
                        let cachedCount: number | undefined = undefined;
                        if (hasCountField(item, countField)) {
                            const fieldValue = item[countField];
                            if (isValidNumber(fieldValue)) {
                                cachedCount = fieldValue;
                            }
                        }
                        
                        if (cachedCount !== undefined && cachedCount !== actualCount) {
                            logger.warning(
                                `Updating ${tableName} ${item.id} ${countField} from ${cachedCount} to ${actualCount}.`,
                                { trace: traceId },
                            );
                            
                            const dbProvider = DbProvider.get();
                            
                            if (!hasTable(dbProvider, tableName)) {
                                logger.error(`Database provider does not have table: ${tableName}`, { trace: traceId });
                                continue;
                            }
                            
                            const dbModel = dbProvider[tableName];
                            
                            if (!hasUpdateMethod(dbModel)) {
                                logger.error(`Database model for ${tableName} does not have update method`, { trace: traceId });
                                continue;
                            }
                            // Validate that countField is a valid property name
                            if (!countField || typeof countField !== "string" || countField.includes("__proto__") || countField.includes("constructor") || countField.includes("prototype")) {
                                logger.error(`Invalid countField: ${countField}`, { trace: traceId, tableName });
                                continue;
                            }
                            
                            await dbModel.update({
                                where: { id: item.id },
                                data: { [countField]: actualCount },
                            });
                        }
                    }
                },
                select: select as TFindManyArgs["select"],
            });
        } catch (error: unknown) {
            logger.error(`verifyCountField processTableInBatches caught error for ${tableName}`, { 
                error, 
                trace: `${traceId}_batch_error`,
                tableName,
            });
        }
    }

    for (const tableName of tableNames) {
        await processTableInBatches(tableName);
    }
}
