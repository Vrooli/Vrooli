import { describe, expect, it } from "vitest";
import { v8CacheOptionsForProfile } from "../v8-cache-options";

describe("v8CacheOptionsForProfile", () => {
    it("uses Electron's default code cache when the optimization is off", () => {
        expect(v8CacheOptionsForProfile(false)).toBe("code");
    });

    it("selects eager code-cache compilation when the optimization is on", () => {
        expect(v8CacheOptionsForProfile(true)).toBe("bypassHeatCheckAndEagerCompile");
    });
});
