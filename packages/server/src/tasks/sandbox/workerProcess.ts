// This file should only be imported by the child process manager
import ivm from "isolated-vm";
// import SuperJSON from "superjson";
import { DEFAULT_HEARTBEAT_SEND_INTERVAL_MS, DEFAULT_JOB_TIMEOUT_MS, DEFAULT_MEMORY_LIMIT_MB, MB } from "./consts.js";
import { SandboxWorkerInput, SandboxWorkerMessage } from "./types.js";
import { bufferRegister, bufferWrapper, getBufferClassString, getFunctionDetails, getURLClassString, urlRegister, urlWrapper } from "./utils.js";

// Maximum length of code that can be executed
const MAX_CODE_LENGTH = 8_192;
// Length of unique identifier to avoid namespace collisions
const UNIQUE_ID_LENGTH = 16;

// Retrieve configuration from environment variables
const memoryLimitBytes = parseInt(process.env.MEMORY_LIMIT || DEFAULT_MEMORY_LIMIT_MB.toString(), 10);
const memoryLimitMb = Math.floor(memoryLimitBytes / MB);
const jobTimeoutMs = parseInt(process.env.JOB_TIMEOUT_MS || DEFAULT_JOB_TIMEOUT_MS.toString(), 10);

function postMessage(message: SandboxWorkerMessage) {
    if (process.send) {
        process.send(message);
    } else {
        console.error("process.send is not defined");
    }
}

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

// Listen for messages from the parent process
process.on("message", async ({ code, input, shouldSpreadInput }: SandboxWorkerInput) => {
    if (code.length > MAX_CODE_LENGTH) {
        postMessage({ __type: "error", error: "Code is too long" });
        return;
    }

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
        const uniqueId = `__ivm_${Math.random().toString(UNIQUE_ID_LENGTH).slice(2, -1)}`;
        const inputId = `${uniqueId}Input`;
        const superjsonId = `${uniqueId}SuperJSON`;

        jail.setSync(inputId, input);

        context.evalClosure(`
            global.${superjsonId} = {
                parse: $0,
                stringify: $1
            }
        `, [
            (...args) => (args.length === 0 ? null : superjson.parse(args[0])),
            (...args) => (args.length === 0 ? null : superjson.stringify(args[0])),
        ]);

        // Set up code to be evaluated
        let evalCode = "";
        // Add additional functions to context if the code might need them
        const needsURL = code.includes("URL");
        if (needsURL) {
            const urlId = `${uniqueId}URL`;
            context.global.setSync(urlId, new ivm.Callback(urlWrapper));
            evalCode += getURLClassString(urlId) + "\n";
        }
        const needsBuffer = code.includes("Buffer");
        if (needsBuffer) {
            const bufferId = `${uniqueId}Buffer`;
            context.global.setSync(bufferId, new ivm.Callback(bufferWrapper));
            evalCode += getBufferClassString(bufferId) + "\n";
        }
        // Add the function being executed
        evalCode += code + "\n";
        // Add an `execute` function that will be used to execute the code
        evalCode += `
async function execute() {
    const input = global.${superjsonId}.parse(global.${inputId});
    const output = await global.${functionName}(${shouldSpreadInput ? "...input" : "input"});
    return global.${superjsonId}.stringify(output);
}
        `;

        // Evaluate the code
        await context.eval(String.raw`${evalCode}`);

        const timeoutPromise = new Promise((_, reject) =>
            setTimeout(() => reject(new Error("Execution timed out")), jobTimeoutMs),
        );
        const result = await Promise.race([
            context.evalClosure("return execute()", [], { result: { promise: true } }),
            timeoutPromise,
        ]);
        const output = result === "undefined" ? "{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}" : result;

        postMessage({ __type: "output", output });
    } catch (error: unknown) {
        if (error instanceof Error) {
            postMessage({ __type: "error", error: error.message });
        } else {
            postMessage({ __type: "error", error: "An unknown error occurred" });
        }
    } finally {
        if (heartbeatInterval) {
            clearInterval(heartbeatInterval);
        }
        context?.release();
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

// Error handling
process.on("uncaughtException", (error) => {
    postMessage({ __type: "error", error: `Uncaught exception: ${error.message}` });
    process.exit(1);
});

process.on("unhandledRejection", (reason) => {
    postMessage({ __type: "error", error: `Unhandled rejection: ${reason}` });
    process.exit(1);
});

// Indicate readiness
postMessage({ __type: "ready" });
