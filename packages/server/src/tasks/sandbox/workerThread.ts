// This file should only be imported by the worker thread manager
import ivm from "isolated-vm";
import SuperJSON from "superjson";
import { parentPort, workerData } from "worker_threads";
import { DEFAULT_JOB_TIMEOUT_MS, DEFAULT_MEMORY_LIMIT_MB, MB } from "./consts.js"; // NOTE: The extension must be specified or else it will throw an error
import { WorkerThreadInput, WorkerThreadOutput } from "./types.js"; // NOTE: The extension must be specified or else it will throw an error
import { getFunctionDetails, urlRegister, urlWrapper } from "./utils.js"; // NOTE: The extension must be specified or else it will throw an error

/** Maximum length of code that can be executed */
const MAX_CODE_LENGTH = 8_192;
/** Length of unique identifier to avoid namespace collisions */
const UNIQUE_ID_LENGTH = 16;

if (!parentPort) {
    throw new Error("Script must be run within a worker thread");
}

/**
 * @returns A unique identifier to avoid namespace collisions when injecting code/inputs into the isolate
 */
function generateUniqueId() {
    return `__ivm_${Math.random().toString(UNIQUE_ID_LENGTH).slice(2, -1)}`; // Not cryptographically secure, but doesn't need to be
}

function postMessage(message: WorkerThreadOutput) {
    parentPort?.postMessage(message);
}

// Create sandboxed environment for code execution. 
const memoryLimitBytes = workerData.memoryLimit || DEFAULT_MEMORY_LIMIT_MB;
// eslint-disable-next-line no-magic-numbers
const memoryLimitMb = Math.floor(memoryLimitBytes / MB);
const jobTimeoutMs = workerData.jobTimeoutMs || DEFAULT_JOB_TIMEOUT_MS;

let isolate: ivm.Isolate | null = null;
function getIsolate() {
    if (!isolate || isolate.isDisposed) {
        isolate = new ivm.Isolate({ memoryLimit: memoryLimitMb });
    }
    return isolate;
}

// Listen for messages from the parent thread
parentPort.on("message", async ({ code, input, shouldSpreadInput }: WorkerThreadInput) => {
    // Throw error is code is too long
    if (code.length > MAX_CODE_LENGTH) {
        postMessage({ __type: "error", error: "Code is too long" });
        return;
    }

    // Extract the function name from the code
    const { functionName } = getFunctionDetails(code);
    if (!functionName) {
        postMessage({ __type: "error", error: "Function name not found" });
        return;
    }

    let context: ivm.Context | null = null;
    let currentIsolate: ivm.Isolate | null = null;
    try {
        // Create a new context within this isolate
        currentIsolate = getIsolate();
        context = currentIsolate.createContextSync();

        // Get a reference to the global object within the context
        const jail = context.global;
        // This makes the global object available in the context as `global`. We use `derefInto()` here
        // because otherwise `global` would actually be a Reference{} object in the new isolate.
        jail.setSync("global", jail.derefInto());

        // Register custom SuperJSON functions for things that aren't really custom,
        // but break because of lack of support in v8.
        SuperJSON.registerCustom(urlRegister, "URL");

        // Generate unique identifiers for our placeholders
        const uniqueId = generateUniqueId();
        const inputId = `${uniqueId}Input`;
        // const executeId = `${uniqueId}Execute`;
        const superjsonId = `${uniqueId}SuperJSON`;
        const urlId = `${uniqueId}URL`;

        // Store the input (which should have been safely serialized before being sent to the worker thread)
        jail.setSync(inputId, input);

        // Need to log something in the vm? Uncomment the code below and add `log()` calls in `wrappedCode`
        // jail.setSync("log", function handleVmLog(...args) {
        //     parentPort?.postMessage({ __type: "log", log: args.join(" ") });
        // });

        // Add SuperJSON to the context so that we can parse and stringify inputs/outputs
        context.evalClosure(`
            global.${superjsonId} = {
                parse: $0,
                stringify: $1
            }
          `,
            [
                (...args) => {
                    if (args.length === 0) {
                        return null;
                    }
                    return SuperJSON.parse(args[0]);
                },
                (...args) => {
                    if (args.length === 0) {
                        return null;
                    }
                    return SuperJSON.stringify(args[0]);
                },
            ]);

        // Add additional functions to context if the code requires them
        let additionalFunctions = "";
        const needsURL = code.includes("URL");
        if (needsURL) {
            context.global.setSync(urlId, new ivm.Callback(urlWrapper));

            // Create a URL class in the isolate
            additionalFunctions += `
                class URL {
                    constructor(url, base) {
                        const urlData = ${urlId}(url, base);
                        Object.assign(this, urlData);
                    }

                    // Add any methods you need, e.g.:
                    toString() {
                        return this.href;
                    }
                }

                // Make URL available globally
                globalThis.URL = URL;

            `;
        }

        // Define the user’s function
        await context.eval(additionalFunctions + code);

        // Execute it asynchronously
        const executeClosure = `
            async function execute() {
                const input = global.${superjsonId}.parse(global.${inputId});
                const output = await global.${functionName}(${shouldSpreadInput ? "...input" : "input"});
                return global.${superjsonId}.stringify(output);
            }
            return execute();
        `;

        // Run the code
        const timeoutPromise = new Promise((_, reject) =>
            setTimeout(() => {
                // if (currentIsolate && !currentIsolate.isDisposed) {
                //     currentIsolate.dispose();
                // }
                reject(new Error("Execution timed out"));
            }, jobTimeoutMs),
        );
        const result = await Promise.race([
            context.evalClosure(executeClosure, [], { result: { promise: true } }),
            timeoutPromise,
        ]);

        // const output = SuperJSON.parse(result);
        const output = result === "undefined" ? undefined : SuperJSON.parse(result);

        // Send the output back to the parent thread
        postMessage({ __type: "output", output });
    } catch (error: unknown) {
        if (error instanceof Error) {
            postMessage({ __type: "error", error: error.message });
        } else {
            postMessage({ __type: "error", error: "An unknown error occurred" });
        }
    } finally {
        // Clean up the context if it was created
        context?.release();
        // Clean up the isolate if it was created
        if (currentIsolate && !currentIsolate.isDisposed) {
            currentIsolate.dispose();
        }
    }
});

// Handle errors in the worker thread
process.on("uncaughtException", (error) => {
    postMessage({ __type: "error", error: `Uncaught exception: ${error.message}` });
    process.exit(1);
});

process.on("unhandledRejection", (reason) => {
    postMessage({ __type: "error", error: `Unhandled rejection: ${reason}` });
    process.exit(1);
});

// Send a ready message to the parent thread, indicating that the worker thread is ready to receive messages
postMessage({ __type: "ready" });
