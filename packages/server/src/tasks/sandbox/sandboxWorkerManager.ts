// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-08-05
/* eslint-disable no-magic-numbers */
import { CodeLanguage } from "@vrooli/shared";
import { type ChildProcess, fork } from "child_process";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { logger } from "../../events/logger.js";
import { DEFAULT_HEARTBEAT_CHECK_INTERVAL_MS, DEFAULT_HEARTBEAT_TIMEOUT_MS, DEFAULT_IDLE_TIMEOUT_MS, DEFAULT_JOB_TIMEOUT_MS, DEFAULT_MEMORY_LIMIT_MB, MB, WORKER_READY_TIMEOUT_MS } from "./consts.js";
import { type RunUserCodeInput, type RunUserCodeOutput, type SandboxWorkerInput, type SandboxWorkerMessage } from "./types.js";
import { bufferRegister, urlRegister } from "./utils.js";

const IN_TEST_MODE = process.env.NODE_ENV === "test";

const dirname = path.dirname(fileURLToPath(import.meta.url));
// const WORKER_THREAD_FILE = `${dirname}/workerThread.js`;
const WORKER_PROCESS_FILE = `${dirname}/workerProcess.js`;
const SUPPORTED_CODE_LANGUAGES = [
    CodeLanguage.Javascript,
];

// Lazy-load superjson so that we only initialize it once
// eslint-disable-next-line @typescript-eslint/consistent-type-imports
let lazySuperJSON: typeof import("superjson").default | null = null;
async function getSuperJSON() {
    if (!lazySuperJSON) {
        lazySuperJSON = (await import("superjson")).default;
        // Register custom types (i.e. types that are not natively serializable that we want to support)
        lazySuperJSON.registerCustom(urlRegister, "URL");
        lazySuperJSON.registerCustom(bufferRegister, "Buffer");
    }
    return lazySuperJSON;
}

type JobQueue = Array<{
    /** Resolves the job's promise with the output of the user code. */
    resolve: (value: RunUserCodeOutput) => void;
    /** Rejects the job's promise with an error. */
    reject: (reason: unknown) => void;
    /** The code and input data to run in the worker. */
    input: RunUserCodeInput;
}>;

type SandboxWorkerManagerParams = {
    memoryLimitBytes?: number;
    idleTimeoutMs?: number;
    jobTimeoutMs?: number;
    heartbeatCheckIntervalMs?: number;
    heartbeatTimeoutMs?: number;
};

type ManagerStatus = "Inactive" | "Starting" | "Idle" | "Processing" | "Terminating";

/**
 * Abstract base class for managing workers that execute user-generated code in a sandboxed environment.
 * Handles job queuing, timeouts, and heartbeat monitoring, while delegating worker-specific operations to subclasses.
 */
export abstract class AbstractSandboxWorkerManager {
    protected memoryLimitBytes: number;
    protected idleTimeoutMs: number;
    protected jobTimeoutMs: number;
    protected heartbeatCheckIntervalMs: number;
    protected heartbeatTimeoutMs: number;

    protected lastHeartbeat: number | null = null;
    private heartbeatIntervalHandle: NodeJS.Timeout | null = null;
    private idleTimeoutHandle: NodeJS.Timeout | null = null;
    private jobTimeoutHandle: NodeJS.Timeout | null = null;

    protected jobQueue: JobQueue = [];
    protected status: ManagerStatus = "Inactive";

    private workerReady: Promise<void>;
    private resolveWorkerReady: (() => void) | null = null;
    private rejectWorkerReady: ((reason: unknown) => void) | null = null;

    constructor(params?: SandboxWorkerManagerParams) {
        this.memoryLimitBytes = params?.memoryLimitBytes ?? DEFAULT_MEMORY_LIMIT_MB * MB;
        this.idleTimeoutMs = params?.idleTimeoutMs ?? DEFAULT_IDLE_TIMEOUT_MS;
        this.jobTimeoutMs = params?.jobTimeoutMs ?? DEFAULT_JOB_TIMEOUT_MS;
        this.heartbeatCheckIntervalMs = params?.heartbeatCheckIntervalMs ?? DEFAULT_HEARTBEAT_CHECK_INTERVAL_MS;
        this.heartbeatTimeoutMs = params?.heartbeatTimeoutMs ?? DEFAULT_HEARTBEAT_TIMEOUT_MS;

        this.workerReady = this._setupWorkerReadyPromise();
    }

    /** Abstract methods to be implemented by subclasses */
    protected abstract _createWorker(): Promise<void>;
    protected abstract _terminateWorker(): Promise<void>;
    protected abstract _sendMessageToWorker(message: SandboxWorkerInput): void;
    /**
     * Checks if the worker is currently active.
     * Must be implemented by subclasses.
     */
    protected abstract _isWorkerActive(): boolean;
    /**
     * Gets an identifier for the worker, or null if the worker is not active.
     * Must be implemented by subclasses.
     */
    protected abstract _getWorkerId(): string | null;
    /**
     * Returns a promise that resolves when the worker exits.
     * Must be implemented by subclasses.
     */
    protected abstract _getWorkerExitPromise(): Promise<void>;

    private _setupWorkerReadyPromise(): Promise<void> {
        return new Promise((resolve, reject) => {
            this.resolveWorkerReady = resolve;
            this.rejectWorkerReady = reject;
        });
    }

    private _clearTimeout(handle: NodeJS.Timeout | null): null {
        if (handle) clearTimeout(handle);
        return null;
    }

    private _startOrResetIdleTimer(): void {
        this.idleTimeoutHandle = this._clearTimeout(this.idleTimeoutHandle);
        this.idleTimeoutHandle = setTimeout(() => {
            this.terminate();
        }, this.idleTimeoutMs);
    }

    private _rejectTopJob(error: unknown): void {
        this.jobTimeoutHandle = this._clearTimeout(this.jobTimeoutHandle);
        const job = this.jobQueue.shift();
        if (job) job.reject(error);
    }

    private async _startWorker(): Promise<void> {
        // Start if something is wrong with the current worker
        if (this._getWorkerId() !== null && !this._isWorkerActive()) {
            await this.terminate();
        }

        // Can only start from an inactive (no worker) state and there are jobs to process
        if (this.status !== "Inactive" || this.jobQueue.length === 0) return;
        this.status = "Starting";

        try {
            await this._createWorker();
            this.lastHeartbeat = Date.now();
            this.heartbeatIntervalHandle = setInterval(() => {
                if (Date.now() - (this.lastHeartbeat || 0) > this.heartbeatTimeoutMs) {
                    console.error("Worker missed heartbeat; assuming crash");
                    this._handleWorkerCrash();
                }
            }, this.heartbeatCheckIntervalMs);

            const workerReadyTimeout = setTimeout(async () => {
                if (this.rejectWorkerReady) {
                    const err = new Error("Worker failed to initialize within timeout");
                    this.rejectWorkerReady(err);
                    this._rejectTopJob(err);
                }
                await this.terminate();
                this.status = "Inactive";
            }, WORKER_READY_TIMEOUT_MS);

            this.workerReady.finally(() => {
                clearTimeout(workerReadyTimeout);
                this.status = "Idle";
                if (this.jobQueue.length > 0) {
                    this._processNextJob();
                }
            });
        } catch (error) {
            this.status = "Inactive";
            if (this.rejectWorkerReady) this.rejectWorkerReady(error);
            this._rejectTopJob(error);
        }
    }

    public async terminate(): Promise<void> {
        if (this.status !== "Idle") return;
        this.status = "Terminating";
        this.idleTimeoutHandle = this._clearTimeout(this.idleTimeoutHandle);
        this.jobTimeoutHandle = this._clearTimeout(this.jobTimeoutHandle);
        this.heartbeatIntervalHandle = this._clearTimeout(this.heartbeatIntervalHandle);
        this.workerReady = this._setupWorkerReadyPromise();
        await this._terminateWorker();
        this.status = "Inactive";
    }

    private async _handleWorkerCrash(): Promise<void> {
        this._rejectTopJob(new Error("Worker crashed or became unresponsive"));
        await this.terminate();
        await this._startWorker();
    }

    protected async _handleWorkerError(error: Error): Promise<void> {
        this._rejectTopJob(error);
        await this.terminate();
        await this._startWorker();
    }

    protected async _handleWorkerExit(code: number | null, signal?: string | null): Promise<void> {
        if (this.status === "Inactive" || this.status === "Terminating") return;
        if (code !== null && code !== 0) {
            const error = new Error(`Worker exited with code ${code}`);
            this._rejectTopJob(error);
            await this.terminate();
            await this._startWorker();
        } else if (signal !== null && signal !== undefined) {
            const error = new Error(`Worker exited with signal ${signal}`);
            this._rejectTopJob(error);
            await this.terminate();
            await this._startWorker();
        }
    }

    protected async _handleWorkerMessage(message: SandboxWorkerMessage): Promise<void> {
        if (message.__type === "log") {
            if (IN_TEST_MODE) {
                console.log("Worker log:", message.log);
            }
        } else if (message.__type === "ready") {
            this.lastHeartbeat = Date.now();
            if (this.resolveWorkerReady) {
                this.resolveWorkerReady();
                this.resolveWorkerReady = null;
                this.rejectWorkerReady = null;
            }
        } else if (message.__type === "heartbeat") {
            this.lastHeartbeat = Date.now();
        } else {
            this.jobTimeoutHandle = this._clearTimeout(this.jobTimeoutHandle);
            this.heartbeatIntervalHandle = this._clearTimeout(this.heartbeatIntervalHandle);
            const job = this.jobQueue.shift();
            if (job) {
                if (message.__type === "output") {
                    let parsedOutput: unknown;
                    try {
                        const SuperJSON = await getSuperJSON();
                        parsedOutput = typeof message.output === "string" ? SuperJSON.parse(message.output) : message.output;
                    } catch (error) {
                        job.reject(error);
                    }
                    job.resolve({ __type: "output", output: parsedOutput });
                } else if (message.__type === "error") {
                    job.reject(new Error(message.error));
                } else {
                    job.resolve(message); // Fallback for unexpected message types
                }
                this._processNextJob();
            }
        }
    }

    private async _processNextJob(): Promise<void> {
        if (this.status !== "Idle") return;
        if (this.jobQueue.length === 0) {
            this.status = "Idle";
            this._startOrResetIdleTimer();
            return;
        }

        this.status = "Processing";
        this.idleTimeoutHandle = this._clearTimeout(this.idleTimeoutHandle);

        const job = this.jobQueue[0];

        try {
            const SuperJSON = await getSuperJSON();
            const safeInput = SuperJSON.stringify(job.input.input);

            await this.workerReady;
            this.jobTimeoutHandle = setTimeout(async () => {
                const error = new Error(`Job timed out after ${this.jobTimeoutMs} ms`);
                this._rejectTopJob(error);
                await this.terminate();
                await this._startWorker();
            }, this.jobTimeoutMs);

            this._sendMessageToWorker({
                code: job.input.code,
                codeLanguage: job.input.codeLanguage,
                input: safeInput,
                shouldSpreadInput: job.input.shouldSpreadInput,
            });
        } catch (error) {
            console.error(`Error processing job: ${error}`);
            this._rejectTopJob(error);
        } finally {
            this.status = "Idle";
            // Shifting the job queue and starting the next job is handled in the message handler
        }
    }

    /**
    * Runs user-provided code in the managed worker thread.
    * @param input The code and input data to run in the worker.
    * @returns A promise that resolves with the output of the user code,
    * or an error caused by the code or the worker.
    */
    public async runUserCode(input: RunUserCodeInput): Promise<RunUserCodeOutput> {
        let result: RunUserCodeOutput;
        try {
            result = await new Promise((resolve, reject) => {
                if (!SUPPORTED_CODE_LANGUAGES.includes(input.codeLanguage)) {
                    reject(new Error(`Unsupported code language: ${input.codeLanguage}`));
                    return;
                }

                this.jobQueue.push({ resolve, reject, input });
                this._startWorker();
                this._processNextJob();
            });
        } catch (error) {
            return { __type: "error", error: error instanceof Error ? error.message : String(error) };
        }
        return result;
    }
}

export class SandboxChildProcessManager extends AbstractSandboxWorkerManager {
    private childProcess: ChildProcess | null = null;

    /**
     * Creates a new child process to execute sandboxed code.
     */
    protected async _createWorker(): Promise<void> {
        let fileName = WORKER_PROCESS_FILE;
        if (IN_TEST_MODE) {
            // Adjust path for test environment to use compiled JS
            fileName = fileName.replace("/src/", "/test-dist/");
        }

        // Verify the worker script exists and is not empty
        const stats = fs.statSync(fileName);
        if (stats.size === 0) {
            throw new Error("Worker process file is empty. Recompile the project.");
        }

        // Fork a new child process
        this.childProcess = fork(fileName, {
            env: {
                ...process.env,
                MEMORY_LIMIT: this.memoryLimitBytes.toString(),
                JOB_TIMEOUT_MS: this.jobTimeoutMs.toString(),
            },
        });

        // Set up event listeners
        this.childProcess.on("error", (error) => this._handleWorkerError(error));
        this.childProcess.on("message", (message) => this._handleWorkerMessage(message as SandboxWorkerMessage));
        this.childProcess.on("exit", (code, signal) => this._handleWorkerExit(code, signal));
    }

    protected _isWorkerActive(): boolean {
        return (
            this.childProcess !== null &&
            !this.childProcess.killed &&
            this.childProcess.exitCode === null
        );
    }

    protected _getWorkerId(): string | null {
        return this.childProcess?.pid?.toString() ?? null;
    }

    /**
     * Returns a promise that resolves when the child process exits.
     * @returns {Promise<void>} Resolves when the 'exit' event is emitted.
     */
    protected _getWorkerExitPromise(): Promise<void> {
        return new Promise((resolve) => {
            if (this.childProcess) {
                this.childProcess.once("exit", () => resolve());
            } else {
                resolve(); // No child process, resolve immediately
            }
        });
    }

    /**
     * Terminates the child process and waits for it to exit.
     * Implements graceful shutdown with force-kill fallback.
     */
    protected async _terminateWorker(): Promise<void> {
        if (this.childProcess) {
            const pid = this.childProcess.pid;
            logger.debug(`Terminating sandbox worker (PID: ${pid}) with SIGTERM...`);

            // Send SIGTERM for graceful shutdown
            this.childProcess.kill("SIGTERM");

            // Race between graceful exit and force-kill timeout
            const forceKillTimeout = new Promise<void>((resolve) => {
                setTimeout(() => {
                    if (this.childProcess && !this.childProcess.killed) {
                        logger.warn(`Sandbox worker (PID: ${pid}) did not exit gracefully, force killing with SIGKILL`);
                        this.childProcess.kill("SIGKILL");
                    }
                    resolve();
                }, 10000); // 10 second grace period
            });

            // Wait for either graceful exit or force-kill timeout
            await Promise.race([
                this._getWorkerExitPromise(),
                forceKillTimeout,
            ]);

            // Ensure we wait for actual process exit after potential force-kill
            await this._getWorkerExitPromise();

            logger.debug(`Sandbox worker (PID: ${pid}) terminated successfully`);
            this.childProcess = null;
        }
    }

    /**
     * Sends a message to the child process.
     * @param {SandboxWorkerInput} message - The message to send.
     * @throws {Error} If the child process is not initialized.
     */
    protected _sendMessageToWorker(message: SandboxWorkerInput): void {
        if (this.childProcess) {
            this.childProcess.send(message);
        } else {
            throw new Error("Child process not initialized");
        }
    }
}
