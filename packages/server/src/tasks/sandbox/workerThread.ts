// This file should only be imported by the worker thread manager
import ivm from "isolated-vm";
import { parentPort, workerData } from "worker_threads";
import { DEFAULT_HEARTBEAT_SEND_INTERVAL_MS, DEFAULT_JOB_TIMEOUT_MS, DEFAULT_MEMORY_LIMIT_MB, MB } from "./consts.js"; // NOTE: The extension must be specified or else it will throw an error
import { SandboxWorkerInput, SandboxWorkerMessage } from "./types.js"; // NOTE: The extension must be specified or else it will throw an error
import { bufferRegister, bufferWrapper, getBufferClassString, getFunctionDetails, getURLClassString, urlRegister, urlWrapper } from "./utils.js"; // NOTE: The extension must be specified or else it will throw an error

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

function postMessage(message: SandboxWorkerMessage) {
    parentPort?.postMessage(message);
}

// Create sandboxed environment for code execution. 
const memoryLimitBytes = workerData.memoryLimit || DEFAULT_MEMORY_LIMIT_MB;
// eslint-disable-next-line no-magic-numbers
const memoryLimitMb = Math.floor(memoryLimitBytes / MB);
const jobTimeoutMs = workerData.jobTimeoutMs || DEFAULT_JOB_TIMEOUT_MS;

// Sandboxed environment setup
let isolate: ivm.Isolate | null = null;
function getIsolate() {
    if (!isolate || isolate.isDisposed) {
        try {
            isolate = new ivm.Isolate({ memoryLimit: memoryLimitMb });
        } catch (error) {
            console.error("Error creating isolate:", error);
        }
    }
    return isolate;
}

// Setup SuperJSON
let lazySuperJSON: typeof import("superjson").default | null = null;
async function getSuperJSON() {
    if (!lazySuperJSON) {
        try {
            lazySuperJSON = (await import("superjson")).default;
            // Register custom types (i.e. types that are not natively serializable that we want to support)
            lazySuperJSON.registerCustom(urlRegister, "URL");
            lazySuperJSON.registerCustom(bufferRegister, "Buffer");
        } catch (error) {
            console.error("Error importing superjson:", error);
        }
    }
    return lazySuperJSON;
}

// Listen for messages from the parent thread
parentPort.on("message", async ({ code, input, shouldSpreadInput }: SandboxWorkerInput) => {
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
    let heartbeatInterval: NodeJS.Timeout | null = null;

    try {
        // Start sending heartbeats
        heartbeatInterval = setInterval(() => {
            postMessage({ __type: "heartbeat" });
        }, DEFAULT_HEARTBEAT_SEND_INTERVAL_MS);

        // Create a new context within this isolate
        currentIsolate = getIsolate();
        if (!currentIsolate) {
            throw new Error("Failed to create isolate");
        }
        context = currentIsolate.createContextSync();

        // Get a reference to the global object within the context
        const jail = context.global;
        // This makes the global object available in the context as `global`. We use `derefInto()` here
        // because otherwise `global` would actually be a Reference{} object in the new isolate.
        jail.setSync("global", jail.derefInto());

        const superjson = await getSuperJSON();
        if (!superjson) {
            throw new Error("Failed to get superjson");
        }

        // Generate unique identifiers for our placeholders
        const uniqueId = generateUniqueId();
        const inputId = `${uniqueId}Input`;
        const superjsonId = `${uniqueId}SuperJSON`;

        // Store the input (which should have been safely serialized before being sent to the worker thread)
        jail.setSync(inputId, input);

        // Add SuperJSON to the context so that we can parse and stringify inputs/outputs
        context.evalClosure(`
            global.${superjsonId} = {
                parse: $0,
                stringify: $1
            }
        `, [
            (...args) => (args.length === 0 ? null : superjson.parse(args[0])),
            (...args) => (args.length === 0 ? null : superjson.stringify(args[0])),
        ]);

        // Add additional functions to context if the code might need them
        let additionalFunctions = "";
        const needsURL = code.includes("URL");
        if (needsURL) {
            const urlId = `${uniqueId}URL`;
            context.global.setSync(urlId, new ivm.Callback(urlWrapper));
            additionalFunctions += getURLClassString(urlId);
        }
        const needsBuffer = code.includes("Buffer");
        if (needsBuffer) {
            const bufferId = `${uniqueId}Buffer`;
            context.global.setSync(bufferId, new ivm.Callback(bufferWrapper));
            additionalFunctions += getBufferClassString(bufferId);
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
        const output = result === "undefined" ? "{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}" : result;

        // Send the output back to the parent thread
        postMessage({ __type: "output", output });
    } catch (error: unknown) {
        if (error instanceof Error) {
            postMessage({ __type: "error", error: error.message });
        } else {
            postMessage({ __type: "error", error: "An unknown error occurred" });
        }
    } finally {
        // Stop sending heartbeats
        if (heartbeatInterval) {
            clearInterval(heartbeatInterval);
        }
        // Clean up the context if it was created
        context?.release();
        // Clean up the isolate if it was created
        if (currentIsolate && !currentIsolate.isDisposed) {
            try {
                currentIsolate.dispose();
            } catch (error: unknown) {
                console.error("Error disposing isolate:", error);
                postMessage({ __type: "error", error: "Failed to dispose isolate" });
            }
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
