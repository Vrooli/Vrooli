// This file should only be imported by the worker thread manager
import { randomBytes } from "crypto";
import ivm from "isolated-vm";
import { parentPort } from "worker_threads";
import { DEFAULT_MEMORY_LIMIT_MB, SCRIPT_TIMEOUT_MS } from "./consts.js"; // NOTE: The extension must be specified or else it will throw an error
import { type RunUserCodeInput, type RunUserCodeOutput } from "./types.js"; // NOTE: The extension must be specified or else it will throw an error
import { getFunctionName, safeParse, safeStringify } from "./utils.js"; // NOTE: The extension must be specified or else it will throw an error

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

        // Safely stringify the input and store it in the context
        const safeInput = safeStringify(input);
        await jail.set(inputId, safeInput);

        // Wrap code in a way that we can pass in inputs and execute it
        const wrappedCode = `
            ${code}
            function ${executeId}() {
                const inputStr = global['${inputId}'];
                let input;
                if (inputStr === '__UNDEFINED__') {
                    input = undefined;
                } else {
                    input = JSON.parse(inputStr, (key, value) => {
                        if (typeof value === 'string' && value.startsWith('__DATE__')) {
                            return new Date(value.slice(8));
                        }
                        return value;
                    });
                }
                const result = ${functionName}(input);
                if (result === undefined) return '__UNDEFINED__';
                return JSON.stringify(result, (key, value) => {
                    if (value instanceof Date) {
                        return \`__DATE__\${value.toISOString()}\`;
                    }
                    return value;
                });
            }
        `;

        // Compile and run the code
        const script = isolate.compileScriptSync(wrappedCode);
        script.runSync(context);
        const result = await context.eval(`${executeId}()`, {
            timeout: SCRIPT_TIMEOUT_MS,
            arguments: { copy: true },
            result: { copy: true },
        });
        const output = safeParse(result);

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
