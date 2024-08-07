// This file should only be imported by the worker thread manager
import { randomBytes } from "crypto";
import ivm from "isolated-vm";
import SuperJSON from "superjson";
import { parentPort } from "worker_threads";
import { DEFAULT_MEMORY_LIMIT_MB, SCRIPT_TIMEOUT_MS } from "./consts.js"; // NOTE: The extension must be specified or else it will throw an error
import { WorkerThreadInput, WorkerThreadOutput } from "./types.js"; // NOTE: The extension must be specified or else it will throw an error
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
    const functionName = getFunctionName(code);
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

        // Generate unique identifiers for our placeholders
        const uniqueId = generateUniqueId();
        const inputId = `${uniqueId}Input`;
        const executeId = `${uniqueId}Execute`;
        const stringifyId = `${uniqueId}Stringify`;
        const parseId = `${uniqueId}Parse`;
        const superjsonId = `${uniqueId}SuperJSON`;

        // Make a safe subset of the superjson module available in the VM
        const stringifyRef = new ivm.Reference(SuperJSON.stringify);
        const parseRef = new ivm.Reference(SuperJSON.parse);

        context.global.setSync(stringifyId, stringifyRef);
        context.global.setSync(parseId, parseRef);

        parentPort.postMessage({ __type: "log", log: `Serialized input: ${input}` });
        parentPort.postMessage({ __type: "log", log: `Parsed input: ${SuperJSON.parse(input)}` });
        parentPort.postMessage({ __type: "log", log: `Reserialized input: ${SuperJSON.stringify(SuperJSON.parse(input))}` });
        const parseRefTest = parseRef.applySync(undefined, [input]);
        parentPort.postMessage({ __type: "log", log: `Parse ref: ${JSON.stringify(parseRef)}` });
        parentPort.postMessage({ __type: "log", log: `Ref parsed input: ${SuperJSON.stringify(parseRefTest)}` });
        const stringifyRefTest = stringifyRef.applySync(undefined, [parseRefTest]);
        parentPort.postMessage({ __type: "log", log: `Ref stringified input: ${stringifyRefTest}` });
        // Store the input (which should have been safely serialized before being sent to the worker thread)
        jail.setSync(inputId, input);

        // const result2 = context.evalClosureSync(code, SuperJSON.parse(input), {
        //     timeout: SCRIPT_TIMEOUT_MS,
        //     arguments: { copy: true },
        //     result: { copy: true },
        // });
        // parentPort.postMessage({ __type: "log", log: `Result2: ${result2}` });

        // Wrap code in a way that we can pass in inputs and execute it
        const wrappedCode = `
        ${code}
        function ${executeId}() {
            const ${superjsonId} = {
                stringify: (...args) => ${stringifyId}.applySync(undefined, args),
                parse: (...args) => ${parseId}.applySync(undefined, args)
            };
            //const input = ${superjsonId}(${inputId});
            //const result = ${functionName}(input);
            //return ${superjsonId}.stringify(result);
            return ${superjsonId}.stringify({ chicken: "sheep" }).toString(); //.toString();
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
