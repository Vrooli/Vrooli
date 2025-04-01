import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { JSDOM } from "jsdom";
import whyIsNodeRunning from "why-is-node-running";

chai.use(chaiAsPromised);

// Create a basic DOM environment
const dom = new JSDOM("<!doctype html><html><body></body></html>", {
    url: "http://localhost", // Define a non-opaque origin for localStorage to work
});

// Extend the global object type
type ExtendedGlobal = typeof globalThis & {
    window: Window | null;
    document: Document | null;
    navigator: Navigator | null;
    localStorage: Storage | null;
    sessionStorage: Storage | null;
    [key: string]: any;
};

function setupDOM() {
    // Set global variables to mimic a browser
    (global as ExtendedGlobal).window = dom.window;
    (global as ExtendedGlobal).document = dom.window.document;
    (global as ExtendedGlobal).navigator = {
        userAgent: "node.js",
    } as Navigator;
    (global as ExtendedGlobal).localStorage = dom.window.localStorage;
    (global as ExtendedGlobal).sessionStorage = dom.window.sessionStorage;

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

function teardownDOM() {
    // Store references to items we need to clean up
    const windowToClose = (global as ExtendedGlobal).window;

    // Clear all global references first
    Object.getOwnPropertyNames(global).forEach((property) => {
        if (property !== "window" && global[property] === dom.window[property]) {
            global[property] = undefined;
        }
    });

    // Clear specific globals we set by nullifying them
    (global as ExtendedGlobal).window = null as unknown as Window & typeof globalThis;
    (global as ExtendedGlobal).document = null as unknown as Document;
    (global as ExtendedGlobal).navigator = null as unknown as Navigator;
    (global as ExtendedGlobal).localStorage = null as unknown as Storage;
    (global as ExtendedGlobal).sessionStorage = null as unknown as Storage;

    // Finally close the window
    if (windowToClose && typeof windowToClose.close === "function") {
        windowToClose.close();
    }
}

before(function setup() {
    // eslint-disable-next-line no-magic-numbers
    this.timeout(10_000);

    setupDOM();
});

after(async function teardown() {
    // eslint-disable-next-line no-magic-numbers
    this.timeout(10_000);

    // First disconnect any sockets
    const SocketService = (await import("../api/socket.js")).SocketService;
    SocketService.get().disconnect();

    // Then teardown the DOM
    teardownDOM();

    // Log active handles to help debug teardown issues
    whyIsNodeRunning();
    console.info("\nIf you see more than 1 active handle and the process is hanging, you may have a teardown issue.");
});

