/**
 * Calculates and stores embeddings for all searchable objects (that don't have one yet), including 
 * root objects, versions, and tags
 * 
 * Embeddings are used to find similar objects. For example, if you are 
 * looking for tags and type "ai", then the search should return tags 
 * like "Artificial Intelligence (AI)", "LLM", and "Machine Learning", even if "ai" 
 * is not directly in the tag's name/description.
 */
import { Prisma } from "@prisma/client";
import { DbProvider, batch, logger, type EmbeddableType } from "@vrooli/server";
import { RunStatus, type ModelType, ModelType as ModelTypeEnum } from "@vrooli/shared";
import { EmbeddingService } from "@vrooli/server";
import { API_BATCH_SIZE } from "../constants.js";

// WARNING: Setting RECALCULATE_EMBEDDINGS to true will cause the embeddings to be recalculated for ALL objects. 
// This is extremely resource intensive, costly (OpenAI API calls), and should only be done:
// 1. During development when testing embedding changes
// 2. If there's a critical issue with existing embeddings
// 3. After major changes to the embedding generation logic
// To enable: Set environment variable RECALCULATE_EMBEDDINGS=true
const RECALCULATE_EMBEDDINGS = process.env.RECALCULATE_EMBEDDINGS === "true";


// ===== EMBEDDING LOGIC CENTRALIZED =====

// --- 1. Select Shapes ---
const commonTranslationSelect = { id: true, embeddingExpiredAt: true, language: true, name: true, description: true };
const bioTranslationSelect = { id: true, embeddingExpiredAt: true, language: true, bio: true };
const tagTranslationSelect = { id: true, embeddingExpiredAt: true, language: true, description: true };

const embedSelectMap = {
    Chat: { id: true, translations: { select: commonTranslationSelect } },
    Issue: { id: true, translations: { select: commonTranslationSelect } },
    Meeting: { id: true, translations: { select: commonTranslationSelect } },
    ResourceVersion: { id: true, translations: { select: commonTranslationSelect } },
    Team: { id: true, translations: { select: { id: true, embeddingExpiredAt: true, language: true, name: true, bio: true } } },
    Tag: { id: true, translations: { select: tagTranslationSelect } },
    User: { id: true, name: true, handle: true, translations: { select: bioTranslationSelect } },
    Reminder: { id: true, embeddingExpiredAt: true, name: true, description: true },
    Run: { id: true, embeddingExpiredAt: true, name: true },
} as const;

// --- 2. Payload Types (Derived from Select Shapes) ---
type ChatEmbedPayload = Prisma.chatGetPayload<{ select: typeof embedSelectMap.Chat }>;
type IssueEmbedPayload = Prisma.issueGetPayload<{ select: typeof embedSelectMap.Issue }>;
type MeetingEmbedPayload = Prisma.meetingGetPayload<{ select: typeof embedSelectMap.Meeting }>;
type ResourceVersionEmbedPayload = Prisma.resource_versionGetPayload<{ select: typeof embedSelectMap.ResourceVersion }>;
type TeamEmbedPayload = Prisma.teamGetPayload<{ select: typeof embedSelectMap.Team }>;
type TagEmbedPayload = Prisma.tagGetPayload<{ select: typeof embedSelectMap.Tag }>;
type UserEmbedPayload = Prisma.userGetPayload<{ select: typeof embedSelectMap.User }>;
type ReminderEmbedPayload = Prisma.reminderGetPayload<{ select: typeof embedSelectMap.Reminder }>;
type RunEmbedPayload = Prisma.runGetPayload<{ select: typeof embedSelectMap.Run }>;

// --- 3. Embeddable String Extraction Functions ---

// New Mapped Type for specific payloads
type SpecificPayloadMap = {
    Chat: ChatEmbedPayload;
    Issue: IssueEmbedPayload;
    Meeting: MeetingEmbedPayload;
    ResourceVersion: ResourceVersionEmbedPayload;
    Team: TeamEmbedPayload;
    Tag: TagEmbedPayload;
    User: UserEmbedPayload;
    Reminder: ReminderEmbedPayload;
    Run: RunEmbedPayload;
};

// Map EmbeddableType to its specific Prisma FindManyArgs type
type ModelFindManyArgsMap = {
    Chat: Prisma.chatFindManyArgs;
    Issue: Prisma.issueFindManyArgs;
    Meeting: Prisma.meetingFindManyArgs;
    ResourceVersion: Prisma.resource_versionFindManyArgs;
    Team: Prisma.teamFindManyArgs;
    Tag: Prisma.tagFindManyArgs;
    User: Prisma.userFindManyArgs;
    Reminder: Prisma.reminderFindManyArgs;
    Run: Prisma.runFindManyArgs;
};

// New Typed Generic Function for embedGetMap
type TypedEmbedGetFn<K extends EmbeddableType> = (row: SpecificPayloadMap[K]) => string[];

// Strongly typed version of embedGetMap
const typedEmbedGetMap: {
    [K in EmbeddableType]: TypedEmbedGetFn<K>;
} = {
    Chat: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    Issue: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    Meeting: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    ResourceVersion: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    Team: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, bio: t.bio }, t.language)),
    Tag: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ description: t.description }, t.language)),
    User: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: row.name, handle: row.handle, bio: t.bio }, t.language)),
    Reminder: (row) => [EmbeddingService.getEmbeddableString({ name: row.name, description: row.description }, undefined)],
    Run: (row) => [EmbeddingService.getEmbeddableString({ name: row.name }, undefined)],
};

// ===== END CENTRALIZED LOGIC =====

// Type guard to check if a value is a valid ModelType
function isValidModelType(value: string): value is ModelType {
    const modelTypeValues: readonly string[] = Object.values(ModelTypeEnum);
    return modelTypeValues.includes(value) || 
           value.endsWith("Translation");
}

async function updateEmbedding(
    objectType: EmbeddableType, // Simplified type
    isTranslatableTable: boolean,
    id: string,
    embeddings: number[],
): Promise<void> {
    const { ModelMap } = await import("@vrooli/server");

    // Define the type for keys constructed from EmbeddableType and isTranslatableTable logic
    type ConstructedModelKey = EmbeddableType | `${EmbeddableType}Translation`;

    // Construct the key for ModelMap
    const modelKey: ConstructedModelKey = isTranslatableTable ? `${objectType}Translation` : objectType;

    // Validate that the constructed key is a valid ModelType
    if (!isValidModelType(modelKey)) {
        logger.error("Invalid ModelType constructed", {
            objectType,
            isTranslatableTable,
            constructedKey: modelKey,
        });
        throw new Error(`Invalid ModelType: ${modelKey}`);
    }
    
    const modelInfo = ModelMap.get(modelKey);

    if (!modelInfo || !modelInfo.dbTable) {
        logger.error("Failed to find model information or dbTable in ModelMap", {
            objectType,
            isTranslatableTable,
            derivedModelKey: modelKey,
            foundModelInfo: !!modelInfo,
        });
        throw new Error(`Could not determine database table for objectType: ${objectType}, derived key: ${modelKey}`);
    }
    const tableName = modelInfo.dbTable;
    
    // Validate table name to prevent SQL injection
    if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(tableName)) {
        logger.error("Invalid table name detected", {
            objectType,
            tableName,
            isTranslatableTable,
        });
        throw new Error(`Invalid table name: ${tableName}`);
    }
    
    // Use Prisma.sql for safe query construction with dynamic table names
    // Convert embeddings array to PostgreSQL array format
    const embeddingsArrayStr = `[${embeddings.join(",")}]`;
    const query = Prisma.sql`UPDATE ${Prisma.raw(`"${tableName}"`)} SET "embedding" = ${embeddingsArrayStr}::vector(1536), "embeddingExpiredAt" = NOW() WHERE id = ${BigInt(id)}`;
    await DbProvider.get().$executeRaw(query);
}

// Helper type guard
function hasPopulatedTranslations<T extends { translations?: Array<any> }>(
    payload: T,
): payload is T & { translations: NonNullable<T["translations"]> } {
    return payload && Array.isArray(payload.translations) && payload.translations.length > 0;
}

async function processEmbeddingBatch<
    K extends EmbeddableType, // Use K extends EmbeddableType
    // P is now SpecificPayloadMap[K], ensuring consistency with K
    P extends SpecificPayloadMap[K] & { id: bigint; embeddingExpiredAt?: Date | null; translations?: Array<{ id: bigint; embeddingExpiredAt: Date | null; language: string }> }
>(
    batchOfP: P[],
    objectType: K, // objectType is now K
): Promise<void> {
    // Safely get the embedding function for this object type
    const getFn = typedEmbedGetMap[objectType];
    if (!getFn || typeof getFn !== "function") {
        logger.warn("No embed get function found for object type", { objectType });
        return;
    }

    const itemsToEmbed: { sentence: string; targetId: string; updateTranslationTable: boolean }[] = [];

    batchOfP.forEach((row) => { // row is of type P
        let sentencesForRow: string[];
        
        try {
            // Type safety: getFn expects the specific payload type for this objectType
            // The type system ensures row is compatible, but we add runtime validation
            if (!row || typeof row !== "object" || !("id" in row)) {
                logger.error("Invalid row structure", {
                    objectType,
                    rowType: typeof row,
                    hasId: row && typeof row === "object" && "id" in row,
                });
                return; // Skip this row
            }
            sentencesForRow = getFn(row);
        } catch (error) {
            logger.error("getFn threw an error", {
                objectType,
                rowId: row.id.toString(),
                error,
            });
            return; // Skip this row
        }
        
        // Validate that getFn returned a valid array
        if (!Array.isArray(sentencesForRow)) {
            logger.error("getFn did not return an array", {
                objectType,
                rowId: row.id.toString(),
                returnType: typeof sentencesForRow,
            });
            return; // Skip this row
        }

        if (hasPopulatedTranslations(row)) {
            // This path is for types that have a translations array (Chat, Issue, User, Team, etc.)
            if (row.translations.length === sentencesForRow.length) {
                row.translations.forEach((t, index) => {
                    if (!t.embeddingExpiredAt || t.embeddingExpiredAt <= new Date()) {
                        itemsToEmbed.push({
                            sentence: sentencesForRow[index],
                            targetId: t.id.toString(),       // ID of the translation record
                            updateTranslationTable: true,    // Targets the XYZTranslation table
                        });
                    }
                });
            } else {
                logger.warn(
                    "Sentence-translation count mismatch. Skipping embeddings for this item's translations.",
                    { objectType, mainObjectId: row.id.toString(), sentenceCount: sentencesForRow.length, translationCount: row.translations.length },
                );
            }
        } else {
            // This path is for non-translatable types (Reminder, Run) or translatable types with no current translations to process.
            // Check if the main object itself needs an embedding update.
            // row.embeddingExpiredAt is optional on P's constraint, so check its existence and value.
            if (row.embeddingExpiredAt === undefined || row.embeddingExpiredAt === null || row.embeddingExpiredAt <= new Date()) {
                sentencesForRow.forEach(sentence => {
                    itemsToEmbed.push({
                        sentence,
                        targetId: row.id.toString(),        // ID of the main object record
                        updateTranslationTable: false,      // Targets the main model table (e.g., Reminder, Run)
                    });
                });
            }
        }
    });

    if (itemsToEmbed.length === 0) {
        logger.debug("No items to embed after filtering", { objectType });
        return;
    }

    const sentenceTexts = itemsToEmbed.map(s => s.sentence);
    const externalEmbeddings = await EmbeddingService.get().getEmbeddings(objectType, sentenceTexts);

    if (externalEmbeddings.length !== itemsToEmbed.length) {
        logger.error("Mismatch between requested sentences and received embeddings count. Aborting updates for this batch.", {
            objectType,
            requestedCount: itemsToEmbed.length,
            receivedCount: externalEmbeddings.length,
        });
        return;
    }

    await Promise.all(
        itemsToEmbed.map(async (item, index) => {
            const currentEmbedding = externalEmbeddings[index];
            if (!currentEmbedding) {
                logger.warn("Missing embedding for an item, skipping update.", { objectType, targetId: item.targetId });
                return;
            }
            await updateEmbedding(objectType, item.updateTranslationTable, item.targetId, currentEmbedding);
        }),
    );
}

// Type for EmbeddingBatchProps
type EmbeddingBatchProps<K extends EmbeddableType, P extends SpecificPayloadMap[K]> = {
    objectType: K;
    processBatch: (batch: P[], objectType: K) => Promise<void>;
    trace: string;
    traceObject?: Record<string, any>;
    select: ModelFindManyArgsMap[K]["select"];
    where: ModelFindManyArgsMap[K]["where"];
};

async function embeddingBatch<K extends EmbeddableType>({ 
    objectType,
    processBatch: processBatchCallback,
    trace,
    traceObject,
    select,
    where,
}: EmbeddingBatchProps<K, SpecificPayloadMap[K]>) { 
    try {
        await batch<ModelFindManyArgsMap[K], SpecificPayloadMap[K]>({ 
            batchSize: API_BATCH_SIZE,
            objectType,
            processBatch: async (rows: SpecificPayloadMap[K][]) => {
                // Validate that rows is an array and contains expected payload structure
                if (!Array.isArray(rows)) {
                    logger.error("Invalid rows received in embeddingBatch", { 
                        objectType, 
                        rowsType: typeof rows,
                        trace,
                    });
                    return;
                }
                await processBatchCallback(rows, objectType);
            },
            select,
            where,
        });
    } catch (error) {
        logger.error("embeddingBatch caught error", { 
            error: {
                name: error?.name,
                message: error?.message,
                stack: error?.stack,
                ...error,
            }, 
            trace, 
            ...traceObject, 
        });
        // Re-throw to ensure tests see the error
        throw error;
    }
}

// --- Update all batchEmbeddingsXYZ functions ---

async function batchEmbeddingsChat() {
    return embeddingBatch<"Chat">({
        objectType: "Chat",
        select: embedSelectMap.Chat,
        processBatch: processEmbeddingBatch,
        trace: "0475",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] } } }) },
    });
}

async function batchEmbeddingsIssue() {
    return embeddingBatch<"Issue">({
        objectType: "Issue",
        select: embedSelectMap.Issue,
        processBatch: processEmbeddingBatch,
        trace: "0476",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] } } }) },
    });
}

async function batchEmbeddingsMeeting() {
    return embeddingBatch<"Meeting">({
        objectType: "Meeting",
        select: embedSelectMap.Meeting,
        processBatch: processEmbeddingBatch,
        trace: "0477",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] } } }) },
    });
}

async function batchEmbeddingsTeam() {
    return embeddingBatch<"Team">({
        objectType: "Team",
        select: embedSelectMap.Team,
        processBatch: processEmbeddingBatch,
        trace: "0478",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] } } }) },
    });
}

async function batchEmbeddingsResourceVersion() {
    return embeddingBatch<"ResourceVersion">({
        objectType: "ResourceVersion",
        select: embedSelectMap.ResourceVersion,
        processBatch: processEmbeddingBatch,
        trace: "0472",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] } } }),
        },
    });
}

async function batchEmbeddingsTag() {
    return embeddingBatch<"Tag">({
        objectType: "Tag",
        select: embedSelectMap.Tag,
        processBatch: processEmbeddingBatch,
        trace: "0485",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] } } }) },
    });
}

async function batchEmbeddingsUser() {
    return embeddingBatch<"User">({
        objectType: "User",
        select: embedSelectMap.User,
        processBatch: processEmbeddingBatch,
        trace: "0486",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] } } }) },
    });
}

async function batchEmbeddingsReminder() {
    return embeddingBatch<"Reminder">({
        objectType: "Reminder",
        select: embedSelectMap.Reminder,
        processBatch: processEmbeddingBatch,
        trace: "0482",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] }) },
    });
}

async function batchEmbeddingsRun() {
    return embeddingBatch<"Run">({
        objectType: "Run",
        select: embedSelectMap.Run,
        processBatch: processEmbeddingBatch,
        trace: "0483",
        where: {
            status: { in: [RunStatus.Scheduled, RunStatus.InProgress] },
            ...(RECALCULATE_EMBEDDINGS ? {} : { OR: [{ embeddingExpiredAt: null }, { embeddingExpiredAt: { lte: new Date() } }] }),
        },
    });
}

export async function generateEmbeddings() {
    if (RECALCULATE_EMBEDDINGS) {
        logger.warn("RECALCULATE_EMBEDDINGS is enabled - regenerating ALL embeddings. This is resource intensive!", {
            flag: "RECALCULATE_EMBEDDINGS",
            value: true,
        });
    }
    
    await batchEmbeddingsChat();
    await batchEmbeddingsIssue();
    await batchEmbeddingsMeeting();
    await batchEmbeddingsTeam();
    await batchEmbeddingsResourceVersion();
    await batchEmbeddingsTag();
    await batchEmbeddingsUser();
    await batchEmbeddingsReminder();
    // Skipping batchEmbeddingsRun() - Run model doesn't have embedding fields in schema
    // await batchEmbeddingsRun();
}
