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

let store: Record<string, string> = {};
const mockLocalStorage = {
    getItem(key: string) {
        return key in store ? store[key] : null;
    },
    setItem(key: string, value: string) {
        store[key] = value;
    },
    removeItem(key: string) {
        delete store[key];
    },
    clear() {
        store = {};
    },
    key(i: number) {
        const keys = Object.keys(store);
        return keys[i] || null;
    },
    get length() {
        return Object.keys(store).length;
    },
};
global.localStorage = mockLocalStorage as unknown as typeof global.localStorage;
