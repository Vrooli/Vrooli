// This file should only be imported by the child process manager, 
// which is responsible for managing the child process that runs user code.
import { VM } from "vm2";
import { parentPort } from "worker_threads";
import { SCRIPT_TIMEOUT_MS } from "./consts";
import { RunUserCodeInput } from "./types";

if (!parentPort) {
    throw new Error("Script must be run within a worker thread");
}

// Listen for messages from the parent thread
parentPort.on("message", ({ code, input }: RunUserCodeInput) => {
    if (!parentPort) {
        throw new Error("Worker thread not available");
    }

    // Create sandboxed environment for code execution. 
    // We can use the basic VM object because we don't need any extra features like "require".
    const vm = new VM({
        sandbox: { input },
        timeout: SCRIPT_TIMEOUT_MS, // Timeout for script execution to prevent infinite loops or recover from malicious code
    });

    try {
        // Execute the user code within the VM, where 'input' is available as a variable
        const result = vm.run(`(function(input) { ${code} })(input)`);
        parentPort.postMessage({ result });
    } catch (error: unknown) {
        if (error instanceof Error) {
            parentPort.postMessage({ error: error.message });
        } else {
            parentPort.postMessage({ error: "An unknown error occurred" });
        }
    }
});
