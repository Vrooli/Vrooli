/* eslint-disable @typescript-eslint/ban-ts-comment */
import { CookiePreferences, Cookies, getCookie, getOrSetCookie, ifAllowed, setCookie, setCookiePreferences } from "./cookies";

describe("cookies (local storage)", () => {
    const originalLocalStorage = global.localStorage;

    // Mock localStorage
    beforeEach(() => {
        let store: Record<string, string> = {};
        const mockLocalStorage = {
            getItem: (key: string) => (key in store ? store[key] : null),
            setItem: (key: string, value: string) => (store[key] = value),
            removeItem: (key: string) => delete store[key],
            clear: () => (store = {}),
        };

        global.localStorage = mockLocalStorage as any;
        global.localStorage.clear();

    });
    afterEach(() => {
        global.localStorage = originalLocalStorage;
    });

    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });

    afterAll(() => {
        jest.restoreAllMocks();
    });

    describe("getCookie", () => {
        it("should retrieve a stored value correctly", () => {
            const testKey = "test";
            const testValue = { a: 1 };
            setCookie(testKey, testValue);
            const result = getCookie(testKey, (value): value is typeof testValue => typeof value === "object" && value !== null && "a" in value);
            expect(result).toEqual(testValue);
        });

        it("should return undefined for non-existent keys", () => {
            const result = getCookie("nonExistentKey", (value): value is number => typeof value === "number");
            expect(result).toBeUndefined();
        });

        it("should handle incorrect types gracefully", () => {
            const testKey = "incorrectType";
            const testValue = "this is a string";
            setCookie(testKey, testValue);
            const result = getCookie(testKey, (value): value is number => typeof value === "number");
            expect(result).toBeUndefined();
        });

        it("should console.warn on JSON parse failure", () => {
            const testKey = "malformedJson";
            global.localStorage.setItem(testKey, "this is not JSON");
            getCookie(testKey, (value): value is object => typeof value === "object");
            expect(console.warn).toHaveBeenCalledWith(`Failed to parse cookie ${testKey}`, "this is not JSON");
        });
    });

    describe("setCookie", () => {
        it("should store stringifiable objects correctly", () => {
            const testKey = "objectTest";
            const testValue = { hello: "world" };
            setCookie(testKey, testValue);
            const storedValue = JSON.parse(global.localStorage.getItem(testKey) ?? "");
            expect(storedValue).toEqual(testValue);
        });

        it("should overwrite existing values", () => {
            const testKey = "overwriteTest";
            setCookie(testKey, { first: "value" });
            setCookie(testKey, { second: "value" });
            const storedValue = JSON.parse(global.localStorage.getItem(testKey) ?? "");
            expect(storedValue).toEqual({ second: "value" });
        });
    });

    describe("getOrSetCookie", () => {
        it("should retrieve an existing cookie", () => {
            setCookie("existingCookie", "existingValue");
            const result = getOrSetCookie("existingCookie", (value): value is string => typeof value === "string");
            expect(result).toEqual("existingValue");
        });

        it("should set the cookie to the default value if it does not exist", () => {
            const result = getOrSetCookie("defaultCookie", (value): value is number => typeof value === "number", 42);
            expect(result).toEqual(42);
            expect(localStorage.getItem("defaultCookie")).toEqual(JSON.stringify(42));
        });

        it("should not set a cookie if the default value is undefined and the cookie does not exist", () => {
            const result = getOrSetCookie("undefinedCookie", (value): value is undefined => typeof value === "undefined");
            expect(result).toBeUndefined();
            expect(localStorage.getItem("undefinedCookie")).toBeNull();
        });

        it("should set the cookie to the default value if the existing cookie fails the type check", () => {
            setCookie("typedCookie", "string");
            const result = getOrSetCookie("typedCookie", (value): value is number => typeof value === "number", 100);
            expect(result).toEqual(100);
            expect(localStorage.getItem("typedCookie")).toEqual(JSON.stringify(100));
        });
    });

    describe("setCookiePreferences", () => {
        it("should store valid cookie preferences", () => {
            const validPreferences = {
                strictlyNecessary: true,
                performance: false,
                functional: true,
                targeting: false,
            };
            setCookiePreferences(validPreferences);
            const storedPreferences = getCookie(Cookies.Preferences, (value): value is CookiePreferences => typeof value === "object");
            expect(storedPreferences).toEqual(validPreferences);
        });

        it("should ignore additional properties", () => {
            const preferencesWithExtras = {
                strictlyNecessary: true,
                performance: false,
                functional: true,
                targeting: false,
                extra: "should be ignored",
            };
            // @ts-ignore: Testing runtime scenario
            setCookiePreferences(preferencesWithExtras);
            const storedPreferences = getCookie(Cookies.Preferences, (value): value is CookiePreferences => typeof value === "object");
            expect(storedPreferences).toEqual({
                strictlyNecessary: true,
                performance: false,
                functional: true,
                targeting: false,
            });
        });

        it("should handle invalid inputs: missing fields", () => {
            const incompletePreferences = {}; // Missing all required fields
            // @ts-ignore: Testing runtime scenario
            setCookiePreferences(incompletePreferences);
            const storedPreferences = getCookie(Cookies.Preferences, (value): value is CookiePreferences => typeof value === "object");
            // Should have defaulted to all fields being false
            expect(storedPreferences).toEqual({
                strictlyNecessary: false,
                performance: false,
                functional: false,
                targeting: false,
            });
        });
    });

    describe("ifAllowed", () => {
        it("calls the callback for strictly necessary cookies regardless of preferences", () => {
            const callback = jest.fn();
            setCookiePreferences({
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
            setCookiePreferences({
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
            setCookiePreferences({
                strictlyNecessary: true,
                performance: false,
                functional: false,
                targeting: false,
            });
            const result = ifAllowed("functional", callback, "fallback");
            expect(callback).not.toHaveBeenCalled();
            expect(console.warn).toHaveBeenCalledWith("Not allowed to get/set cookie functional");
            expect(result).toBe("fallback");
        });
    });
});
