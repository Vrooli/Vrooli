import fs from "fs";
import path from "path";
import SuperJSON from "superjson";
import { fileURLToPath } from "url";
import { Worker } from "worker_threads";
import { DEFAULT_IDLE_TIMEOUT_MS, DEFAULT_MEMORY_LIMIT_MB, MB, WORKER_READY_TIMEOUT_MS } from "./consts";
import { RunUserCodeInput, RunUserCodeOutput, WorkerThreadOutput } from "./types";
import { urlRegister } from "./utils";

const dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * Manages a worker thread to safely execute user-generated code in a sandboxed environment.
 * Ensures the worker is started when needed, reused for subsequent tasks, and
 * terminated if it remains idle for a specified period, freeing up resources.
 */
export class WorkerThreadManager {
    private memoryLimit: number;
    private idleTimeout: number;
    private worker: Worker | null;
    private timeoutHandle: NodeJS.Timeout | null;
    private jobQueue: Array<{
        resolve: (value: RunUserCodeOutput) => void;
        reject: (reason: any) => void;
        input: RunUserCodeInput;
    }>;
    private isProcessing: boolean;
    private workerReady: Promise<void>;
    private resolveWorkerReady: (() => void) | null;
    private rejectWorkerReady: ((reason: any) => void) | null;

    constructor(memoryLimit = DEFAULT_MEMORY_LIMIT_MB * MB, idleTimeout = DEFAULT_IDLE_TIMEOUT_MS) {
        this.memoryLimit = memoryLimit;
        this.idleTimeout = idleTimeout;
        this.worker = null;
        this.timeoutHandle = null;
        this.jobQueue = [];
        this.isProcessing = false;
        this.resolveWorkerReady = null;
        this.rejectWorkerReady = null;
        this.workerReady = new Promise((resolve, reject) => {
            this.resolveWorkerReady = resolve;
            this.rejectWorkerReady = reject;
        });
    }

    /**
     * Starts a new worker thread to execute the user code.
     * Configures the worker thread with a memory limit and sets up event listeners
     * for process exit to handle unexpected terminations.
     */
    private _startWorker(): void {
        let fileName = `${dirname}/workerThread.js`;
        // If we're running tests, we may need to change the file path to point to the compiled JS file
        if (process.env.NODE_ENV === "test") {
            fileName = fileName.replace("/src/", "/dist/");
        }

        // Check if the file exists and is not empty
        try {
            const stats = fs.statSync(fileName);
            if (stats.size === 0) {
                throw new Error("Worker thread file is empty. You may need to recomplie the project.");
            }
        } catch (error) {
            console.error(`Error reading worker thread file: ${error}`);
            if (this.rejectWorkerReady) {
                this.rejectWorkerReady(error);
            }
            return;
        }

        try {
            this.worker = new Worker(fileName, {
                workerData: {
                    memoryLimit: this.memoryLimit,
                },
            });
            this.worker.on("error", this._handleWorkerError.bind(this));
            this.worker.on("message", this._handleWorkerMessage.bind(this));

            // Set a timeout for the worker to become ready
            const workerReadyTimeout = setTimeout(() => {
                if (this.rejectWorkerReady) {
                    this.rejectWorkerReady(new Error("Worker failed to initialize within the timeout period"));
                }
                this._terminateWorker();
            }, WORKER_READY_TIMEOUT_MS);

            // Clear the timeout when the worker is ready
            this.workerReady.then(() => {
                clearTimeout(workerReadyTimeout);
            }).catch(() => {
                clearTimeout(workerReadyTimeout);
            });

            this._resetIdleTimer();
        } catch (error) {
            console.error(`Error starting worker thread: ${error}`);
            if (this.rejectWorkerReady) {
                this.rejectWorkerReady(error);
            }
        }
    }

    /**
     * Resets the idle timer that determines when to terminate the worker thread. 
     * This should be called every time the worker thread is used, so that it 
     * stays alive when in frequent use.
     */
    private _resetIdleTimer(): void {
        if (this.timeoutHandle) {
            clearTimeout(this.timeoutHandle);
        }
        this.timeoutHandle = setTimeout(() => this._terminateWorker(), this.idleTimeout);
    }

    /**
     * Terminates the child process if it is running.
     */
    private _terminateWorker(): void {
        if (this.worker) {
            this.worker.terminate();
            this.worker = null;
            // Reset the workerReady promise
            this.workerReady = new Promise((resolve, reject) => {
                this.resolveWorkerReady = resolve;
                this.rejectWorkerReady = reject;
            });
        }
    }

    private _handleWorkerError(error: Error): void {
        const job = this.jobQueue.shift();
        if (job) {
            job.reject(error);
        }
        this._terminateWorker();
        this._processNextJob();
    }

    private _handleWorkerMessage(message: WorkerThreadOutput): void {
        if (message.__type === "log") {
            console.log("Worker log:", message.log);
        } else if (message.__type === "ready") {
            if (this.resolveWorkerReady) {
                this.resolveWorkerReady();
            }
        } else {
            const job = this.jobQueue.shift();
            if (job) {
                job.resolve(message);
                this._processNextJob();
            }
        }
    }

    private async _processNextJob(): Promise<void> {
        if (this.jobQueue.length === 0) {
            this.isProcessing = false;
            return;
        }

        this.isProcessing = true;
        const job = this.jobQueue[0];

        try {
            SuperJSON.registerCustom(urlRegister, "URL");
            const safeInput = SuperJSON.stringify(job.input.input);
            if (!this.worker) {
                console.error("Worker thread not started. Call _startWorker() before processing jobs.");
                return;
            }
            // Wait for the worker to be ready before sending the message
            await this.workerReady;
            this.worker.postMessage({
                code: job.input.code,
                input: safeInput,
                shouldSpreadInput: job.input.shouldSpreadInput,
            });
        } catch (error) {
            console.error(`Caught error while sending message: ${error}`);
            this.jobQueue.shift();
            job.reject(error);
            this._processNextJob();
        }
    }

    /**
    * Runs user-provided code in the managed worker thread.
    * @param input The string input to be passed to the user code.
    * @returns A promise that resolves with the output of the user code, or
    * rejects if an error occurs during execution.
    */
    public async runUserCode(input: RunUserCodeInput): Promise<RunUserCodeOutput> {
        return new Promise((resolve, reject) => {
            this.jobQueue.push({ resolve, reject, input });

            if (!this.worker) {
                this._startWorker();
            }

            this._resetIdleTimer();

            if (!this.isProcessing) {
                this._processNextJob();
            }
        });
    }
}

