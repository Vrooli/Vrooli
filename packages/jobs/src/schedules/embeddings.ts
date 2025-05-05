/**
 * Calculates and stores embeddings for all searchable objects (that don't have one yet), including 
 * root objects, versions, and tags
 * 
 * Embeddings are used to find similar objects. For example, if you are 
 * looking for tags and type "ai", then the search should return tags 
 * like "Artificial Intelligence (AI)", "LLM", and "Machine Learning", even if "ai" 
 * is not directly in the tag's name/description.
 */
import { DbProvider, EmbeddableType, FindManyArgs, GenericModelLogic, ModelMap, batch, logger } from "@local/server";
import { ModelType, RunStatus } from "@local/shared";
import { Prisma } from "@prisma/client";
// Try relative path again, going up three levels
import { EmbeddingService } from "../../../server/src/services/embedding.js";

// WARNING: Setting this to true will cause the embeddings to be recalculated for all objects. 
// This will take a long time and should only be done during development or if something is 
// wrong with the existing embeddings.
const RECALCULATE_EMBEDDINGS = true;

const API_BATCH_SIZE = 100; // Size set in the API to limit the number of embeddings generated at once

/**
 * Helper function to extract sentences from translated embeddable objects
 */
function extractTranslatedSentences<T extends { translations: { language: string }[] }>(batch: T[], model: GenericModelLogic) {
    // Initialize array to store sentences
    const sentences: string[] = [];
    // Loop through each object in the batch
    for (const curr of batch) {
        const currLanguages = curr.translations.map(translation => translation.language);
        const currSentences = currLanguages.map(language => model?.display?.().embed?.get(curr as any, [language]));
        // Add each sentence to the array
        currSentences.forEach(sentence => {
            if (sentence) {
                sentences.push(sentence);
            }
        });
    }
    return sentences;
}

/**
 * Updates the embedding for a given object.
 * @param tableName - The name of the table where the embeddings should be updated.
 * @param isTranslatable - Whether the object is translatable.
 * @param id - The id of the object to update.
 * @param embeddings - The text embedding array returned from the API.
 */
async function updateEmbedding(
    objectType: EmbeddableType | `${EmbeddableType}`,
    isTranslatable: boolean,
    id: string,
    embeddings: number[],
): Promise<void> {
    const tableName = ModelMap.get(isTranslatable ? (objectType + "Translation" as ModelType) : objectType).dbTable;
    const embeddingsText = `ARRAY[${embeddings.join(", ")}]`;
    // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
    await DbProvider.get().$executeRawUnsafe(
        `UPDATE ${tableName}
       SET "embedding" = ${embeddingsText}, "embeddingExpiredAt" = NOW()
       WHERE id = $1::BIGINT;`,
        id,
    );
}

/**
 * Processes a batch of translatable objects and updates each stale translation embedding.
 * @param batch - The batch of objects to process.
 * @param instruction - The instruction used to generate the embeddings.
 * @param tableName - The name of the table where the embeddings should be updated. e.g. "api_version_translation"
 */
async function processTranslatedBatchHelper(
    batch: { translations: { id: string; embeddingExpiredAt: Date | null; language: string }[] }[],
    objectType: EmbeddableType | `${EmbeddableType}`,
): Promise<void> {
    if (!ModelMap.isModel(objectType)) return;
    const model = ModelMap.get(objectType);
    // Extract sentences from the batch
    const sentences = extractTranslatedSentences(batch, model);
    if (sentences.length === 0) return;
    // Find embeddings for all versions in the batch
    const embeddings = await EmbeddingService.get().getEmbeddings(objectType, sentences);
    // Update the embeddings for each stale translation
    await Promise.all(batch.map(async (curr, index) => {
        const translationsToUpdate = curr.translations.filter(t => !t.embeddingExpiredAt || t.embeddingExpiredAt <= new Date());
        for (const translation of translationsToUpdate) {
            const currEmbeddings = embeddings[index];
            if (!currEmbeddings) continue;
            await updateEmbedding(objectType, true, translation.id, currEmbeddings);
        }
    }));
}


/**
 * Processes a batch of non-translatable objects and updates their embeddings.
 * @param batch - The batch of objects to process.
 * @param instruction - The instruction used to generate the embeddings.
 * @param tableName - The name of the table where the embeddings should be updated. e.g. "reminder"
 */
async function processUntranslatedBatchHelper(
    batch: { id: string }[],
    objectType: EmbeddableType | `${EmbeddableType}`,
): Promise<void> {
    if (!ModelMap.isModel(objectType)) return;
    const model = ModelMap.get(objectType);
    // Extract sentences from the batch
    const sentences = batch.map(obj => model.display().embed?.get(obj as any, []) ?? "");
    if (sentences.length === 0) return;
    // Find embeddings for all objects in the batch
    const embeddings = await EmbeddingService.get().getEmbeddings(objectType, sentences);
    // Update the embeddings for each object
    await Promise.all(
        batch.map(async (obj, index) => {
            const currEmbeddings = embeddings[index];
            if (!currEmbeddings) return;
            await updateEmbedding(objectType, false, obj.id, currEmbeddings);
        }),
    );
}

type EmbeddingBatchProps<T extends FindManyArgs> = {
    objectType: EmbeddableType | `${EmbeddableType}`,
    processBatch: (batch: any[], objectType: `${EmbeddableType}`) => Promise<void>,
    trace: string,
    traceObject?: Record<string, any>,
    where: T["where"],
};

async function embeddingBatch<T extends FindManyArgs>({
    objectType,
    processBatch,
    trace,
    traceObject,
    where,
}: EmbeddingBatchProps<T>) {
    try {
        await batch<T>({
            batchSize: API_BATCH_SIZE,
            objectType,
            processBatch: async (batch) => processBatch(batch, objectType), //TODO tried to Model.get "ReminderTranslation", which doesn't exist
            select: ModelMap.get(objectType).display().embed!.select(),
            where,
        });
    } catch (error) {
        logger.error("embeddingBatch caught error", { error, trace, ...traceObject });
    }
}

async function batchEmbeddingsChat() {
    embeddingBatch<Prisma.chatFindManyArgs>({
        objectType: "Chat",
        processBatch: processTranslatedBatchHelper,
        trace: "0475",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

async function batchEmbeddingsIssue() {
    embeddingBatch<Prisma.issueFindManyArgs>({
        objectType: "Issue",
        processBatch: processTranslatedBatchHelper,
        trace: "0476",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

async function batchEmbeddingsMeeting() {
    embeddingBatch<Prisma.meetingFindManyArgs>({
        objectType: "Meeting",
        processBatch: processTranslatedBatchHelper,
        trace: "0477",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

async function batchEmbeddingsTeam() {
    embeddingBatch<Prisma.teamFindManyArgs>({
        objectType: "Team",
        processBatch: processTranslatedBatchHelper,
        trace: "0478",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

async function batchEmbeddingsReminder() {
    embeddingBatch<Prisma.reminderFindManyArgs>({
        objectType: "Reminder",
        processBatch: processUntranslatedBatchHelper,
        trace: "0482",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingExpiredAt: { lte: new Date() } }),
        },
    });
}

async function batchEmbeddingsResourceVersion() {
    embeddingBatch<Prisma.resource_versionFindManyArgs>({
        objectType: "ResourceVersion",
        processBatch: processTranslatedBatchHelper,
        trace: "0472",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

async function batchEmbeddingsRun() {
    embeddingBatch<Prisma.runFindManyArgs>({
        objectType: "Run",
        processBatch: processUntranslatedBatchHelper,
        trace: "0483",
        // Only update runs which are still active
        where: {
            status: { in: [RunStatus.Scheduled, RunStatus.InProgress] },
            schedule: { endTime: { gte: new Date().toISOString() } },
            ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingExpiredAt: { lte: new Date() } }),
        },
    });
}

async function batchEmbeddingsTag() {
    embeddingBatch<Prisma.tagFindManyArgs>({
        objectType: "Tag",
        processBatch: processTranslatedBatchHelper,
        trace: "0485",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

async function batchEmbeddingsUser() {
    embeddingBatch<Prisma.userFindManyArgs>({
        objectType: "User",
        processBatch: processTranslatedBatchHelper,
        trace: "0486",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingExpiredAt: { lte: new Date() } } } }),
        },
    });
}

export async function generateEmbeddings() {
    await batchEmbeddingsChat();
    await batchEmbeddingsIssue();
    await batchEmbeddingsMeeting();
    await batchEmbeddingsReminder();
    await batchEmbeddingsResourceVersion();
    await batchEmbeddingsRun();
    await batchEmbeddingsTag();
    await batchEmbeddingsTeam();
    await batchEmbeddingsUser();
}
