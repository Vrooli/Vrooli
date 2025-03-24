import { JSDOM } from "jsdom";
import whyIsNodeRunning from "why-is-node-running";

// Create a basic DOM environment
const dom = new JSDOM("<!doctype html><html><body></body></html>", {
    url: "http://localhost", // Define a non-opaque origin for localStorage to work
});

// Set global variables to mimic a browser
global.window = dom.window;
global.document = dom.window.document;
global.navigator = {
    userAgent: "node.js",
} as Navigator;
global.localStorage = dom.window.localStorage;
global.sessionStorage = dom.window.sessionStorage;

// Attempt to copy other window properties to global,
// but skip any that are read-only.
Object.getOwnPropertyNames(dom.window).forEach((property) => {
    if (typeof global[property] === "undefined") {
        try {
            global[property] = dom.window[property];
        } catch (err) {
            // Ignore properties that cannot be assigned
        }
    }
});

after(async function teardown() {
    if (global.window && typeof global.window.close === "function") {
        global.window.close();
    }
    const SocketService = (await import("../api/socket.js")).SocketService;
    SocketService.get().disconnect();

    // Log active handles to help debug teardown issues.
    whyIsNodeRunning();
    console.info("\nIf you see more than 1 active handle and the process is hanging, you may have a teardown issue.");
});

