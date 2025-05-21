/**
 * Calculates and stores embeddings for all searchable objects (that don't have one yet), including 
 * root objects, versions, and tags
 * 
 * Embeddings are used to find similar objects. For example, if you are 
 * looking for tags and type "ai", then the search should return tags 
 * like "Artificial Intelligence (AI)", "LLM", and "Machine Learning", even if "ai" 
 * is not directly in the tag's name/description.
 */
import { DbProvider, type EmbeddableType, type FindManyArgs, batch, logger } from "@local/server";
import { type ModelType, RunStatus } from "@local/shared";
import { type Prisma } from "@prisma/client";
import { EmbeddingService } from "../../../server/src/services/embedding.js";

// WARNING: Setting this to true will cause the embeddings to be recalculated for all objects. 
// This will take a long time and should only be done during development or if something is 
// wrong with the existing embeddings.
const RECALCULATE_EMBEDDINGS = true;

const API_BATCH_SIZE = 100; // Size set in the API to limit the number of embeddings generated at once

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
    Team: { id: true, name: true, translations: { select: { id: true, embeddingExpiredAt: true, language: true, bio: true } } },
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
type EmbedGetFn<P> = (row: P) => string[];

const embedGetMap: {
    Chat: EmbedGetFn<ChatEmbedPayload>;
    Issue: EmbedGetFn<IssueEmbedPayload>;
    Meeting: EmbedGetFn<MeetingEmbedPayload>;
    ResourceVersion: EmbedGetFn<ResourceVersionEmbedPayload>;
    Team: EmbedGetFn<TeamEmbedPayload>;
    Tag: EmbedGetFn<TagEmbedPayload>;
    User: EmbedGetFn<UserEmbedPayload>;
    Reminder: EmbedGetFn<ReminderEmbedPayload>;
    Run: EmbedGetFn<RunEmbedPayload>;
} = {
    Chat: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    Issue: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    Meeting: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    ResourceVersion: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: t.name, description: t.description }, t.language)),
    Team: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: row.name, bio: t.bio }, t.language)),
    Tag: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ description: t.description }, t.language)),
    User: (row) => row.translations.map(t => EmbeddingService.getEmbeddableString({ name: row.name, handle: row.handle, bio: t.bio }, t.language)),
    Reminder: (row) => [EmbeddingService.getEmbeddableString({ name: row.name, description: row.description }, undefined)],
    Run: (row) => [EmbeddingService.getEmbeddableString({ name: row.name }, undefined)],
};

// ===== END CENTRALIZED LOGIC =====

async function updateEmbedding(
    objectType: EmbeddableType | `${EmbeddableType}`,
    isTranslatableTable: boolean, // True if updating XYZTranslation table, false for main table
    id: string,
    embeddings: number[],
): Promise<void> {
    const { ModelMap } = await import("@local/server");
    const tableName = ModelMap.get(isTranslatableTable ? (objectType + "Translation" as ModelType) : objectType).dbTable;
    const embeddingsText = `ARRAY[${embeddings.join(", ")}]`;
    await DbProvider.get().$executeRawUnsafe(
        `UPDATE ${tableName}
       SET "embedding" = ${embeddingsText}, "embeddingExpiredAt" = NOW()
       WHERE id = $1::BIGINT;`,
        id,
    );
}

// Helper type guard
function hasPopulatedTranslations<T extends { translations?: Array<any> }>(
    payload: T,
): payload is T & { translations: NonNullable<T["translations"]> } {
    return payload && Array.isArray(payload.translations) && payload.translations.length > 0;
}

async function processEmbeddingBatch<
    P extends { id: bigint; embeddingExpiredAt?: Date | null; translations?: Array<{ id: bigint; embeddingExpiredAt: Date | null; language: string }> }
>(
    batchOfP: P[],
    objectType: EmbeddableType | `${EmbeddableType}`,
): Promise<void> {
    const getFn = embedGetMap[objectType as keyof typeof embedGetMap] as unknown as EmbedGetFn<P> | undefined;
    if (!getFn) {
        logger.warn("No embed get function found for object type", { objectType });
        return;
    }

    const itemsToEmbed: { sentence: string; targetId: string; updateTranslationTable: boolean }[] = [];

    batchOfP.forEach((row) => { // row is of type P
        const sentencesForRow = getFn(row);

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
type EmbeddingBatchProps<T extends FindManyArgs, P> = {
    objectType: EmbeddableType | `${EmbeddableType}`;
    processBatch: (batch: P[], objectType: `${EmbeddableType}`) => Promise<void>;
    trace: string;
    traceObject?: Record<string, any>;
    select: T["select"]; // 'select' is mandatory
    where: T["where"];
};

async function embeddingBatch<T extends FindManyArgs, P>({
    objectType,
    processBatch: processBatchCallback, // Renamed to avoid confusion with Prisma's batch
    trace,
    traceObject,
    select,
    where,
}: EmbeddingBatchProps<T, P>) {
    try {
        await batch<T, P>({ // Prisma's batch utility
            batchSize: API_BATCH_SIZE,
            objectType, // For Prisma's batch utility context if needed
            processBatch: async (rows) => processBatchCallback(rows, objectType), // Call the provided processBatch (i.e., processEmbeddingBatch)
            select,     // This is Prisma.ModelFindManyArgs['select'], aligning with P
            where,
        });
    } catch (error) {
        logger.error("embeddingBatch caught error", { error, trace, ...traceObject });
    }
}

// --- Update all batchEmbeddingsXYZ functions ---

async function batchEmbeddingsChat() {
    return embeddingBatch<Prisma.chatFindManyArgs, ChatEmbedPayload>({
        objectType: "Chat",
        select: embedSelectMap.Chat,
        processBatch: processEmbeddingBatch, // Use the consolidated helper
        trace: "0475",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }) },
    });
}

async function batchEmbeddingsIssue() {
    return embeddingBatch<Prisma.issueFindManyArgs, IssueEmbedPayload>({
        objectType: "Issue",
        select: embedSelectMap.Issue,
        processBatch: processEmbeddingBatch,
        trace: "0476",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }) },
    });
}

async function batchEmbeddingsMeeting() {
    return embeddingBatch<Prisma.meetingFindManyArgs, MeetingEmbedPayload>({
        objectType: "Meeting",
        select: embedSelectMap.Meeting,
        processBatch: processEmbeddingBatch,
        trace: "0477",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }) },
    });
}

async function batchEmbeddingsTeam() {
    return embeddingBatch<Prisma.teamFindManyArgs, TeamEmbedPayload>({
        objectType: "Team",
        select: embedSelectMap.Team,
        processBatch: processEmbeddingBatch,
        trace: "0478",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }) },
    });
}

async function batchEmbeddingsResourceVersion() {
    return embeddingBatch<Prisma.resource_versionFindManyArgs, ResourceVersionEmbedPayload>({
        objectType: "ResourceVersion",
        select: embedSelectMap.ResourceVersion,
        processBatch: processEmbeddingBatch,
        trace: "0472",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

async function batchEmbeddingsTag() {
    return embeddingBatch<Prisma.tagFindManyArgs, TagEmbedPayload>({
        objectType: "Tag",
        select: embedSelectMap.Tag,
        processBatch: processEmbeddingBatch,
        trace: "0485",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }) },
    });
}

async function batchEmbeddingsUser() {
    return embeddingBatch<Prisma.userFindManyArgs, UserEmbedPayload>({
        objectType: "User",
        select: embedSelectMap.User,
        processBatch: processEmbeddingBatch,
        trace: "0486",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }) },
    });
}

async function batchEmbeddingsReminder() {
    return embeddingBatch<Prisma.reminderFindManyArgs, ReminderEmbedPayload>({
        objectType: "Reminder",
        select: embedSelectMap.Reminder,
        processBatch: processEmbeddingBatch,
        trace: "0482",
        where: { ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingExpiredAt: { lte: new Date() } }) },
    });
}

async function batchEmbeddingsRun() {
    return embeddingBatch<Prisma.runFindManyArgs, RunEmbedPayload>({
        objectType: "Run",
        select: embedSelectMap.Run,
        processBatch: processEmbeddingBatch,
        trace: "0483",
        where: {
            status: { in: [RunStatus.Scheduled, RunStatus.InProgress] },
            ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingExpiredAt: { lte: new Date() } }),
        },
    });
}

export async function generateEmbeddings() {
    await batchEmbeddingsChat();
    await batchEmbeddingsIssue();
    await batchEmbeddingsMeeting();
    await batchEmbeddingsTeam();
    await batchEmbeddingsResourceVersion();
    await batchEmbeddingsTag();
    await batchEmbeddingsUser();
    await batchEmbeddingsReminder();
    await batchEmbeddingsRun();
}
