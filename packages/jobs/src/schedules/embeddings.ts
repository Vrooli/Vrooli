/**
 * Calculates and stores embeddings for all searchable objects (that don't have one yet), including 
 * root objects, versions, and tags. Objects that don't have embeddings 
 * include labels, comments, and focus modes.
 * 
 * Embeddings are used to find similar objects. For example, if you are 
 * looking for tags and type "ai", then the search should return tags 
 * like "Artificial Intelligence (AI)", "LLM", and "Machine Learning", even if "ai" 
 * is not directly in the tag's name/description.
 */
import { EmbeddableType, FindManyArgs, GenericModelLogic, ModelMap, batch, getEmbeddings, logger, prismaInstance } from "@local/server";
import { GqlModelType, RunStatus } from "@local/shared";
import { Prisma } from "@prisma/client";

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
    const tableName = ModelMap.get(isTranslatable ? (objectType + "Translation" as GqlModelType) : objectType).dbTable;
    const embeddingsText = `ARRAY[${embeddings.join(", ")}]`;
    // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
    await prismaInstance.$executeRawUnsafe(`UPDATE ${tableName} SET "embedding" = ${embeddingsText}, "embeddingNeedsUpdate" = false WHERE id = $1::UUID;`, id);
}

/**
 * Processes a batch of translatable objects and updates each stale translation embedding.
 * @param batch - The batch of objects to process.
 * @param instruction - The instruction used to generate the embeddings.
 * @param tableName - The name of the table where the embeddings should be updated. e.g. "api_version_translation"
 */
async function processTranslatedBatchHelper(
    batch: { translations: { id: string, embeddingNeedsUpdate: boolean, language: string }[] }[],
    objectType: EmbeddableType | `${EmbeddableType}`,
): Promise<void> {
    if (!ModelMap.isModel(objectType)) return;
    const model = ModelMap.get(objectType);
    // Extract sentences from the batch
    const sentences = extractTranslatedSentences(batch, model);
    if (sentences.length === 0) return;
    // Find embeddings for all versions in the batch
    const embeddings = await getEmbeddings(objectType, sentences);
    // Update the embeddings for each stale translation
    await Promise.all(batch.map(async (curr, index) => {
        const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
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
    const embeddings = await getEmbeddings(objectType, sentences);
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

async function batchEmbeddingsApiVersion() {
    embeddingBatch<Prisma.api_versionFindManyArgs>({
        objectType: "ApiVersion",
        processBatch: processTranslatedBatchHelper,
        trace: "0469",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsChat() {
    embeddingBatch<Prisma.chatFindManyArgs>({
        objectType: "Chat",
        processBatch: processTranslatedBatchHelper,
        trace: "0475",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsIssue() {
    embeddingBatch<Prisma.issueFindManyArgs>({
        objectType: "Issue",
        processBatch: processTranslatedBatchHelper,
        trace: "0476",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsMeeting() {
    embeddingBatch<Prisma.meetingFindManyArgs>({
        objectType: "Meeting",
        processBatch: processTranslatedBatchHelper,
        trace: "0477",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsNoteVersion() {
    embeddingBatch<Prisma.note_versionFindManyArgs>({
        objectType: "NoteVersion",
        processBatch: processTranslatedBatchHelper,
        trace: "0470",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsTeam() {
    embeddingBatch<Prisma.teamFindManyArgs>({
        objectType: "Team",
        processBatch: processTranslatedBatchHelper,
        trace: "0478",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsPost() {
    embeddingBatch<Prisma.postFindManyArgs>({
        objectType: "Post",
        processBatch: processTranslatedBatchHelper,
        trace: "0479",
        where: {
            isDeleted: false,
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsProjectVersion() {
    embeddingBatch<Prisma.project_versionFindManyArgs>({
        objectType: "ProjectVersion",
        processBatch: processTranslatedBatchHelper,
        trace: "0471",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsQuestion() {
    embeddingBatch<Prisma.questionFindManyArgs>({
        objectType: "Question",
        processBatch: processTranslatedBatchHelper,
        trace: "0480",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsQuiz() {
    embeddingBatch<Prisma.quizFindManyArgs>({
        objectType: "Quiz",
        processBatch: processTranslatedBatchHelper,
        trace: "0481",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsReminder() {
    embeddingBatch<Prisma.reminderFindManyArgs>({
        objectType: "Reminder",
        processBatch: processUntranslatedBatchHelper,
        trace: "0482",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingNeedsUpdate: true }),
        },
    });
}

async function batchEmbeddingsRoutineVersion() {
    embeddingBatch<Prisma.routine_versionFindManyArgs>({
        objectType: "RoutineVersion",
        processBatch: processTranslatedBatchHelper,
        trace: "0472",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsRunProject() {
    embeddingBatch<Prisma.run_projectFindManyArgs>({
        objectType: "RunProject",
        processBatch: processUntranslatedBatchHelper,
        trace: "0483",
        // Only update runs which are still active
        where: {
            status: { in: [RunStatus.Scheduled, RunStatus.InProgress] },
            schedule: { endTime: { gte: new Date().toISOString() } },
            ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingNeedsUpdate: true }),
        },
    });
}

async function batchEmbeddingsRunRoutine() {
    embeddingBatch<Prisma.run_routineFindManyArgs>({
        objectType: "RunRoutine",
        processBatch: processUntranslatedBatchHelper,
        trace: "0484",
        // Only update runs which are still active
        where: {
            status: { in: [RunStatus.Scheduled, RunStatus.InProgress] },
            schedule: { endTime: { gte: new Date().toISOString() } },
            ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingNeedsUpdate: true }),
        },
    });
}

async function batchEmbeddingsCodeVersion() {
    embeddingBatch<Prisma.code_versionFindManyArgs>({
        objectType: "CodeVersion",
        processBatch: processTranslatedBatchHelper,
        trace: "0473",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsStandardVersion() {
    embeddingBatch<Prisma.standard_versionFindManyArgs>({
        objectType: "StandardVersion",
        processBatch: processTranslatedBatchHelper,
        trace: "0474",
        where: {
            isDeleted: false,
            root: { isDeleted: false },
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsTag() {
    embeddingBatch<Prisma.tagFindManyArgs>({
        objectType: "Tag",
        processBatch: processTranslatedBatchHelper,
        trace: "0485",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

async function batchEmbeddingsUser() {
    embeddingBatch<Prisma.userFindManyArgs>({
        objectType: "User",
        processBatch: processTranslatedBatchHelper,
        trace: "0486",
        where: {
            ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
        },
    });
}

export async function generateEmbeddings() {
    await batchEmbeddingsApiVersion();
    await batchEmbeddingsChat();
    await batchEmbeddingsCodeVersion();
    await batchEmbeddingsIssue();
    await batchEmbeddingsMeeting();
    await batchEmbeddingsNoteVersion();
    await batchEmbeddingsPost();
    await batchEmbeddingsProjectVersion();
    await batchEmbeddingsQuestion();
    await batchEmbeddingsQuiz();
    await batchEmbeddingsReminder();
    await batchEmbeddingsRoutineVersion();
    await batchEmbeddingsRunProject();
    await batchEmbeddingsRunRoutine();
    await batchEmbeddingsStandardVersion();
    await batchEmbeddingsTag();
    await batchEmbeddingsTeam();
    await batchEmbeddingsUser();
}
