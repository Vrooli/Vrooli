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
import { Prisma } from "@prisma/client";
import https from "https";
import { ApiVersionModel, NoteVersionModel, ProjectVersionModel, RoutineVersionModel, SmartContractVersionModel, StandardVersionModel } from "../../models";
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
    // Might not want to embed all runs. Maybe just ones that are scheduled?
    // Valyxa doesn't really have a need to look through your one-off runs, 
    // really just the recurring ones.
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
                const embeddings = JSON.parse(responseBody);
                resolve(embeddings);
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
const extractTranslatedSentences = <T extends { id: string, translations: { language: string }[] }>(batch: T[]) => {
    // Initialize array to store sentences
    const sentences: string[] = [];
    // Loop through each object in the batch
    for (const curr of batch) {
        const currLanguages = curr.translations.map(translation => translation.language);
        const currSentences = currLanguages.map(language => ApiVersionModel.display.embed?.get(curr as any, [language]));
        // Add each sentence to the array
        currSentences.forEach(sentence => {
            if (sentence) {
                sentences.push(sentence);
            }
        });
    }
    return sentences;
};

const batchEmbeddingsApiVersion = async () => batch<Prisma.api_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "ApiVersion",
    processBatch: async (batch, prisma) => {
        // Extract sentences and version ids from the batch
        const sentences = extractTranslatedSentences(batch);
        // Find embeddings for all versions in the batch
        const embeddings = await getEmbeddings(Instructions.Api, sentences);
        // Update the embeddings for each translation
        await Promise.all(batch.map(async (curr, index) => {
            const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
            for (const translation of translationsToUpdate) {
                // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
                await prisma.$executeRaw`UPDATE api_version_translation
                                         SET embedding = ${embeddings[index]}::vector,
                                             embeddingNeedsUpdate = false
                                         WHERE id = ${translation.id}`;
            }
        }));
    },
    select: ApiVersionModel.display.embed!.select(),
    trace: "0469",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsNoteVersion = async () => batch<Prisma.note_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "NoteVersion",
    processBatch: async (batch, prisma) => {
        // Extract sentences and version ids from the batch
        const sentences = extractTranslatedSentences(batch);
        // Find embeddings for all versions in the batch
        const embeddings = await getEmbeddings(Instructions.Note, sentences);
        // Update the embeddings for each translation
        await Promise.all(batch.map(async (curr, index) => {
            const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
            for (const translation of translationsToUpdate) {
                // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
                await prisma.$executeRaw`UPDATE note_version_translation
                                         SET embedding = ${embeddings[index]}::vector,
                                             embeddingNeedsUpdate = false
                                         WHERE id = ${translation.id}`;
            }
        }));
    },
    select: NoteVersionModel.display.embed!.select(),
    trace: "0470",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsProjectVersion = async () => batch<Prisma.project_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "ProjectVersion",
    processBatch: async (batch, prisma) => {
        // Extract sentences and version ids from the batch
        const sentences = extractTranslatedSentences(batch);
        // Find embeddings for all versions in the batch
        const embeddings = await getEmbeddings(Instructions.Project, sentences);
        // Update the embeddings for each translation
        await Promise.all(batch.map(async (curr, index) => {
            const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
            for (const translation of translationsToUpdate) {
                // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
                await prisma.$executeRaw`UPDATE project_version_translation
                                         SET embedding = ${embeddings[index]}::vector,
                                             embeddingNeedsUpdate = false
                                         WHERE id = ${translation.id}`;
            }
        }));
    },
    select: ProjectVersionModel.display.embed!.select(),
    trace: "0471",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsRoutineVersion = async () => batch<Prisma.routine_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "RoutineVersion",
    processBatch: async (batch, prisma) => {
        // Extract sentences and version ids from the batch
        const sentences = extractTranslatedSentences(batch);
        // Find embeddings for all versions in the batch
        const embeddings = await getEmbeddings(Instructions.Routine, sentences);
        // Update the embeddings for each translation
        await Promise.all(batch.map(async (curr, index) => {
            const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
            for (const translation of translationsToUpdate) {
                // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
                await prisma.$executeRaw`UPDATE routine_version_translation
                                         SET embedding = ${embeddings[index]}::vector,
                                             embeddingNeedsUpdate = false
                                         WHERE id = ${translation.id}`;
            }
        }));
    },
    select: RoutineVersionModel.display.embed!.select(),
    trace: "0472",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsSmartContractVersion = async () => batch<Prisma.smart_contract_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "SmartContractVersion",
    processBatch: async (batch, prisma) => {
        // Extract sentences and version ids from the batch
        const sentences = extractTranslatedSentences(batch);
        // Find embeddings for all versions in the batch
        const embeddings = await getEmbeddings(Instructions.SmartContract, sentences);
        // Update the embeddings for each translation
        await Promise.all(batch.map(async (curr, index) => {
            const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
            for (const translation of translationsToUpdate) {
                // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
                await prisma.$executeRaw`UPDATE smart_contract_version_translation
                                         SET embedding = ${embeddings[index]}::vector,
                                             embeddingNeedsUpdate = false
                                         WHERE id = ${translation.id}`;
            }
        }));
    },
    select: SmartContractVersionModel.display.embed!.select(),
    trace: "0473",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const batchEmbeddingsStandardVersion = async () => batch<Prisma.standard_versionFindManyArgs>({
    batchSize: API_BATCH_SIZE,
    objectType: "StandardVersion",
    processBatch: async (batch, prisma) => {
        // Extract sentences and version ids from the batch
        const sentences = extractTranslatedSentences(batch);
        // Find embeddings for all versions in the batch
        const embeddings = await getEmbeddings(Instructions.Standard, sentences);
        // Update the embeddings for each translation
        await Promise.all(batch.map(async (curr, index) => {
            const translationsToUpdate = curr.translations.filter(t => t.embeddingNeedsUpdate);
            for (const translation of translationsToUpdate) {
                // Use raw query to update the embedding, because the Prisma client doesn't support Postgres vectors
                await prisma.$executeRaw`UPDATE standard_version_translation
                                         SET embedding = ${embeddings[index]}::vector,
                                             embeddingNeedsUpdate = false
                                         WHERE id = ${translation.id}`;
            }
        }));
    },
    select: StandardVersionModel.display.embed!.select(),
    trace: "0474",
    where: { translations: { some: { embeddingNeedsUpdate: true } } },
});

const generateEmbeddings = async () => {
    await batchEmbeddingsApiVersion();
    await batchEmbeddingsNoteVersion();
    await batchEmbeddingsProjectVersion();
    await batchEmbeddingsRoutineVersion();
    await batchEmbeddingsSmartContractVersion();
    await batchEmbeddingsStandardVersion();
};

/**
 * Initializes cron jobs for generating text embeddings
 */
export const initGenerateEmbeddingsCronJob = () => {
    initializeCronJob(cronTimes.embeddings, generateEmbeddings, "generate embeddings");
};
