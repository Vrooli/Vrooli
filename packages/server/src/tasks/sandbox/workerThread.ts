// This file should only be imported by the worker thread manager
// import ivm from "isolated-vm";
import { parentPort } from "worker_threads";
// import { DEFAULT_MEMORY_LIMIT_MB, SCRIPT_TIMEOUT_MS } from "./consts.js"; // NOTE: The extension must be specified or else it will throw an error
import { type RunUserCodeInput } from "./types.js"; // NOTE: The extension must be specified or else it will throw an error
// import { getFunctionName } from "./utils.js"; // NOTE: The extension must be specified or else it will throw an error

const MAX_CODE_LENGTH = 8_192; // Maximum length of code that can be executed

if (!parentPort) {
    throw new Error("Script must be run within a worker thread");
}

// Listen for messages from the parent thread
parentPort.on("message", async ({ code, input }: RunUserCodeInput) => {
    //TODO for testing purposes, return the input instead of running the code
    console.log("in childprocess", code, input);
    parentPort!.postMessage({ result: input });

    // console.log("in childprocess a", code, input);
    // if (!parentPort) {
    //     throw new Error("Worker thread not available");
    // }
    // console.log("in childprocess b");

    // // Throw error is code is too long
    // if (code.length > MAX_CODE_LENGTH) {
    //     parentPort.postMessage({ error: "Code is too long" });
    //     return;
    // }

    // // Extract the function name from the code
    // const functionName = getFunctionName(code);
    // if (!functionName) {
    //     parentPort.postMessage({ error: "Function name not found" });
    //     return;
    // }

    // // Create sandboxed environment for code execution. 
    // const isolate = new ivm.Isolate({ memoryLimit: DEFAULT_MEMORY_LIMIT_MB });
    // console.log("in childprocess c");


    // try {
    //     console.log("running vm");
    //     // Create a new context within this isolate
    //     const context = isolate.createContextSync();
    //     // Get a reference to the global object within the context
    //     const jail = context.global;
    //     // This makes the global object available in the context as `global`. We use `derefInto()` here
    //     // because otherwise `global` would actually be a Reference{} object in the new isolate.
    //     jail.setSync("global", jail.derefInto());
    //     // Append `main = ${functionName}` to the code so that the user function can be called
    //     const wrappedCode = `${code}\nmain = ${functionName}`;
    //     // Compile and run the code
    //     const script = isolate.compileScriptSync(wrappedCode);
    //     script.runSync(context);
    //     // Invoke "main" function with the input
    //     const result = context.evalClosureSync("main($0)", [input], {
    //         arguments: { copy: true },
    //         timeout: SCRIPT_TIMEOUT_MS,
    //     });
    //     // Clean up resources
    //     isolate.dispose();
    //     console.log("in childprocess d", result);
    //     parentPort.postMessage({ result });
    //     console.log("in childprocess e");
    // } catch (error: unknown) {
    //     if (error instanceof Error) {
    //         parentPort.postMessage({ error: error.message });
    //     } else {
    //         parentPort.postMessage({ error: "An unknown error occurred" });
    //     }
    // }
});

// Handle errors in the worker thread
process.on("uncaughtException", (error) => {
    console.error("Uncaught Exception:", error);
    if (parentPort) {
        parentPort.postMessage({ error: `Uncaught exception: ${error.message}` });
    }
    process.exit(1);
});

process.on("unhandledRejection", (reason, promise) => {
    console.error("Unhandled Rejection at:", promise, "reason:", reason);
    if (parentPort) {
        parentPort.postMessage({ error: `Unhandled rejection: ${reason}` });
    }
    process.exit(1);
});
