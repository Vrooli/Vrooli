/**
 * Simple sharp and nodejs-snowflake mocks for vitest
 */
import { vi } from "vitest";
// Mock sharp to prevent native module issues
vi.doMock("sharp", () => {
    function makeChain() {
        const chain = {};
        function pass() {
            return chain;
        }
        // Only include essential methods to avoid duplicates
        Object.assign(chain, {
            resize: pass,
            rotate: pass,
            jpeg: pass,
            png: pass,
            webp: pass,
            toBuffer: async () => Buffer.alloc(0),
            toFile: async () => ({ size: 0 }),
            metadata: async () => ({ width: 100, height: 100, format: "jpeg" }),
        });
        return chain;
    }
    function sharp(_input) {
        return makeChain();
    }
    sharp.format = {
        jpeg: { id: "jpeg" },
        png: { id: "png" },
        webp: { id: "webp" },
    };
    sharp.versions = { sharp: "0.32.6-mocked" };
    sharp.cache = vi.fn();
    sharp.concurrency = vi.fn();
    sharp.counters = vi.fn(() => ({ queue: 0, process: 0 }));
    sharp.simd = vi.fn();
    return {
        __esModule: true,
        default: sharp,
        sharp,
    };
});
// Mock nodejs-snowflake as fallback
vi.doMock("nodejs-snowflake", () => {
    class MockSnowflake {
        constructor(_options) { }
        getUniqueID() {
            return BigInt(Date.now()) * 1000n + BigInt(Math.floor(Math.random() * 1000));
        }
    }
    return {
        __esModule: true,
        default: MockSnowflake,
        Snowflake: MockSnowflake,
    };
});
console.log("[VITEST] Sharp and nodejs-snowflake modules mocked");
