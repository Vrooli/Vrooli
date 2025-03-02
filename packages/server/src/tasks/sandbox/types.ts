import { CodeLanguage, SessionUser, TaskStatus } from "@local/shared";

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
    userData: SessionUser;
}

export type RunUserCodeInput = Pick<SandboxProcessPayload, "input" | "shouldSpreadInput"> & {
    /**
     * The user code to be executed. Will be run in a secure sandboxed environment, 
     * with no access to the file system or network.
     */
    code: string;
    /**
     * The language of the user code.
     * 
     * Any language not supported by the sandbox (which will probably only be JavaScript for a long time) will be rejected.
     */
    codeLanguage: CodeLanguage;
}

export type SandboxWorkerInput = Omit<RunUserCodeInput, "input"> & {
    input: string;
}

export type SandboxWorkerMessageError = {
    __type: "error";
    error: string;
}
export type SandboxWorkerMessageLog = {
    __type: "log";
    log: string;
}
export type SandboxWorkerMessageHeartbeat = {
    __type: "heartbeat";
}
export type SandboxWorkerMessageOutput = {
    __type: "output";
    output: string;
}
export type SandboxWorkerMessageReady = {
    __type: "ready";
}
export type SandboxWorkerMessage = SandboxWorkerMessageError | SandboxWorkerMessageLog | SandboxWorkerMessageHeartbeat | SandboxWorkerMessageOutput | SandboxWorkerMessageReady;
export type RunUserCodeOutput = {
    __type: "error" | "output";
    error?: string;
    output?: unknown;
}
