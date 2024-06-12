import { Job } from "bull";
import { prismaInstance } from "../../db/instance";
import { logger } from "../../events/logger";
import { ChildProcessManager } from "./childProcessManager";
import { SandboxProcessPayload } from "./types";

export const sandboxProcess = async ({ data }: Job<SandboxProcessPayload>) => {
    try {
        // Create a new child process manager to run the user code
        const manager = new ChildProcessManager();
        // Read the user code from the database
        const codeObject = await prismaInstance.code_version.findUnique({
            where: { id: data.codeVersionId },
            select: { id: true, content: true },
        });
        //TODO need permissions check to make sure you can run this code
        const code = codeObject!.content;
        // Run the user code with the provided input
        const result = await manager.runUserCode({ code, input: data.input });
        return result;
    } catch (err) {
        logger.error("Error importing data", { trace: "0554" });
    }
};
