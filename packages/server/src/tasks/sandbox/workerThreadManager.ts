import path from "path";
import { fileURLToPath } from "url";
import { Worker } from "worker_threads";
import { DEFAULT_IDLE_TIMEOUT_MS, DEFAULT_MEMORY_LIMIT_MB, MB } from "./consts";
import { RunUserCodeInput, RunUserCodeOutput } from "./types";

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

    constructor(memoryLimit = DEFAULT_MEMORY_LIMIT_MB * MB, idleTimeout = DEFAULT_IDLE_TIMEOUT_MS) {
        this.memoryLimit = memoryLimit;
        this.idleTimeout = idleTimeout;
        this.worker = null;
        this.timeoutHandle = null;
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
        this.worker = new Worker(fileName, {
            workerData: {
                memoryLimit: this.memoryLimit,
            },
        });
        this.worker.on("message", (message) => {
            console.log("Worker message:", message);
        });
        this.worker.on("error", (error) => {
            console.error("Worker error:", error);
        });
        this.worker.on("exit", (code) => {
            console.log(`Worker exited with code ${code}`);
            this.worker = null;
        });
        this._resetIdleTimer();
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
            console.log("Terminating worker thread");
            this.worker.terminate();
            this.worker = null;
        }
    }

    /**
    * Runs user-provided code in the managed worker thread.
    * @param input The string input to be passed to the user code.
    * @returns A promise that resolves with the output of the user code, or
    * rejects if an error occurs during execution.
    */
    public async runUserCode({
        code,
        input,
        shouldSpreadInput = false,
    }: RunUserCodeInput): Promise<RunUserCodeOutput> {
        console.log("in runUserCode a", this.worker);
        // Start process if it is not running
        if (!this.worker) {
            this._startWorker();
        }

        // Reset timer that shuts down the process if it is idle
        this._resetIdleTimer();

        // Send input to the child process and return a promise that resolves with the output
        return new Promise((resolve, reject) => {
            if (!this.worker) {
                console.log("Error: Worker thread not running");
                reject(new Error("Worker thread not running"));
                return;
            }

            function messageHandler(message: any) {
                console.log("qwerty messageHandler", message);
                cleanup();
                resolve(message);
            }

            function errorHandler(error: Error) {
                console.log("qwerty errorHandler", error);
                cleanup();
                reject(error);
            }

            function exitHandler(code: number | null) {
                console.log("qwerty exitHandler", code);
                cleanup();
                reject(new Error(`Child process exited with code ${code}`));
            }

            const cleanup = () => {
                if (this.worker) {
                    this.worker.removeListener("message", messageHandler);
                    this.worker.removeListener("error", errorHandler);
                    this.worker.removeListener("exit", exitHandler);
                }
            };

            this.worker.on("message", messageHandler);
            this.worker.on("error", errorHandler);
            this.worker.on("exit", exitHandler);

            try {
                console.log("Sending message to worker thread");
                this.worker.postMessage({ code, input, shouldSpreadInput });
                console.log("Message sent to worker thread successfully");
            } catch (error) {
                console.log(`Caught error while sending message: ${error}`);
                cleanup();
                reject(error);
            }
        });
    }
}

