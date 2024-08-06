// eslint-disable-next-line @typescript-eslint/no-var-requires
const path = require("path");

if (typeof setImmediate === "undefined") {
    global.setImmediate = (fn) => setTimeout(fn, 0);
}

// eslint-disable-next-line no-undef
jest.mock("worker_threads", () => {
    return require(path.resolve(__dirname, "../../src/__mocks__/worker_threads.ts"));
}, { virtual: true });
