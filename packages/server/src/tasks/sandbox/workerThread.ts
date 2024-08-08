// This file should only be imported by the worker thread manager
import { randomBytes } from "crypto";
import ivm from "isolated-vm";
import SuperJSON from "superjson";
import { parentPort } from "worker_threads";
import { DEFAULT_MEMORY_LIMIT_MB, SCRIPT_TIMEOUT_MS } from "./consts.js"; // NOTE: The extension must be specified or else it will throw an error
import { WorkerThreadInput, WorkerThreadOutput } from "./types.js"; // NOTE: The extension must be specified or else it will throw an error
import { bufferRegister, getFunctionDetails, urlRegister, urlWrapper } from "./utils.js"; // NOTE: The extension must be specified or else it will throw an error

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

function postMessage(message: WorkerThreadOutput) {
    if (!parentPort) {
        throw new Error("Worker thread not available");
    }
    parentPort.postMessage(message);
}

// Listen for messages from the parent thread
parentPort.on("message", ({ code, input }: WorkerThreadInput) => {
    if (!parentPort) {
        throw new Error("Worker thread not available");
    }

    // Throw error is code is too long
    if (code.length > MAX_CODE_LENGTH) {
        postMessage({ __type: "error", error: "Code is too long" });
        return;
    }

    // Extract the function name from the code
    const { functionName, isAsync } = getFunctionDetails(code);
    if (!functionName) {
        postMessage({ __type: "error", error: "Function name not found" });
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

        // Register custom SuperJSON functions for things that aren't really custom,
        // but break because of lack of support in v8.
        SuperJSON.registerCustom(urlRegister, "URL");
        SuperJSON.registerCustom(bufferRegister, "Buffer");

        // Generate unique identifiers for our placeholders
        const uniqueId = generateUniqueId();
        const inputId = `${uniqueId}Input`;
        const executeId = `${uniqueId}Execute`;
        const superjsonId = `${uniqueId}SuperJSON`;
        const urlId = `${uniqueId}URL`;
        const bufferId = `${uniqueId}Buffer`;

        parentPort.postMessage({ __type: "log", log: `Serialized input: ${input}` });
        parentPort.postMessage({ __type: "log", log: `Parsed input: ${SuperJSON.parse(input)}` });
        // Store the input (which should have been safely serialized before being sent to the worker thread)
        jail.setSync(inputId, input);

        // Need to log something in the vm? Uncomment the code below and add `log()` calls in `wrappedCode`
        jail.setSync("log", function handleVmLog(...args) {
            parentPort?.postMessage({ __type: "log", log: args.join(" ") });
        });

        // Add SuperJSON to the context so that we can parse and stringify inputs/outputs
        context.evalClosureSync(`
            global.${superjsonId} = {
                parse: $0,
                stringify: $1
            }
          `,
            [
                (...args) => {
                    return SuperJSON.parse(args[0]);
                },
                (...args) => {
                    return SuperJSON.stringify(args[0]);
                },
            ]);

        // Add additional functions to context if the code requires them
        const needsURL = code.includes("URL");
        if (needsURL) {
            context.global.setSync(urlId, new ivm.Callback(urlWrapper));

            // Create a URL class in the isolate
            context.evalSync(`
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
            `);
        }
        const needsBuffer = code.includes("Buffer");
        if (needsBuffer) {
            // Add Buffer class to the isolate
            context.evalSync(`
                class Buffer {
                    constructor(input) {
                        const bufferData = ${bufferId}(input);
                        Object.assign(this, bufferData);
                    }

                    toString(encoding = 'utf8') {
                        if (encoding === 'hex') {
                            return this.data.map(byte => byte.toString(16).padStart(2, '0')).join('');
                        }
                        // For simplicity, we'll only implement utf8 and hex encoding here.
                        // You can add more encodings as needed.
                        return String.fromCharCode.apply(null, this.data);
                    }

                    static from(data) {
                        return new Buffer(data);
                    }

                    static isBuffer(obj) {
                        return obj instanceof Buffer;
                    }
                }

                // Make Buffer available globally
                globalThis.Buffer = Buffer;
            `);
        }

        // Wrap code in a way that we can pass in inputs and execute it
        const wrappedCode = `
        ${code}
        function ${executeId}() {
            log('in executeId...', ${inputId});
            const input = ${superjsonId}.parse(${inputId});
            log('input ', input);
            const result = ${functionName}(input);
            // log('result ', JSON.stringify(result));
            return ${superjsonId}.stringify(result);
        }
    `;

        // Compile and run the code
        const script = isolate.compileScriptSync(wrappedCode);
        script.runSync(context);
        const result = context.evalSync(`${executeId}()`, {
            timeout: SCRIPT_TIMEOUT_MS,
            arguments: { copy: true },
            result: { copy: true },
        });
        parentPort.postMessage({ __type: "log", log: `Result: ${result}` });
        const output = SuperJSON.parse(result);

        // Clean up resources
        isolate.dispose();

        // Send the output back to the parent thread
        postMessage({ __type: "output", output });
    } catch (error: unknown) {
        if (error instanceof Error) {
            postMessage({ __type: "error", error: error.message });
        } else {
            postMessage({ __type: "error", error: "An unknown error occurred" });
        }
    }
});

// Handle errors in the worker thread
process.on("uncaughtException", (error) => {
    console.error("Uncaught Exception:", error);
    postMessage({ __type: "error", error: `Uncaught exception: ${error.message}` });
    process.exit(1);
});

process.on("unhandledRejection", (reason, promise) => {
    console.error("Unhandled Rejection at:", promise, "reason:", reason);
    postMessage({ __type: "error", error: `Unhandled rejection: ${reason}` });
    process.exit(1);
});
