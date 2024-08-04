import { CodeVersion } from "@local/shared";
import { Job } from "bull";
import { readOneHelper } from "../../actions/reads";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { ChildProcessManager } from "./childProcessManager";
import { SandboxRequestPayload } from "./queue";
import { SandboxProcessPayload } from "./types";

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
        const manager = new ChildProcessManager();
        // Read the user code from the database, in a way that throws an error if the code is not found or the user desn't have permission
        const req = { session: { languages: userData.languages, users: [userData] } };
        const codeObject = await readOneHelper<CodeVersion>({
            info: codeVersionSelect,
            input: { id: codeVersionId },
            objectType: "CodeVersion",
            req,
        });
        const code = codeObject.content || "";
        // Run the user code with the provided input
        const result = await manager.runUserCode({ code, input });
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
