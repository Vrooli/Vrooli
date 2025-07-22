import { CodeLanguage, CodeVersionConfig, type ExecutionEnvironment, type ResourceVersion } from "@vrooli/shared";
import { type Job } from "bullmq";
import { readOneHelper } from "../../actions/reads.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { validateLocalExecutionSafety } from "../../utils/executionMode.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../../validators/permissions.js";
import { type SandboxTask } from "../taskTypes.js";
import { getSupportedExecutionEnvironments } from "./executionEnvironmentSupport.js";
import { getLocalExecutionManager } from "./localExecutionManager.js";
import { SandboxChildProcessManager } from "./sandboxWorkerManager.js";
import { type RunUserCodeInput, type RunUserCodeOutput } from "./types.js";

// Type for the sandbox process payload - picks the essential fields from SandboxTask
type SandboxProcessPayload = Pick<SandboxTask, "codeVersionId" | "input" | "userData">;

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
    executionEnvironment?: ExecutionEnvironment;
    environmentConfig?: {
        workingDirectory?: string;
        allowedPaths?: string[];
        timeoutMs?: number;
        memoryLimitMb?: number;
        environmentVariables?: Record<string, string>;
    };
}

// Create a new child process manager to run the user code in sandbox
const sandboxManager = new SandboxChildProcessManager();

/**
 * Runs user code in the appropriate execution environment,
 * when you already know the code content and validated the user's permissions.
 */
export async function runUserCode({
    code,
    codeLanguage,
    input,
    executionEnvironment = "sandbox",
    environmentConfig,
}: RunUserCodeInput): Promise<RunUserCodeOutput> {
    // Route execution based on environment
    switch (executionEnvironment) {
        case "sandbox":
            // Use the existing sandbox manager for isolated JavaScript execution
            return await sandboxManager.runUserCode({
                code,
                codeLanguage: codeLanguage as CodeLanguage,
                input,
                executionEnvironment,
                environmentConfig,
            });

        case "local": {
            // Additional safety validation for local execution
            try {
                validateLocalExecutionSafety();
            } catch (error) {
                logger.warn("Local execution safety check failed", {
                    error: error instanceof Error ? error.message : String(error),
                    trace: "runUserCode-local-safety-fail",
                });

                // Provide configuration-aware error message
                const errorMessage = error instanceof CustomError ? error.message :
                    (error instanceof Error ? error.message : "Local execution safety check failed");

                return {
                    __type: "error",
                    error: errorMessage,
                };
            }

            // Use the local execution manager for development/testing
            const localManager = getLocalExecutionManager();
            return await localManager.runUserCode({
                code,
                codeLanguage: codeLanguage as CodeLanguage,
                input,
                executionEnvironment,
                environmentConfig,
            });
        }

        default: {
            const supportedEnvs = getSupportedExecutionEnvironments().join(", ");
            return {
                __type: "error",
                error: `Unsupported execution environment '${executionEnvironment}'. Supported environments: ${supportedEnvs}`,
            };
        }
    }
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
                {
                    config: dbResult.config as ResourceVersion["config"],
                    codeLanguage: dbResult.codeLanguage as ResourceVersion["codeLanguage"],
                },
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
                executionEnvironment: parsedConfig.executionEnvironment,
                environmentConfig: parsedConfig.environmentConfig,
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

        result = await runUserCode({
            codeLanguage: codeLang,
            code,
            input,
            executionEnvironment: codeObject.executionEnvironment,
            environmentConfig: codeObject.environmentConfig,
        });

        if (!result) {
            throw new CustomError("0622", "InternalError", { process: "doSandbox", reason: "runUserCode returned undefined" });
        }
        return result;

    } catch (error: unknown) {
        if (error instanceof CustomError) {
            logger.warn(`CustomError in doSandbox: ${error.message}`, {
                trace: "doSandbox-CustomError",
                codeVersionId,
                errorCode: error.code,
                errorName: error.name,
                errorContext: "context" in error ? error.context : ("payload" in error ? (error as { payload: unknown }).payload : undefined),
            });
            throw error; // Rethrow CustomError to be handled by higher-level handlers
        }

        // For non-CustomErrors, log and return a generic error response
        logger.error("Unexpected error in doSandbox", { trace: "0554", codeVersionId, error });
        return { __type: "error", error: "An internal error occurred while processing the sandbox request." };
    }
}

export async function sandboxProcess({ data }: Job<SandboxTask>) {
    // For backward compatibility, check if __process is provided
    const processType = ("__process" in data && typeof data.__process === "string") ? data.__process : "Sandbox";

    switch (processType) {
        case "Sandbox":
            return doSandbox(data);
        case "Test":
            logger.info("sandboxProcess test triggered");
            return { __typename: "Success" as const, success: true };
        default:
            throw new CustomError("0608", "InternalError", { process: processType });
    }
}
