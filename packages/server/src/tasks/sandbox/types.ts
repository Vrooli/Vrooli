import { type CodeLanguage, type ExecutionEnvironment, type EnvironmentConfig } from "@vrooli/shared";
import type { SandboxTask } from "../taskTypes.js";

export type RunUserCodeInput = Pick<SandboxTask, "input" | "shouldSpreadInput"> & {
    /**
     * The user code to be executed. Will be run according to the execution environment specified.
     * 
     * NOTE: To preserve escape characters, use a String.raw template literal.
     * Example: `String.raw`function test() { return "Hello, world!"; }`,
     */
    code: string;
    /**
     * The language of the user code.
     * 
     * Supported languages depend on the execution environment.
     */
    codeLanguage: CodeLanguage;
    /**
     * The execution environment where the code should run.
     * 
     * - "sandbox": Secure, isolated JavaScript execution (default)
     * - "local": Local execution for development (requires dev/test environment + feature flag)
     */
    executionEnvironment?: ExecutionEnvironment;
    /**
     * Environment-specific configuration options.
     */
    environmentConfig?: EnvironmentConfig;
}

export type SandboxWorkerInput = Omit<RunUserCodeInput, "input" | "executionEnvironment" | "environmentConfig"> & {
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
