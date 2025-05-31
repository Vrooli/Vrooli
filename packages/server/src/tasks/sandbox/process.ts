import { CodeLanguage, CodeVersionConfig, type ResourceVersion } from "@local/shared";
import { type Job } from "bullmq";
import { readOneHelper } from "../../actions/reads.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../../validators/permissions.js";
import { type SandboxRequestPayload } from "./queue.js";
import { SandboxChildProcessManager } from "./sandboxWorkerManager.js";
import { type RunUserCodeInput, type RunUserCodeOutput, type SandboxProcessPayload } from "./types.js";

const codeVersionSelect = {
    id: true,
    config: true,
    codeLanguage: true,
} as const;

// Define the expected structure for the cached object
interface CachedCodeObject {
    id: string;
    content: string;
    codeLanguage: CodeLanguage;
}

// Create a new child process manager to run the user code
const manager = new SandboxChildProcessManager();

/**
 * Runs sandboxed user code in a secure environment, 
 * when you already know the code content and validated the user's permissions.
 */
export async function runUserCode({
    code,
    codeLanguage,
    input,
}: RunUserCodeInput): Promise<RunUserCodeOutput> {
    // Run the user code with the provided input
    return await manager.runUserCode({
        code,
        codeLanguage: codeLanguage as CodeLanguage,
        input,
    });
}

/**
 * Runs sandboxed user code in a secure environment. 
 * Fetches code from a cache if available, otherwise reads from the database.
 * Ensures that the user has the necessary permissions to access the code.
 */
export async function doSandbox({
    codeVersionId,
    input,
    userData,
}: SandboxProcessPayload): Promise<RunUserCodeOutput> {
    try {
        let result: RunUserCodeOutput | undefined = undefined;

        const cacheService = CacheService.get();
        const redisKey = `codeVersion:${codeVersionId}`;

        let codeObject: CachedCodeObject | null = await cacheService.get<CachedCodeObject>(redisKey);

        if (codeObject) {
            // Code found in cache, perform permission checks
            const authDataById = await getAuthenticatedData({ "ResourceVersion": [codeVersionId] }, userData);
            if (Object.keys(authDataById).length === 0) {
                throw new CustomError("0620", "NotFound", { message: "Authentication data not found for code version.", codeVersionId });
            }
            await permissionsCheck(authDataById, { ["Read"]: [codeVersionId] }, {}, userData);
        } else {
            // Code not in cache, fetch from DB
            const req = { session: { languages: userData.languages, users: [userData] } };
            const dbResult = await readOneHelper<ResourceVersion>({
                info: codeVersionSelect,
                input: { id: codeVersionId },
                objectType: "ResourceVersion",
                req,
            });

            if (!dbResult || !dbResult.id) {
                logger.error("Code version not found in DB or permission denied, and readOneHelper did not throw.", { trace: "doSandbox-dblookup-fail", codeVersionId });
                throw new CustomError("0624", "NotFound", { reason: "Code version not found in DB or permission denied.", codeVersionId });
            }

            // Deserialize config to get the actual code content
            const parsedConfig = CodeVersionConfig.parse(
                // Provide the expected Pick<ResourceVersion, "codeLanguage" | "config"> object
                { config: dbResult.config, codeLanguage: dbResult.codeLanguage },
                logger, // Pass the logger instance
            );

            if (!parsedConfig || typeof parsedConfig.content !== "string") {
                logger.error("Failed to parse code config or content is missing/invalid", { trace: "doSandbox-config-deserialize-fail", codeVersionId });
                throw new CustomError("0627", "InternalError", { reason: "Failed to process code configuration.", codeVersionId });
            }

            codeObject = {
                id: dbResult.id.toString(),
                content: parsedConfig.content,
                codeLanguage: parsedConfig.codeLanguage as CodeLanguage, // Ensure this is CodeLanguage enum
            };

            await cacheService.set(redisKey, codeObject);
        }

        if (!codeObject) {
            logger.error("Code object is null after cache and DB checks", { trace: "doSandbox-finalnullcheck", codeVersionId });
            throw new CustomError("0628", "InternalError", { reason: "Code object could not be loaded." });
        }

        const code = codeObject.content;

        if (!codeObject.codeLanguage || !Object.values(CodeLanguage).includes(codeObject.codeLanguage as CodeLanguage)) {
            logger.error("Invalid code language in code object", { trace: "doSandbox-invalidlangobj", codeVersionId, codeLanguage: codeObject.codeLanguage });
            throw new CustomError("0626", "InternalError", { reason: "Invalid code language specified in code object", codeVersionId });
        }
        const codeLang = codeObject.codeLanguage as CodeLanguage;

        result = await runUserCode({ codeLanguage: codeLang, code, input });

        if (!result) {
            throw new CustomError("0622", "InternalError", { process: "doSandbox", reason: "runUserCode returned undefined" });
        }
        return result;

    } catch (error) {
        if (error instanceof CustomError) {
            logger.warn(`CustomError in doSandbox: ${error.message}`, {
                trace: "doSandbox-CustomError",
                codeVersionId,
                errorCode: error.code,
                errorName: error.name,
                errorContext: (error as any).context ?? (error as any).payload,
            });
            throw error; // Rethrow CustomError to be handled by higher-level handlers
        }

        // For non-CustomErrors, log and return a generic error response
        logger.error("Unexpected error in doSandbox", { trace: "0554", codeVersionId, error });
        return { __type: "error", error: "An internal error occurred while processing the sandbox request." };
    }
}

export async function sandboxProcess({ data }: Job<SandboxRequestPayload>) {
    switch (data.__process) {
        case "Sandbox":
            return doSandbox(data);
        case "Test":
            logger.info("sandboxProcess test triggered");
            return { __typename: "Success" as const, success: true };
        default:
            throw new CustomError("0608", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}
