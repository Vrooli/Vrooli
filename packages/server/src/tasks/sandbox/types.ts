import { TaskStatus } from "@local/shared";
import { SessionUserToken } from "../../types";

export type SandboxProcessPayload = {
    __process: "Sandbox";
    /** The ID of the code version in the database */
    codeVersionId: string;
    /**
     * The input to be passed to the user code.
     * 
     * Examples:
     * - { plainText: "Hello, world!" }
     * - { numbers: [1, 2, 3, 4, 5], operation: "sum" }
     * - [1, 2, 3, 4, 5]
     * - "Hello, world!"
     */
    input?: unknown;
    /**
     * True if the input is an array and should be passed as separate arguments to the user code (i.e. spread). 
     * Defaults to false.
     */
    shouldSpreadInput?: boolean;
    /** The status of the job process */
    status: TaskStatus | `${TaskStatus}`;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
}

export type RunUserCodeInput = Pick<SandboxProcessPayload, "input" | "shouldSpreadInput"> & {
    /**
     * The user code to be executed. Will be run in a secure sandboxed environment, 
     * with no access to the file system or network.
     */
    code: string;
}

export type WorkerThreadInput = Omit<RunUserCodeInput, "input"> & {
    input: string;
}

export type WorkerThreadMessageError = {
    __type: "error";
    error: string;
}
export type WorkerThreadMessageLog = {
    __type: "log";
    log: string;
}
export type WorkerThreadMessageOutput = {
    __type: "output";
    output: unknown;
}
export type WorkerThreadMessageReady = {
    __type: "ready";
}
export type WorkerThreadOutput = WorkerThreadMessageError | WorkerThreadMessageLog | WorkerThreadMessageOutput | WorkerThreadMessageReady;

export interface RunUserCodeOutput {
    /**
     * The error message if an error occurred during execution, or undefined if successful.
     */
    error?: string;
    /**
     * The output of the user code execution, or undefined if an error occurred or no output was provided.
     */
    output?: unknown;
}
