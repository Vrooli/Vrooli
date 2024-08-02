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
     */
    input: object;
    /** The status of the job process */
    status: TaskStatus | `${TaskStatus}`;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
}

export interface RunUserCodeInput {
    /**
     * The user code to be executed. Will be run in a secure sandboxed environment, 
     * with no access to the file system or network.
     */
    code: string;
    input: SandboxProcessPayload["input"];
}

export interface RunUserCodeOutput {
    /**
     * The error message if an error occurred during execution, or undefined if successful.
     */
    error?: string;
    /**
     * The output of the user code execution, or undefined if an error occurred.
     * 
     * Examples:
     * - { plainText: "Hello, world!" }
     * - { result: 15 }
     */
    output?: object;
}
