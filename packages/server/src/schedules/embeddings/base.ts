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
import { ValueOf } from "@local/shared";
import { Prisma, RunStatus } from "@prisma/client";
import { ObjectMap } from "../../models/base";
import { ModelLogic } from "../../models/types";
import { PrismaType } from "../../types";
import { FindManyArgs } from "../../utils";
import { batch, BatchProps } from "../../utils/batch";
import { EmbeddableType, EmbeddingTables, getEmbeddings } from "../../utils/embeddings/getEmbeddings";
import { cronTimes } from "../cronTimes";
import { initializeCronJob } from "../initializeCronJob";

// WARNING: Setting this to true will cause the embeddings to be recalculated for all objects. 
// This will take a long time and should only be done during development or if something is 
// wrong with the existing embeddings.
const RECALCULATE_EMBEDDINGS = true;

const API_BATCH_SIZE = 100; // Size set in the API to limit the number of embeddings generated at once

/**
 * Helper function to extract sentences from translated embeddable objects
 */
const extractTranslatedSentences = <T extends { translations: { language: string }[] }>(batch: T[], model: ValueOf<typeof ObjectMap>) => {
    // Initialize array to store sentences
    const sentences: string[] = [];
    // Loop through each object in the batch
    for (const curr of batch) {
        const currLanguages = curr.translations.map(translation => translation.language);
        const currSentences = currLanguages.map(language => model.display.embed?.get(curr as any, [language]));
        // Add each sentence to the array
        currSentences.forEach(sentence => {
            if (sentence) {
                sentences.push(sentence);
            }
        });
    }
    return sentences;
};

/**
 * Updates the embedding for a given object.
 * @param prisma - The Prisma client instance.
 * @param tableName - The name of the table where the embeddings should be updated.
 * @param id - The id of the object to update.
 * @param embeddings - The text embedding array returned from the API.
 */
const updateEmbedding = async (
    prisma: PrismaType,
    objectType: EmbeddableType | `${EmbeddableType}`,
    id: string,
    embeddings: number[],
): Promise<void> => {
    const tableName = EmbeddingTables[objectType];
    const embeddingsText = `ARRAY[${embeddings.join(", ")}]`;
    // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
    await prisma.$executeRawUnsafe(`UPDATE ${tableName} SET "embedding" = ${embeddingsText}, "embeddingNeedsUpdate" = false WHERE id = $1::UUID;`, id);
};

/**
 * Processes a batch of translatable objects and updates each stale translation embedding.
 * @param batch - The batch of objects to process.
 * @param prisma - The Prisma client instance.
 * @param instruction - The instruction used to generate the embeddings.
 * @param tableName - The name of the table where the embeddings should be updated. e.g. "api_version_translation"
 */
const processTranslatedBatchHelper = async (
    batch: { translations: { id: string, embeddingNeedsUpdate: boolean, language: string }[] }[],
    prisma: PrismaType,
    objectType: EmbeddableType | `${EmbeddableType}`,
): Promise<void> => {
    const model = ObjectMap[objectType];
    // Extract sentences from the batch
    const sentences = extractTranslatedSentences(batch, model);
    if (sentences.length === 0) return;
    // Find embeddings for all versions in the batch
    const { data: embeddings } = await getEmbeddings(objectType, sentences);
    // Update the embeddings for each stale translation
    await Promise.all(batch.map(async (curr, index) => {
        const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
        for (const translation of translationsToUpdate) {
            await updateEmbedding(prisma, objectType, translation.id, embeddings[index]);
        }
    }));
};


/**
 * Processes a batch of non-translatable objects and updates their embeddings.
 * @param batch - The batch of objects to process.
 * @param prisma - The Prisma client instance.
 * @param instruction - The instruction used to generate the embeddings.
 * @param tableName - The name of the table where the embeddings should be updated. e.g. "reminder"
 */
const processUntranslatedBatchHelper = async (
    batch: { id: string }[],
    prisma: PrismaType,
    objectType: EmbeddableType | `${EmbeddableType}`,
): Promise<void> => {
    const model = ObjectMap[objectType] as ModelLogic<any, any>;
    // Extract sentences from the batch
    const sentences = batch.map(obj => model.display.embed?.get(obj as any, []) ?? "");
    if (sentences.length === 0) return;
    // Find embeddings for all objects in the batch
    const { data: embeddings } = await getEmbeddings(objectType, sentences);
    // Update the embeddings for each object
    await Promise.all(
        batch.map(async (obj, index) => {
            await updateEmbedding(prisma, objectType, obj.id, embeddings[index]);
        }),
    );
};

const embeddingBatch = async <T extends FindManyArgs>({
    objectType,
    processBatch,
    trace,
    traceObject,
    ...props
}: Omit<BatchProps<T>, "batchSize" | "objectType" | "processBatch" | "select"> & {
    objectType: EmbeddableType | `${EmbeddableType}`,
    processBatch: (batch: any[], prisma: PrismaType, objectType: `${EmbeddableType}`) => Promise<void>,
}) => batch<T>({
    batchSize: API_BATCH_SIZE,
    objectType,
    processBatch: async (batch, prisma) => await processBatch(batch, prisma, objectType),
    select: ObjectMap[objectType]!.display.embed!.select(),
    trace,
    traceObject,
    ...props,
});

const batchEmbeddingsApiVersion = async () => embeddingBatch<Prisma.api_versionFindManyArgs>({
    objectType: "ApiVersion",
    processBatch: processTranslatedBatchHelper,
    trace: "0469",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsChat = async () => embeddingBatch<Prisma.chatFindManyArgs>({
    objectType: "Chat",
    processBatch: processTranslatedBatchHelper,
    trace: "0475",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsIssue = async () => embeddingBatch<Prisma.issueFindManyArgs>({
    objectType: "Issue",
    processBatch: processTranslatedBatchHelper,
    trace: "0476",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsMeeting = async () => embeddingBatch<Prisma.meetingFindManyArgs>({
    objectType: "Meeting",
    processBatch: processTranslatedBatchHelper,
    trace: "0477",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsNoteVersion = async () => embeddingBatch<Prisma.note_versionFindManyArgs>({
    objectType: "NoteVersion",
    processBatch: processTranslatedBatchHelper,
    trace: "0470",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsOrganization = async () => embeddingBatch<Prisma.organizationFindManyArgs>({
    objectType: "Organization",
    processBatch: processTranslatedBatchHelper,
    trace: "0478",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsPost = async () => embeddingBatch<Prisma.postFindManyArgs>({
    objectType: "Post",
    processBatch: processTranslatedBatchHelper,
    trace: "0479",
    where: {
        isDeleted: false,
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsProjectVersion = async () => embeddingBatch<Prisma.project_versionFindManyArgs>({
    objectType: "ProjectVersion",
    processBatch: processTranslatedBatchHelper,
    trace: "0471",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsQuestion = async () => embeddingBatch<Prisma.questionFindManyArgs>({
    objectType: "Question",
    processBatch: processTranslatedBatchHelper,
    trace: "0480",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsQuiz = async () => embeddingBatch<Prisma.quizFindManyArgs>({
    objectType: "Quiz",
    processBatch: processTranslatedBatchHelper,
    trace: "0481",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsReminder = async () => embeddingBatch<Prisma.reminderFindManyArgs>({
    objectType: "Reminder",
    processBatch: processUntranslatedBatchHelper,
    trace: "0482",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { embeddingNeedsUpdate: true }),
    },
});

const batchEmbeddingsRoutineVersion = async () => embeddingBatch<Prisma.routine_versionFindManyArgs>({
    objectType: "RoutineVersion",
    processBatch: processTranslatedBatchHelper,
    trace: "0472",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsRunProject = async () => embeddingBatch<Prisma.run_projectFindManyArgs>({
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

const batchEmbeddingsRunRoutine = async () => embeddingBatch<Prisma.run_routineFindManyArgs>({
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

const batchEmbeddingsSmartContractVersion = async () => embeddingBatch<Prisma.smart_contract_versionFindManyArgs>({
    objectType: "SmartContractVersion",
    processBatch: processTranslatedBatchHelper,
    trace: "0473",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsStandardVersion = async () => embeddingBatch<Prisma.standard_versionFindManyArgs>({
    objectType: "StandardVersion",
    processBatch: processTranslatedBatchHelper,
    trace: "0474",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsTag = async () => embeddingBatch<Prisma.tagFindManyArgs>({
    objectType: "Tag",
    processBatch: processTranslatedBatchHelper,
    trace: "0485",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const batchEmbeddingsUser = async () => embeddingBatch<Prisma.userFindManyArgs>({
    objectType: "User",
    processBatch: processTranslatedBatchHelper,
    trace: "0486",
    where: {
        ...(RECALCULATE_EMBEDDINGS ? {} : { translations: { some: { embeddingNeedsUpdate: true } } }),
    },
});

const generateEmbeddings = async () => {
    await batchEmbeddingsApiVersion();
    await batchEmbeddingsChat();
    await batchEmbeddingsIssue();
    await batchEmbeddingsMeeting();
    await batchEmbeddingsNoteVersion();
    await batchEmbeddingsOrganization();
    await batchEmbeddingsPost();
    await batchEmbeddingsProjectVersion();
    await batchEmbeddingsQuestion();
    await batchEmbeddingsQuiz();
    await batchEmbeddingsReminder();
    await batchEmbeddingsRoutineVersion();
    await batchEmbeddingsRunProject();
    await batchEmbeddingsRunRoutine();
    await batchEmbeddingsSmartContractVersion();
    await batchEmbeddingsStandardVersion();
    await batchEmbeddingsTag();
    await batchEmbeddingsUser();
};

/**
 * Initializes cron jobs for generating text embeddings
 */
export const initGenerateEmbeddingsCronJob = () => {
    initializeCronJob(cronTimes.embeddings, generateEmbeddings, "generate embeddings");
};
