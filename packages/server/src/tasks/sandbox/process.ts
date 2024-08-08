import { CodeVersion, HOURS_1_S } from "@local/shared";
import { Job } from "bull";
import { readOneHelper } from "../../actions/reads";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { withRedis } from "../../redisConn";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData";
import { permissionsCheck } from "../../validators/permissions";
import { SandboxRequestPayload } from "./queue";
import { RunUserCodeOutput, SandboxProcessPayload } from "./types";
import { WorkerThreadManager } from "./workerThreadManager";

/** How long to cache the code in Redis */
const CACHE_TTL_SECONDS = HOURS_1_S;

const codeVersionSelect = {
    id: true,
    content: true,
} as const;

export async function doSandbox({
    codeVersionId,
    input,
    userData,
}: SandboxProcessPayload) {
    try {
        // Create a new child process manager to run the user code
        const manager = new WorkerThreadManager();

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
                        throw new CustomError("0620", "NotFound", userData.languages, { codeVersionId });
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
                result = await manager.runUserCode({ code, input });
            },
            trace: "0645",
        });

        if (!result) {
            throw new CustomError("0622", "InternalError", ["en"], { process: "doSandbox" });
        }
        return result;
    } catch (err) {
        logger.error("Error importing data", { trace: "0554" });
    }
}

export async function sandboxrocess({ data }: Job<SandboxRequestPayload>) {
    switch (data.__process) {
        case "Sandbox":
            return doSandbox(data);
        case "Test":
            logger.info("sandboxProcess test triggered");
            return { __typename: "Success" as const, success: true };
        default:
            throw new CustomError("0608", "InternalError", ["en"], { process: (data as { __process?: unknown }).__process });
    }
}
