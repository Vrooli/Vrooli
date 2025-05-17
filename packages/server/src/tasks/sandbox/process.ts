import { CodeLanguage, CodeVersion, HOURS_1_S } from "@local/shared";
import { Job } from "bullmq";
import { readOneHelper } from "../../actions/reads.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { withRedis } from "../../redisConn.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../../validators/permissions.js";
import { SandboxRequestPayload } from "./queue.js";
import { SandboxChildProcessManager } from "./sandboxWorkerManager.js";
import { RunUserCodeInput, RunUserCodeOutput, SandboxProcessPayload } from "./types.js";

/** How long to cache the code in Redis */
const CACHE_TTL_SECONDS = HOURS_1_S;

const codeVersionSelect = {
    id: true,
    content: true,
} as const;

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
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;

                // Check Redis cache for the code version
                const redisKey = `codeVersion:${codeVersionId}`;
                const cachedCode = await redisClient.get(redisKey);

                let codeObject;
                if (cachedCode) {
                    // Query for all authentication data
                    const authDataById = await getAuthenticatedData({ "CodeVersion": [codeVersionId] }, userData);
                    if (Object.keys(authDataById).length === 0) {
                        throw new CustomError("0620", "NotFound", { codeVersionId });
                    }
                    // Check permissions
                    await permissionsCheck(authDataById, { ["Read"]: [codeVersionId] }, {}, userData);
                    // Use the cached code
                    codeObject = JSON.parse(cachedCode);
                } else {
                    // Read the user code from the database using `readOneHelper`. This function checks permissions, so we don't need to do it again.
                    const req = { session: { languages: userData.languages, users: [userData] } };
                    codeObject = await readOneHelper<CodeVersion>({
                        info: codeVersionSelect,
                        input: { id: codeVersionId },
                        objectType: "CodeVersion",
                        req,
                    });

                    await redisClient.set(redisKey, JSON.stringify(codeObject), { EX: CACHE_TTL_SECONDS });
                }

                const code = codeObject.content || "";

                // Run the user code with the provided input
                result = await runUserCode({ codeLanguage: codeObject.codeLanguage, code, input });
            },
            trace: "0645",
        });

        if (!result) {
            throw new CustomError("0622", "InternalError", { process: "doSandbox" });
        }
        return result;
    } catch (error) {
        logger.error("Error importing data", { trace: "0554", error });
        return { __type: "error", error: "Internal error" };
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
