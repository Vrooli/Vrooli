// eslint-disable-next-line @typescript-eslint/no-var-requires
const path = require("path");

if (typeof setImmediate === "undefined") {
    global.setImmediate = (fn) => setTimeout(fn, 0);
}

// eslint-disable-next-line no-undef
jest.mock("worker_threads", () => {
    return require(path.resolve(__dirname, "../../src/__mocks__/worker_threads.ts"));
}, { virtual: true });

// eslint-disable-next-line no-undef
expect.extend({
    toBeBigInt(received, expected) {
        const pass = received === BigInt(expected);
        if (pass) {
            return {
                message: () => "",
                pass: true,
            };
        } else {
            return {
                message: () => `Received ${received}. Expected ${expected}`,
                pass: false,
            };
        }
    },
});

// Add BigInt serialization support
BigInt.prototype.toJSON = function toJSONFunc() {
    return this.toString();
};
