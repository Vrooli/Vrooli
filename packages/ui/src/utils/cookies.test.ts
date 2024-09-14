/* eslint-disable @typescript-eslint/ban-ts-comment */
import { noop } from "@local/shared";
import { cookies, getCookie, getOrSetCookie, getStorageItem, ifAllowed, setCookie } from "./cookies";

describe("cookies (local storage)", () => {
    beforeAll(() => {
        jest.spyOn(console, "warn").mockImplementation(noop);
    });
    beforeEach(() => {
        jest.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        });
    });
    afterAll(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
    });

    describe("getCookie", () => {
        it("should retrieve a stored value correctly", () => {
            const testKey = "Theme";
            const testValue = "light";
            setCookie(testKey, testValue);
            const result = getCookie(testKey);
            expect(result).toEqual(testValue);
        });

        it("should return undefined for non-existent keys", () => {
            // @ts-ignore: Testing runtime scenario
            const result = getCookie("nonExistentKey");
            expect(result).toBeUndefined();
        });

        it("should handle incorrect types gracefully", () => {
            const testKey = "Theme";
            const testValue = { beep: "boop" }; // Should be "light" or "dark"
            // @ts-ignore: Testing runtime scenario
            setCookie(testKey, testValue);
            const result = getCookie(testKey);
            expect(result).toBe(cookies[testKey].fallback);
        });

        it("should console.warn on JSON parse failure", () => {
            const testKey = "malformedJson";
            global.localStorage.setItem(testKey, "this is not JSON");
            getStorageItem(testKey, (value): value is object => typeof value === "object");
            expect(console.warn).toHaveBeenCalledWith(`Failed to parse cookie ${testKey}`, "this is not JSON");
        });
    });

    describe("setCookie", () => {
        it("should store stringifiable objects correctly", () => {
            const testKey = "Preferences";
            const testValue = {
                strictlyNecessary: true,
                performance: true,
                functional: true,
                targeting: false,
            };
            setCookie(testKey, testValue);
            const storedValue = JSON.parse(global.localStorage.getItem(testKey) ?? "");
            expect(storedValue).toEqual(testValue);
        });

        it("should overwrite existing values", () => {
            const testKey = "Theme";
            setCookie(testKey, "light");
            setCookie(testKey, "dark");
            const storedValue = JSON.parse(global.localStorage.getItem(testKey) ?? "");
            expect(storedValue).toEqual("dark");
        });
    });

    describe("getOrSetCookie", () => {
        it("should retrieve an existing cookie", () => {
            const testKey = "Theme";

            setCookie(testKey, "light");
            const result1 = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result1).toEqual("light");

            setCookie(testKey, "dark");
            const result2 = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result2).toEqual("dark");
        });

        it("should set the cookie to the default value if it does not exist", () => {
            const result = getOrSetCookie("defaultCookie", (value): value is number => typeof value === "number", 42);
            expect(result).toEqual(42);
            expect(localStorage.getItem("defaultCookie")).toEqual(JSON.stringify(42));
        });

        it("should set the cookie to the default value if the existing cookie fails the type check", () => {
            const testKey = "Theme";
            // @ts-ignore: Testing runtime scenario
            setCookie(testKey, 22);
            const result = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result).toEqual(cookies[testKey].fallback);
            expect(localStorage.getItem(testKey)).toEqual(JSON.stringify(cookies[testKey].fallback));
        });
    });

    describe("ifAllowed", () => {
        it("calls the callback for strictly necessary cookies regardless of preferences", () => {
            const callback = jest.fn();
            setCookie("Preferences", {
                strictlyNecessary: false,
                performance: false,
                functional: false,
                targeting: false,
            });
            ifAllowed("strictlyNecessary", callback, "fallback");
            expect(callback).toHaveBeenCalled();
        });

        it("calls the callback when preferences allow the cookie type", () => {
            const callback = jest.fn();
            setCookie("Preferences", {
                strictlyNecessary: true,
                performance: true,
                functional: false,
                targeting: false,
            });
            ifAllowed("performance", callback, "fallback");
            expect(callback).toHaveBeenCalled();
        });

        it("does not call the callback and returns fallback for disallowed cookie types", () => {
            const callback = jest.fn();
            setCookie("Preferences", {
                strictlyNecessary: true,
                performance: false,
                functional: false,
                targeting: false,
            });
            const result = ifAllowed("functional", callback, "fallback");
            expect(callback).not.toHaveBeenCalled();
            expect(result).toBe("fallback");
        });
    });
});
