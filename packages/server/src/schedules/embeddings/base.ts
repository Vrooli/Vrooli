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
import { Prisma, RunStatus } from "@prisma/client";
import https from "https";
import { ApiVersionModel, ChatModel, IssueModel, MeetingModel, NoteVersionModel, OrganizationModel, PostModel, ProjectVersionModel, QuestionModel, QuizModel, ReminderModel, RoutineVersionModel, RunProjectModel, RunRoutineModel, SmartContractVersionModel, StandardVersionModel, TagModel, UserModel } from "../../models";
import { ModelLogic } from "../../models/types";
import { PrismaType } from "../../types";
import { batch } from "../../utils/batch";
import { cronTimes } from "../cronTimes";
import { initializeCronJob } from "../initializeCronJob";

const API_BATCH_SIZE = 100; // Size set in the API to limit the number of embeddings generated at once

// The model used for embedding (instructor-base) requires an instruction 
// to create embeddings. This acts as a way to fine-tune the model for 
// a specific task. Most of our objects may end up using the same 
// instruction, but it's possible that some may need a different one. 
// See https://github.com/Vrooli/text-embedder-tests for more details.
const INSTRUCTION_COMMON = "Represent the text for classification";
const Instructions = {
    "Api": INSTRUCTION_COMMON,
    "ApiVersion": INSTRUCTION_COMMON,
    "Chat": INSTRUCTION_COMMON,
    "Issue": INSTRUCTION_COMMON,
    "Meeting": INSTRUCTION_COMMON,
    "Note": INSTRUCTION_COMMON,
    "NoteVersion": INSTRUCTION_COMMON,
    "Organization": INSTRUCTION_COMMON,
    "Post": INSTRUCTION_COMMON,
    "Project": INSTRUCTION_COMMON,
    "ProjectVersion": INSTRUCTION_COMMON,
    "Question": INSTRUCTION_COMMON,
    "Quiz": INSTRUCTION_COMMON,
    "Reminder": INSTRUCTION_COMMON,
    "Routine": INSTRUCTION_COMMON,
    "RoutineVersion": INSTRUCTION_COMMON,
    "RunProject": INSTRUCTION_COMMON,
    "RunRoutine": INSTRUCTION_COMMON,
    "SmartContract": INSTRUCTION_COMMON,
    "SmartContractVersion": INSTRUCTION_COMMON,
    "Standard": INSTRUCTION_COMMON,
    "StandardVersion": INSTRUCTION_COMMON,
    "Tag": INSTRUCTION_COMMON,
    "User": INSTRUCTION_COMMON,
};

/**
 * Function to get embeddings from your new API
 * @param instruction The instruction for what to generate embeddings for
 * @param sentences The sentences to generate embeddings for. Each sentence 
 * can be plaintext (if only embedding one field) or stringified JSON (for multiple fields)
 * @returns A Promise that resolves with the embeddings, in the same order as the sentences
 * @throws An Error if the API request fails
 */
async function getEmbeddings(instruction: string, sentences: string[]): Promise<any> {
    return new Promise((resolve, reject) => {
        const data = JSON.stringify({ instruction, sentences });
        const options = {
            hostname: "embedtext.com",
            port: 443,
            path: "/",
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Content-Length": data.length,
            },
        };
        const apiRequest = https.request(options, apiRes => {
            let responseBody = "";
            apiRes.on("data", chunk => {
                responseBody += chunk;
            });
            apiRes.on("end", () => {
                const result = JSON.parse(responseBody);
                resolve(result.embeddings);
            });
        });
        apiRequest.on("error", error => {
            console.error(`Error: ${error}`);
            reject(error);
        });
        apiRequest.write(data);
        apiRequest.end();
    });
}

/**
 * Helper function to extract sentences from translated embeddable objects
 */
const extractTranslatedSentences = <T extends { id: string, translations: { language: string }[] }>(batch: T[], model: ModelLogic<any, any>) => {
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
    tableName: string,
    id: string,
    embeddings: number[],
): Promise<void> => {
    const embeddingsText = `ARRAY[${embeddings.join(", ")}]`;//`${JSON.stringify(embeddings)}::vector`;
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
    batch: { id: string, translations: { id: string, embeddingNeedsUpdate: boolean, language: string }[] }[],
    prisma: PrismaType,
    model: ModelLogic<any, any>,
    instruction: string,
    tableName: string,
): Promise<void> => {
    // Extract sentences from the batch
    const sentences = extractTranslatedSentences(batch, model);
    if (sentences.length === 0) return;
    // Find embeddings for all versions in the batch
    const embeddings = await getEmbeddings(instruction, sentences);
    // Update the embeddings for each stale translation
    await Promise.all(batch.map(async (curr, index) => {
        const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
        for (const translation of translationsToUpdate) {
            await updateEmbedding(prisma, tableName, translation.id, embeddings[index]);
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
    model: ModelLogic<any, any>,
    instruction: string,
    tableName: string,
): Promise<void> => {
    // Extract sentences from the batch
    const sentences = batch.map(obj => model.display.embed!.get(obj as any, []));
    if (sentences.length === 0) return;
    // Find embeddings for all objects in the batch
    const embeddings = await getEmbeddings(instruction, sentences);
    // Update the embeddings for each object
    await Promise.all(
        batch.map(async (obj, index) => {
            await updateEmbedding(prisma, tableName, obj.id, embeddings[index]);
        }),
    );
};

const batchEmbeddingsApiVersion = async () => batch<Prisma.api_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "ApiVersion",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, ApiVersionModel, Instructions.Api, "api_version_translation");
    },
    select: ApiVersionModel.display.embed!.select(),
    trace: "0469",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        translations: { some: { embeddingNeedsUpdate: true } },
    },
});

const batchEmbeddingsChat = async () => batch<Prisma.chatFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Chat",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, ChatModel, Instructions.Chat, "chat_translation");
    },
    select: ChatModel.display.embed!.select(),
    trace: "0475",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsIssue = async () => batch<Prisma.issueFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Issue",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, IssueModel, Instructions.Issue, "issue_translation");
    },
    select: IssueModel.display.embed!.select(),
    trace: "0476",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsMeeting = async () => batch<Prisma.meetingFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Meeting",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, MeetingModel, Instructions.Meeting, "meeting_translation");
    },
    select: MeetingModel.display.embed!.select(),
    trace: "0477",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsNoteVersion = async () => batch<Prisma.note_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "NoteVersion",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, NoteVersionModel, Instructions.Note, "note_version_translation");
    },
    select: NoteVersionModel.display.embed!.select(),
    trace: "0470",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        translations: { some: { embeddingNeedsUpdate: true } },
    },
});

const batchEmbeddingsOrganization = async () => batch<Prisma.organizationFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Organization",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, OrganizationModel, Instructions.Organization, "organization_translation");
    },
    select: OrganizationModel.display.embed!.select(),
    trace: "0478",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsPost = async () => batch<Prisma.postFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Post",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, PostModel, Instructions.Post, "post_translation");
    },
    select: PostModel.display.embed!.select(),
    trace: "0479",
    where: {
        isDeleted: false,
        translations: { some: { embeddingNeedsUpdate: true } },
    },
});

const batchEmbeddingsProjectVersion = async () => batch<Prisma.project_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "ProjectVersion",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, ProjectVersionModel, Instructions.Project, "project_version_translation");
    },
    select: ProjectVersionModel.display.embed!.select(),
    trace: "0471",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        translations: { some: { embeddingNeedsUpdate: true } },
    },
});

const batchEmbeddingsQuestion = async () => batch<Prisma.questionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Question",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, QuestionModel, Instructions.Question, "question_translation");
    },
    select: QuestionModel.display.embed!.select(),
    trace: "0480",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsQuiz = async () => batch<Prisma.quizFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Quiz",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, QuizModel, Instructions.Quiz, "quiz_translation");
    },
    select: QuizModel.display.embed!.select(),
    trace: "0481",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsReminder = async () => batch<Prisma.reminderFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Reminder",
    processBatch: async (batch, prisma) => {
        await processUntranslatedBatchHelper(batch, prisma, ReminderModel, Instructions.Reminder, "reminder");
    },
    select: ReminderModel.display.embed!.select(),
    trace: "0482",
    where: { embeddingNeedsUpdate: true },
});

const batchEmbeddingsRoutineVersion = async () => batch<Prisma.routine_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "RoutineVersion",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, RoutineVersionModel, Instructions.Routine, "routine_version_translation");
    },
    select: RoutineVersionModel.display.embed!.select(),
    trace: "0472",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        translations: { some: { embeddingNeedsUpdate: true } },
    },
});

const batchEmbeddingsRunProject = async () => batch<Prisma.run_projectFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "RunProject",
    processBatch: async (batch, prisma) => {
        await processUntranslatedBatchHelper(batch, prisma, RunProjectModel, Instructions.RunProject, "run_project");
    },
    select: RunProjectModel.display.embed!.select(),
    trace: "0483",
    // Only update runs which are still active
    where: {
        embeddingNeedsUpdate: true,
        status: { in: [RunStatus.Scheduled, RunStatus.InProgress] },
        schedule: { endTime: { gte: new Date().toISOString() } },
    },
});

const batchEmbeddingsRunRoutine = async () => batch<Prisma.run_routineFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "RunRoutine",
    processBatch: async (batch, prisma) => {
        await processUntranslatedBatchHelper(batch, prisma, RunRoutineModel, Instructions.RunRoutine, "run_routine");
    },
    select: RunRoutineModel.display.embed!.select(),
    trace: "0484",
    // Only update runs which are still active
    where: {
        embeddingNeedsUpdate: true,
        status: { in: [RunStatus.Scheduled, RunStatus.InProgress] },
        schedule: { endTime: { gte: new Date().toISOString() } },
    },
});

const batchEmbeddingsSmartContractVersion = async () => batch<Prisma.smart_contract_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "SmartContractVersion",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, SmartContractVersionModel, Instructions.SmartContract, "smart_contract_version_translation");
    },
    select: SmartContractVersionModel.display.embed!.select(),
    trace: "0473",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        translations: { some: { embeddingNeedsUpdate: true } },
    },
});

const batchEmbeddingsStandardVersion = async () => batch<Prisma.standard_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "StandardVersion",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, StandardVersionModel, Instructions.Standard, "standard_version_translation");
    },
    select: StandardVersionModel.display.embed!.select(),
    trace: "0474",
    where: {
        isDeleted: false,
        root: { isDeleted: false },
        translations: { some: { embeddingNeedsUpdate: true } },
    },
});

const batchEmbeddingsTag = async () => batch<Prisma.tagFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "Tag",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, TagModel, Instructions.Tag, "tag_translation");
    },
    select: TagModel.display.embed!.select(),
    trace: "0485",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsUser = async () => batch<Prisma.userFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "User",
    processBatch: async (batch, prisma) => {
        await processTranslatedBatchHelper(batch, prisma, UserModel, Instructions.User, "user_translation");
    },
    select: UserModel.display.embed!.select(),
    trace: "0486",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
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
