import "@testing-library/jest-dom";

window.matchMedia = (query) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
    addListener: jest.fn(),
    removeListener: jest.fn(),
});

Object.defineProperty(URL, "createObjectURL", {
    writable: true,
    value: jest.fn(),
});

if (typeof setImmediate === "undefined") {
    global.setImmediate = setTimeout as any;
    global.clearImmediate = clearTimeout as any;
}
