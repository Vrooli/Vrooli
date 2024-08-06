// This file should only be imported by the worker thread manager
import { randomBytes } from "crypto";
import ivm from "isolated-vm";
import { parentPort } from "worker_threads";
import { DEFAULT_MEMORY_LIMIT_MB, SCRIPT_TIMEOUT_MS } from "./consts.js"; // NOTE: The extension must be specified or else it will throw an error
import { type RunUserCodeInput, type RunUserCodeOutput } from "./types.js"; // NOTE: The extension must be specified or else it will throw an error
import { getFunctionName } from "./utils.js"; // NOTE: The extension must be specified or else it will throw an error

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
    return `__ivm_${randomBytes(UNIQUE_ID_LENGTH).toString("hex")}`;
}

function postMessage(message: RunUserCodeOutput) {
    if (!parentPort) {
        throw new Error("Worker thread not available");
    }
    parentPort.postMessage(message);
}

// Listen for messages from the parent thread
parentPort.on("message", async ({ code, input }: RunUserCodeInput) => {
    if (!parentPort) {
        throw new Error("Worker thread not available");
    }

    // Throw error is code is too long
    if (code.length > MAX_CODE_LENGTH) {
        postMessage({ error: "Code is too long" });
        return;
    }

    // Extract the function name from the code
    const functionName = getFunctionName(code);
    if (!functionName) {
        postMessage({ error: "Function name not found" });
        return;
    }

    // Create sandboxed environment for code execution. 
    const isolate = new ivm.Isolate({ memoryLimit: DEFAULT_MEMORY_LIMIT_MB });


    try {
        // Create a new context within this isolate
        const context = isolate.createContextSync();

        // Get a reference to the global object within the context
        const jail = context.global;
        // This makes the global object available in the context as `global`. We use `derefInto()` here
        // because otherwise `global` would actually be a Reference{} object in the new isolate.
        jail.setSync("global", jail.derefInto());

        // Generate unique identifiers for our placeholders
        const inputId = generateUniqueId();
        const executeId = generateUniqueId();

        // Create a new instance of the input in the isolated context
        const isolatedInput = new ivm.ExternalCopy(input).copyInto();
        // Inject the input into the context
        context.global.setSync(inputId, isolatedInput);

        // Wrap code in a way that we can pass in inputs and execute it
        const wrappedCode = `
            ${code}
            function ${executeId}() {
                return ${functionName}(global['${inputId}']);
            }
        `;
        console.log("in childprocess a", wrappedCode, input);

        // Compile and run the code
        const script = isolate.compileScriptSync(wrappedCode);
        script.runSync(context);
        const output = await context.eval(`${executeId}()`, {
            timeout: SCRIPT_TIMEOUT_MS,
            arguments: { copy: true },
            result: { copy: true },
        });
        console.log("in childprocess b", output);

        // Clean up resources
        isolate.dispose();

        // Send the output back to the parent thread
        postMessage({ output });
    } catch (error: unknown) {
        if (error instanceof Error) {
            postMessage({ error: error.message });
        } else {
            postMessage({ error: "An unknown error occurred" });
        }
    }
});

// Handle errors in the worker thread
process.on("uncaughtException", (error) => {
    console.error("Uncaught Exception:", error);
    postMessage({ error: `Uncaught exception: ${error.message}` });
    process.exit(1);
});

process.on("unhandledRejection", (reason, promise) => {
    console.error("Unhandled Rejection at:", promise, "reason:", reason);
    postMessage({ error: `Unhandled rejection: ${reason}` });
    process.exit(1);
});
