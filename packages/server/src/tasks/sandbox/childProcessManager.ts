import { fork, type ChildProcess } from "child_process";
import { DEFAULT_IDLE_TIMEOUT_MS, DEFAULT_MEMORY_LIMIT_MB, MB } from "./consts";
import { RunUserCodeInput, RunUserCodeOutput } from "./types";

/**
 * Manages a child process to safely execute user-generated code in a sandboxed environment.
 * Ensures the child process is started when needed, reused for subsequent tasks, and
 * terminated if it remains idle for a specified period, freeing up resources.
 */
export class ChildProcessManager {
    private memoryLimit: number;
    private idleTimeout: number;
    private child: ChildProcess | null;
    private timeoutHandle: NodeJS.Timeout | null;

    constructor(memoryLimit = DEFAULT_MEMORY_LIMIT_MB * MB, idleTimeout = DEFAULT_IDLE_TIMEOUT_MS) {
        this.memoryLimit = memoryLimit;
        this.idleTimeout = idleTimeout;
        this.child = null;
        this.timeoutHandle = null;
    }

    /**
     * Starts a new child process to execute the user code.
     * Configures the child process with a memory limit and sets up event listeners
     * for process exit to handle unexpected terminations.
     */
    private _startChild(): void {
        this.child = fork(`${__dirname}/childScript.js`, [], {
            execArgv: [`--max-old-space-size=${this.memoryLimit / MB}`],
        });
        this.child.on("exit", (code) => {
            console.log(`Child process exited with code ${code}`);
            this.child = null;
        });
        this._resetIdleTimer();
    }

    /**
     * Resets the idle timer that determines when to terminate the child process. 
     * This should be called every time the child process is used, so that it 
     * stays alive when in frequent use.
     */
    private _resetIdleTimer(): void {
        if (this.timeoutHandle) {
            clearTimeout(this.timeoutHandle);
        }
        this.timeoutHandle = setTimeout(() => this._terminateChild(), this.idleTimeout);
    }

    /**
     * Terminates the child process if it is running.
     */
    private _terminateChild(): void {
        if (this.child) {
            this.child.kill();
            this.child = null;
        }
    }

    /**
    * Runs user-provided code in the managed child process.
    * @param input The string input to be passed to the user code.
    * @returns A promise that resolves with the output of the user code, or
    * rejects if an error occurs during execution.
    */
    public async runUserCode({
        code,
        input,
    }: RunUserCodeInput): Promise<RunUserCodeOutput> {
        // Start process if it is not running
        if (!this.child) {
            this._startChild();
        }

        // Reset timer that shuts down the process if it is idle
        this._resetIdleTimer();

        // Send input to the child process and return a promise that resolves with the output
        return new Promise((resolve, reject) => {
            if (!this.child) {
                reject(new Error("Child process not running"));
                return;
            }
            // Set up event listeners to receive response or error from child process
            this.child.once("message", resolve);
            this.child.once("error", reject);
            // Send the code and input to the child process
            this.child.send({ code, input });
        });
    }
}

