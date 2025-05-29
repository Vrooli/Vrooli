import { JSDOM } from "jsdom";

const originalGlobals: { [key: string]: any } = {};

// Properties JSDOM might introduce that we'll manage.
const jsdomGlobalProperties = [
    "window", "document", "navigator", "localStorage", "sessionStorage",
    "Node", "Element", "HTMLElement", "HTMLDocument", "Event", "CustomEvent", "MouseEvent",
    "DocumentFragment", "Text", "Comment", "CDATASection", "ProcessingInstruction",
    "Attr", "NamedNodeMap", "DOMTokenList", "NodeList", "HTMLCollection",
    "URL", "URLSearchParams", "AbortController", "AbortSignal",
];

let currentDOM: JSDOM | null = null;

export function setupDOM(initialUrl = "http://localhost") {
    // If a DOM already exists from a previous unterminated test, try to clean it.
    if (currentDOM) {
        console.warn("JSDOM instance found at start of setupDOM. Tearing down previous instance.");
        teardownDOM(); // Attempt to clean up previous state
    }

    currentDOM = new JSDOM("<!doctype html><html><body></body></html>", {
        url: initialUrl,
        runScripts: "dangerously",
        pretendToBeVisual: true,
    });
    const { window: jsdomWindow } = currentDOM;

    const propertiesToManage = [...new Set([...jsdomGlobalProperties, "window", "document", "navigator", "location", "history"])];

    propertiesToManage.forEach(prop => {
        if (Object.prototype.hasOwnProperty.call(global, prop) && !Object.prototype.hasOwnProperty.call(originalGlobals, prop)) {
            originalGlobals[prop] = (global as any)[prop];
        }

        if (Object.prototype.hasOwnProperty.call(jsdomWindow, prop)) {
            Object.defineProperty(global, prop, {
                value: (jsdomWindow as any)[prop],
                writable: true,
                enumerable: true,
                configurable: true,
            });
        } else {
            if (Object.prototype.hasOwnProperty.call(global, prop) && !Object.prototype.hasOwnProperty.call(originalGlobals, prop)) {
                delete (global as any)[prop];
            }
        }
    });
}

export function teardownDOM() {
    const propertiesToCleanup = [...new Set([...jsdomGlobalProperties, "window", "document", "navigator", "location", "history"])];

    propertiesToCleanup.forEach(prop => {
        if (Object.prototype.hasOwnProperty.call(originalGlobals, prop)) {
            Object.defineProperty(global, prop, {
                value: originalGlobals[prop],
                writable: true,
                enumerable: true,
                configurable: true,
            });
        } else if (Object.prototype.hasOwnProperty.call(global, prop)) {
            // Only delete if it was not an original global property
            delete (global as any)[prop];
        }
    });

    if (currentDOM) {
        currentDOM.window.close();
        currentDOM = null;
    }

    // Clear originalGlobals for the next test.
    for (const key in originalGlobals) {
        if (Object.prototype.hasOwnProperty.call(originalGlobals, key)) {
            delete originalGlobals[key];
        }
    }
}
