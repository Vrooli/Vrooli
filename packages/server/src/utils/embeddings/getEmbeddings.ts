import https from "https";
import { logger } from "../../events";

export type EmbeddableType = "ApiVersion" | "Chat" | "Issue" | "Meeting" | "NoteVersion" | "Organization" | "Post" | "ProjectVersion" | "Question" | "Quiz" | "Reminder" | "RoutineVersion" | "RunProject" | "RunRoutine" | "SmartContractVersion" | "StandardVersion" | "Tag" | "User";

// The model used for embedding (instructor-base) requires an instruction 
// to create embeddings. This acts as a way to fine-tune the model for 
// a specific task. Most of our objects may end up using the same 
// instruction, but it's possible that some may need a different one. 
// See https://github.com/Vrooli/text-embedder-tests for more details.
const INSTRUCTION_COMMON = "Embed this text";
const Instructions: { [key in EmbeddableType]: string } = {
    "ApiVersion": INSTRUCTION_COMMON,
    "Chat": INSTRUCTION_COMMON,
    "Issue": INSTRUCTION_COMMON,
    "Meeting": INSTRUCTION_COMMON,
    "NoteVersion": INSTRUCTION_COMMON,
    "Organization": INSTRUCTION_COMMON,
    "Post": INSTRUCTION_COMMON,
    "ProjectVersion": INSTRUCTION_COMMON,
    "Question": INSTRUCTION_COMMON,
    "Quiz": INSTRUCTION_COMMON,
    "Reminder": INSTRUCTION_COMMON,
    "RoutineVersion": INSTRUCTION_COMMON,
    "RunProject": INSTRUCTION_COMMON,
    "RunRoutine": INSTRUCTION_COMMON,
    "SmartContractVersion": INSTRUCTION_COMMON,
    "StandardVersion": INSTRUCTION_COMMON,
    "Tag": INSTRUCTION_COMMON,
    "User": INSTRUCTION_COMMON,
};

// Depending on the object type, the table containing the embeddings may be different.
export const EmbeddingTables: { [key in EmbeddableType]: string } = {
    "ApiVersion": "api_version_translation",
    "Chat": "chat_translation",
    "Issue": "issue_translation",
    "Meeting": "meeting_translation",
    "NoteVersion": "note_version_translation",
    "Organization": "organization_translation",
    "Post": "post_translation",
    "ProjectVersion": "project_version_translation",
    "Question": "question_translation",
    "Quiz": "quiz_translation",
    "Reminder": "reminder",
    "RoutineVersion": "routine_version_translation",
    "RunProject": "run_project",
    "RunRoutine": "run_routine",
    "SmartContractVersion": "smart_contract_version_translation",
    "StandardVersion": "standard_version_translation",
    "Tag": "tag_translation",
    "User": "user_translation",
};

/**
 * Function to get embeddings from a https://github.com/Vrooli/text-embedder-endpoint API
 * @param objectType The type of object to generate embeddings for
 * @param sentences The sentences to generate embeddings for. Each sentence 
 * can be plaintext (if only embedding one field) or stringified JSON (for multiple fields)
 * @returns A Promise that resolves with the embeddings, in the same order as the sentences
 * @throws An Error if the API request fails
 */
export async function getEmbeddings(objectType: EmbeddableType | `${EmbeddableType}`, sentences: string[]): Promise<number[][]> {
    try {
        const instruction = Instructions[objectType];
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
                    // TODO ensure result is of expected type, or log error. Make sure to handle errors where this function is used
                    // Check type of result
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
    } catch (error) {
        logger.error("Error fetching embeddings", { trace: "0084", error });
        return [];
    }
}
