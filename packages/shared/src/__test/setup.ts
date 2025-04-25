import { JSDOM } from "jsdom";

// Create a basic DOM environment
const dom = new JSDOM("<!doctype html><html><body></body></html>", {
    url: "http://localhost", // Define a non-opaque origin for localStorage to work
});

// Since the shared package is used in both the server and the client,
// we let individual tests decide whether to set up a DOM environment.
export function setupDOM() {
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
}

export function teardownDOM() {
    if (global.window && typeof global.window.close === "function") {
        global.window.close();
    }
}
