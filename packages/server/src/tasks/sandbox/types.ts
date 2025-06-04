import { type CodeLanguage } from "@local/shared";
import type { SandboxTask } from "../taskTypes.js";

export type RunUserCodeInput = Pick<SandboxTask, "input" | "shouldSpreadInput"> & {
    /**
     * The user code to be executed. Will be run in a secure sandboxed environment, 
     * with no access to the file system or network.
     * 
     * NOTE: To preserve escape characters, use a String.raw template literal.
     * Example: `String.raw`function test() { return "Hello, world!"; }`,
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
