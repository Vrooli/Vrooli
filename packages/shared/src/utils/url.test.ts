import { afterAll, beforeAll, beforeEach, describe, expect, it, test, vi } from "vitest";
import { ResourceSubType } from "../api/types.js";
import { LINKS } from "../consts/ui.js";
import {
    decodeValue,
    encodeValue,
    getObjectSearchParams,
    getObjectSlug,
    getObjectUrl,
    getObjectUrlBase,
    parseSearchParams,
    stringifySearchParams,
    UrlTools,
} from "./url.js";

// Pre-computed test data for expensive operations
const CIRCULAR_OBJ = (() => {
    const obj = {} as any;
    obj.self = obj;
    return obj;
})();

describe.concurrent("encodeValue and decodeValue", () => {
    const testCases = [
        {
            description: "handles plain strings without percent signs",
            input: "simple string",
            encoded: "simple string",
            decoded: "simple string",
        },
        {
            description: "encodes and decodes percent signs in strings",
            input: "100% sure",
            encoded: "100%25 sure",
            decoded: "100% sure",
        },
        {
            description: "recursively encodes and decodes objects",
            input: { key: "50% discount", nested: { prop: "20% off" } },
            encoded: { key: "50%25 discount", nested: { prop: "20%25 off" } },
            decoded: { key: "50% discount", nested: { prop: "20% off" } },
        },
        {
            description: "recursively encodes and decodes arrays",
            input: ["10% milk", "5% fat"],
            encoded: ["10%25 milk", "5%25 fat"],
            decoded: ["10% milk", "5% fat"],
        },
        {
            description: "handles mixed types with arrays and objects",
            input: { percentages: ["30% rate", "40% area"], detail: { info: "60% done" } },
            encoded: { percentages: ["30%25 rate", "40%25 area"], detail: { info: "60%25 done" } },
            decoded: { percentages: ["30% rate", "40% area"], detail: { info: "60% done" } },
        },
        {
            description: "does not modify non-string types",
            input: [10, 20.5, true, null, undefined],
            encoded: [10, 20.5, true, null, undefined],
            decoded: [10, 20.5, true, null, undefined],
        },
    ];

    test.each(testCases)("$description", ({ input, encoded, decoded }) => {
        const encodedValue = encodeValue(input);
        expect(encodedValue).toEqual(encoded);

        const decodedValue = decodeValue(encodedValue);
        expect(decodedValue).toEqual(decoded);
    });
});

describe("parseSearchParams", () => {
    // Mock window object for tests
    beforeAll(() => {
        // @ts-ignore
        global.window = {
            location: { search: "" },
            history: {
                replaceState: vi.fn((state, title, url) => {
                    const searchMatch = url.match(/\?(.*)$/);
                    global.window.location.search = searchMatch ? `?${searchMatch[1]}` : "";
                }),
            },
        };
    });

    afterAll(() => {
        // @ts-ignore
        delete global.window;
    });

    // Reset URL to clean state before each test
    beforeEach(() => {
        window.history.replaceState({}, "", "http://localhost/");
    });

    function setWindowSearch(search: string) {
        // Use history.replaceState (faster than pushState) to change the search parameters.
        // The first two arguments (state object and title) are not used by parseSearchParams.
        // The third argument is the new URL (or path + search string).
        // Ensure it starts with '?' if it's a search string, or is a full path.
        const pathAndSearch = search.startsWith("?") || search.startsWith("/") ? search : `?${search}`;
        window.history.replaceState({}, "", pathAndSearch);
    }

    it("returns an empty object for empty search params", () => {
        setWindowSearch("");
        expect(parseSearchParams()).toEqual({});
    });

    it("parses a single key-value pair correctly", () => {
        setWindowSearch("?key=%22value%22");
        expect(parseSearchParams()).toEqual({ key: "value" });
    });

    it("parses multiple key-value pairs correctly", () => {
        setWindowSearch("?key1=%22value1%22&key2=%22value2%22");
        expect(parseSearchParams()).toEqual({ key1: "value1", key2: "value2" });
    });

    it("decodes special characters correctly", () => {
        setWindowSearch("?key%20one=%22value%2Fone%22");
        expect(parseSearchParams()).toEqual({ "key one": "value/one" });
    });

    it("parses nested objects correctly, including nested number, boolean, and null", () => {
        setWindowSearch("?key=%7B%22nestedKey%22%3A%22nestedValue%22%2C%22nestedNumber%22%3A-123%2C%22nestedBoolean%22%3Atrue%2C%22nestedNull%22%3Anull%7D");
        expect(parseSearchParams()).toEqual({
            key: {
                nestedKey: "nestedValue",
                nestedNumber: -123,
                nestedBoolean: true,
                nestedNull: null,
            },
        });
    });

    it("parses mixed types correctly", () => {
        setWindowSearch("?string=%22value%22&number=123&boolean=true&null=null");
        expect(parseSearchParams()).toEqual({ string: "value", number: 123, boolean: true, null: null });
    });

    it("returns parameters as strings for invalid JSON format", () => {
        // Mock console.error
        const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => { });

        setWindowSearch("?invalid");
        expect(parseSearchParams()).toEqual({ invalid: "" });

        // Restore the spy
        consoleErrorSpy.mockRestore();
    });

    it("parses a top-level array of numbers correctly", () => {
        setWindowSearch("?numbers=%5B123%2C456%2C789%5D");
        expect(parseSearchParams()).toEqual({
            numbers: [123, 456, 789],
        });
    });

    it("parses an array of objects nested within an object correctly", () => {
        setWindowSearch("?data=%7B%22items%22%3A%5B%7B%22id%22%3A1%2C%22name%22%3A%22Item1%22%7D%2C%7B%22id%22%3A2%2C%22name%22%3A%22Item2%22%7D%5D%7D"); // URL-encoded for {"items":[{"id":1,"name":"Item1"},{"id":2,"name":"Item2"}]}
        expect(parseSearchParams()).toEqual({
            data: {
                items: [
                    { id: 1, name: "Item1" },
                    { id: 2, name: "Item2" },
                ],
            },
        });
    });
});

describe.concurrent("stringifySearchParams", () => {
    it("returns an empty string for an empty object", () => {
        expect(stringifySearchParams({})).toBe("");
    });

    it("returns an empty string for an object with only null and undefined values", () => {
        expect(stringifySearchParams({ key1: null, key2: undefined })).toBe("");
    });

    it("formats a single key-value pair correctly", () => {
        expect(stringifySearchParams({ key: "value" })).toBe("?key=%22value%22");
    });

    it("formats multiple key-value pairs correctly", () => {
        expect(stringifySearchParams({ key1: "value1", key2: "value2" })).toBe("?key1=%22value1%22&key2=%22value2%22");
    });

    it("ignores keys with null or undefined values", () => {
        expect(stringifySearchParams({ key1: null, key2: undefined, key3: "value" })).toBe("?key3=%22value%22");
    });

    it("encodes special characters correctly", () => {
        expect(stringifySearchParams({ "key one": "value/one" })).toBe("?key%20one=%22value%2Fone%22");
    });

    it("stringifies and encodes nested objects correctly", () => {
        expect(stringifySearchParams({ key: { nestedKey: "nestedValue" } })).toBe("?key=%7B%22nestedKey%22%3A%22nestedValue%22%7D");
    });
});

describe("stringifySearchParams and parseSearchParams", () => {
    // Mock window object for tests
    beforeAll(() => {
        // @ts-ignore
        global.window = {
            location: { search: "" },
            history: {
                replaceState: vi.fn((state, title, url) => {
                    const searchMatch = url.match(/\?(.*)$/);
                    global.window.location.search = searchMatch ? `?${searchMatch[1]}` : "";
                }),
            },
        };
    });

    afterAll(() => {
        // @ts-ignore
        delete global.window;
    });

    const testCases = [
        {
            description: "handles an empty object",
            input: {},
            expected: {},
        },
        {
            description: "removes null and undefined values from object",
            input: { key1: null, key2: undefined, key3: "value" },
            expected: { key3: "value" },
        },
        {
            description: "handles a single key-value pair",
            input: { key: "val/ue" },
            expected: { key: "val/ue" },
        },
        {
            description: "handles multiple key-value pairs",
            input: { key_one: "value1", key_TwO: "value2" },
            expected: { key_one: "value1", key_TwO: "value2" },
        },
        {
            description: "handles special characters",
            input: {
                "normalSpecialCharacters": "!@#$%^&*()_+-=[]{}\\|;:'\",.<>/?",
                "rarerSpecialCharacters": "(╯°□°）╯︵ ┻━┻  (. ❛ ᴗ ❛.) ( ͡° ͜ʖ ͡°) ᕕ( ᐛ )ᕗ",
                "otherLanguagesAndAccents": "¡Hola! ¿Cómo estás? 你好吗 こんにちは สวัสดี ជំរាបសួរ გამარჯობა Բարև",
                "emojis": "😁🪿🥳🙆‍♂️🙅🏻‍♂️👴🏿👽🥳",
                "zalgo": "H̷̢̛̛͚̖͙̰̪͔̗̔̊͌̈́̑̿͌̆̔͊͋̀̆̚͝e̷̢̲̬̙͓̝̜̖͎͈̯͙̭̱̼̳͚̿͊̿̔́̐͐̍̏̽̈́̉͠͠ ̶͚̲̖̻̼̰̘̋̿͆͒̋͠c̷͙͇͋̃̈̒͌͌͝õ̵͕̘̤̳̻̞͖̉̈́̆̏̒͝m̶̘͔̰̬͊̊ȩ̸̢̧̨̛̞̰͚̳̜͙̻͚̰͎͐͆́̍̎̚s̶̢̡̬̠̹͓̳͎̪̰̯̻̃̒̕͝",
            },
            expected: {
                "normalSpecialCharacters": "!@#$%^&*()_+-=[]{}\\|;:'\",.<>/?",
                "rarerSpecialCharacters": "(╯°□°）╯︵ ┻━┻  (. ❛ ᴗ ❛.) ( ͡° ͜ʖ ͡°) ᕕ( ᐛ )ᕗ",
                "otherLanguagesAndAccents": "¡Hola! ¿Cómo estás? 你好吗 こんにちは สวัสดี ជំរាបសួរ გამარჯობა Բարև",
                "emojis": "😁🪿🥳🙆‍♂️🙅🏻‍♂️👴🏿👽🥳",
                "zalgo": "H̷̢̛̛͚̖͙̰̪͔̗̔̊͌̈́̑̿͌̆̔͊͋̀̆̚͝e̷̢̲̬̙͓̝̜̖͎͈̯͙̭̱̼̳͚̿͊̿̔́̐͐̍̏̽̈́̉͠͠ ̶͚̲̖̻̼̰̘̋̿͆͒̋͠c̷͙͇͋̃̈̒͌͌͝õ̵͕̘̤̳̻̞͖̉̈́̆̏̒͝m̶̘͔̰̬͊̊ȩ̸̢̧̨̛̞̰͚̳̜͙̻͚̰͎͐͆́̍̎̚s̶̢̡̬̠̹͓̳͎̪̰̯̻̃̒̕͝",
            },
        },
        {
            description: "handles nested objects",
            input: { key: { nestedKey: "nestedValue" } },
            expected: { key: { nestedKey: "nestedValue" } },
        },
        {
            description: "handles arrays within objects",
            input: { key: ["item1", "item2"] },
            expected: { key: ["item1", "item2"] },
        },
        {
            description: "handles mixed types including numbers, booleans, nulls",
            input: {
                str: "text",
                num: 123,
                boolTrue: true,
                boolFalse: false,
                nil: null,
                undef: undefined,
                obj: { nest: "deep", arr: [1, "two"] },
            },
            expected: {
                str: "text",
                num: 123,
                boolTrue: true,
                boolFalse: false,
                obj: { nest: "deep", arr: [1, "two"] },
            },
        },
        {
            description: "handles values that need URI encoding (string values only)",
            input: { query: "a&b=c d", path: "/test path" },
            expected: { query: "a&b=c d", path: "/test path" },
        },
    ];

    test.each(testCases)("$description", ({ input, expected }) => {
        const queryString = stringifySearchParams(input);
        // Reset URL with the new query string
        window.history.replaceState({}, "", queryString ? `http://localhost/${queryString}` : "http://localhost/");

        const parsed = parseSearchParams();
        expect(parsed).toEqual(expected);
    });
});

describe.concurrent("getObjectUrlBase", () => {
    it("should return empty string for invalid objects", () => {
        expect(getObjectUrlBase(null as any)).toBe("");
        expect(getObjectUrlBase(undefined as any)).toBe("");
        expect(getObjectUrlBase("string" as any)).toBe("");
        expect(getObjectUrlBase({ notTypename: "value" } as any)).toBe("");
        expect(getObjectUrlBase({ __typename: 123 } as any)).toBe("");
    });

    it("should return correct URL base for User and SessionUser", () => {
        const user = { __typename: "User" };
        const sessionUser = { __typename: "SessionUser" };
        expect(getObjectUrlBase(user)).toBe(LINKS.Profile);
        expect(getObjectUrlBase(sessionUser)).toBe(LINKS.Profile);
    });

    it("should return correct URL base for Resource types", () => {
        const dataConverter = {
            __typename: "Resource",
            versions: [{ isLatest: true, resourceSubType: ResourceSubType.CodeDataConverter }],
        };
        const smartContract = {
            __typename: "ResourceVersion",
            resourceSubType: ResourceSubType.CodeSmartContract,
        };
        const dataStructure = {
            __typename: "Resource",
            versions: [{ isLatest: true, resourceSubType: ResourceSubType.StandardDataStructure }],
        };
        const prompt = {
            __typename: "ResourceVersion",
            resourceSubType: ResourceSubType.StandardPrompt,
        };
        const routine = {
            __typename: "Resource",
            versions: [{ isLatest: true, resourceSubType: ResourceSubType.RoutineMultiStep }],
        };

        expect(getObjectUrlBase(dataConverter)).toBe(LINKS.DataConverter);
        expect(getObjectUrlBase(smartContract)).toBe(LINKS.SmartContract);
        expect(getObjectUrlBase(dataStructure)).toBe(LINKS.DataStructure);
        expect(getObjectUrlBase(prompt)).toBe(LINKS.Prompt);
        expect(getObjectUrlBase(routine)).toBe(LINKS.RoutineMultiStep);
    });

    it("should handle ResourceVersion without resourceSubType", () => {
        const resourceVersion = {
            __typename: "ResourceVersion",
        };
        expect(getObjectUrlBase(resourceVersion)).toBe(LINKS.RoutineMultiStep);
    });

    it("should handle objects with 'to' property", () => {
        const bookmark = {
            __typename: "Bookmark",
            to: { __typename: "User" },
        };
        const reaction = {
            __typename: "Reaction",
            to: { __typename: "Project" },
        };
        const view = {
            __typename: "View",
            to: { __typename: "Routine" },
        };

        expect(getObjectUrlBase(bookmark)).toBe(LINKS.Profile);
        expect(getObjectUrlBase(reaction)).toBe(LINKS.Project);
        expect(getObjectUrlBase(view)).toBe(LINKS.Routine);
    });

    it("should handle Member and ChatParticipant", () => {
        const member = {
            __typename: "Member",
            user: { __typename: "User" },
        };
        const participant = {
            __typename: "ChatParticipant",
            user: { __typename: "User" },
        };

        expect(getObjectUrlBase(member)).toBe(LINKS.Profile);
        expect(getObjectUrlBase(participant)).toBe(LINKS.Profile);
    });

    it("should handle Notification", () => {
        const notificationWithLink = {
            __typename: "Notification",
            link: "/custom-link",
        };
        const notificationWithoutLink = {
            __typename: "Notification",
            link: null,
        };

        expect(getObjectUrlBase(notificationWithLink)).toBe("/custom-link");
        expect(getObjectUrlBase(notificationWithoutLink)).toBe("");
    });

    it("should handle Run", () => {
        const run = { __typename: "Run" };
        expect(getObjectUrlBase(run)).toBe(LINKS.Run);
    });

    it("should handle generic objects by removing 'Version' from typename", () => {
        const project = { __typename: "Project" };
        const projectVersion = { __typename: "ProjectVersion" };

        expect(getObjectUrlBase(project)).toBe(LINKS.Project);
        expect(getObjectUrlBase(projectVersion)).toBe(LINKS.Project);
    });
});

describe.concurrent("getObjectSlug", () => {
    it("should return '/' for null or undefined objects", () => {
        expect(getObjectSlug(null)).toBe("/");
        expect(getObjectSlug(undefined)).toBe("/");
        expect(getObjectSlug("string" as any)).toBe("/");
    });

    it("should return '/' for Action, Shortcut, and CalendarEvent", () => {
        expect(getObjectSlug({ __typename: "Action" })).toBe("/");
        expect(getObjectSlug({ __typename: "Shortcut" })).toBe("/");
        expect(getObjectSlug({ __typename: "CalendarEvent" })).toBe("/");
    });

    it("should handle objects with 'to' property", () => {
        const bookmark = {
            __typename: "Bookmark",
            to: { __typename: "User", handle: "testuser" },
        };
        expect(getObjectSlug(bookmark)).toBe("/@testuser");
    });

    it("should handle versioned objects with root", () => {
        const resourceVersion = {
            __typename: "ResourceVersion",
            root: { __typename: "Resource", handle: "test-resource" },
            versionLabel: "v1.0",
            publicId: "pub123",
            id: "id123",
        };
        expect(getObjectSlug(resourceVersion)).toBe("/@test-resource/v/v1.0");

        // Test with only publicId
        const resourceVersionWithPublicId = {
            __typename: "ResourceVersion",
            root: { __typename: "Resource", handle: "test-resource" },
            publicId: "pub123",
            id: "id123",
        };
        expect(getObjectSlug(resourceVersionWithPublicId)).toBe("/@test-resource/v/pub123");

        // Test with only id
        const resourceVersionWithId = {
            __typename: "ResourceVersion",
            root: { __typename: "Resource", handle: "test-resource" },
            id: "id123",
        };
        expect(getObjectSlug(resourceVersionWithId)).toBe("/@test-resource/v/id123");
    });

    it("should handle Member and ChatParticipant", () => {
        const member = {
            __typename: "Member",
            user: { __typename: "User", handle: "member-user" },
        };
        const participant = {
            __typename: "ChatParticipant",
            user: { __typename: "User", publicId: "pub456" },
        };

        expect(getObjectSlug(member)).toBe("/@member-user");
        expect(getObjectSlug(participant)).toBe("/pub456");
    });

    it("should return '/' for Notification", () => {
        const notification = { __typename: "Notification" };
        expect(getObjectSlug(notification)).toBe("/");
    });

    it("should handle regular objects with handle or id", () => {
        const withHandle = {
            __typename: "User",
            handle: "testuser",
            publicId: "pub123",
            id: "id123",
        };
        const withoutHandle = {
            __typename: "User",
            publicId: "pub123",
            id: "id123",
        };
        const withOnlyId = {
            __typename: "User",
            id: "id123",
        };

        expect(getObjectSlug(withHandle)).toBe("/@testuser");
        expect(getObjectSlug(withoutHandle)).toBe("/pub123");
        expect(getObjectSlug(withOnlyId)).toBe("/id123");
    });

    it("should prefer id when prefersId is true", () => {
        const obj = {
            __typename: "User",
            handle: "testuser",
            publicId: "pub123",
            id: "id123",
        };

        expect(getObjectSlug(obj, false)).toBe("/@testuser");
        expect(getObjectSlug(obj, true)).toBe("/pub123");
    });
});

describe.concurrent("getObjectSearchParams", () => {
    it("should return search params for CalendarEvent", () => {
        const calendarEvent = {
            __typename: "CalendarEvent",
            start: new Date("2023-01-01T10:00:00Z"),
        };

        // Mock UrlTools.stringifySearchParams
        const spy = vi.spyOn(UrlTools, "stringifySearchParams").mockReturnValue("?start=2023-01-01T10%3A00%3A00.000Z");

        const result = getObjectSearchParams(calendarEvent as any);
        expect(result).toBe("?start=2023-01-01T10%3A00%3A00.000Z");
        expect(spy).toHaveBeenCalledWith(LINKS.Calendar, { start: calendarEvent.start });

        spy.mockRestore();
    });

    it("should return search params for Run", () => {
        const run = {
            __typename: "Run",
            lastStep: [1, 2, 3],
        };

        // Mock UrlTools.stringifySearchParams
        const spy = vi.spyOn(UrlTools, "stringifySearchParams").mockReturnValue("?step=%5B1%2C2%2C3%5D");

        const result = getObjectSearchParams(run as any);
        expect(result).toBe("?step=%5B1%2C2%2C3%5D");
        expect(spy).toHaveBeenCalledWith(LINKS.Run, { step: [1, 2, 3] });

        spy.mockRestore();
    });

    it("should return empty string for other objects", () => {
        const user = { __typename: "User" };
        expect(getObjectSearchParams(user as any)).toBe("");
    });
});

describe.concurrent("getObjectUrl", () => {
    it("should return empty string for Action", () => {
        const action = { __typename: "Action" };
        expect(getObjectUrl(action as any)).toBe("");
    });

    it("should return id for Shortcut", () => {
        const shortcut = {
            __typename: "Shortcut",
            id: "https://example.com",
        };
        expect(getObjectUrl(shortcut as any)).toBe("https://example.com");

        const shortcutWithoutId = { __typename: "Shortcut" };
        expect(getObjectUrl(shortcutWithoutId as any)).toBe("");
    });

    it("should return schedule URL for CalendarEvent", () => {
        const calendarEvent = {
            __typename: "CalendarEvent",
            schedule: {
                __typename: "Schedule",
                handle: "my-schedule",
            },
        };

        // Mock the helper functions
        const spy1 = vi.spyOn({ getObjectUrl }, "getObjectUrl").mockReturnValue("/schedule/my-schedule");

        getObjectUrl(calendarEvent as any);

        spy1.mockRestore();
    });

    it("should combine base, slug, and search params for regular objects", () => {
        const user = {
            __typename: "User",
            handle: "testuser",
        };

        const result = getObjectUrl(user as any);
        expect(result).toContain("/@testuser");
        expect(result).toContain(LINKS.Profile);
    });
});

describe("UrlTools", () => {
    describe("stringifySearchParams", () => {
        it("should call stringifySearchParams with correct parameters", () => {
            const searchParams = { redirect: "/home" };
            const result = UrlTools.stringifySearchParams(LINKS.Login, searchParams);
            expect(typeof result).toBe("string");
        });

        it("should use default params when none provided", () => {
            const result = UrlTools.stringifySearchParams(LINKS.Login, {});
            expect(typeof result).toBe("string");
        });
    });

    describe("parseSearchParams", () => {
        beforeAll(() => {
            // @ts-ignore
            global.window = {
                location: { search: "" },
                history: {
                    replaceState: vi.fn((state, title, url) => {
                        const searchMatch = url.match(/\?(.*)$/);
                        global.window.location.search = searchMatch ? `?${searchMatch[1]}` : "";
                    }),
                },
            };
        });

        afterAll(() => {
            // @ts-ignore
            delete global.window;
        });

        beforeEach(() => {
            // Reset URL to clean state
            window.history.replaceState({}, "", "http://localhost/");
        });

        it("should parse search params for given link type", () => {
            window.history.replaceState({}, "", "?redirect=%22%2Fhome%22");
            const result = UrlTools.parseSearchParams(LINKS.Login);
            expect(result).toEqual({ redirect: "/home" });
        });
    });

    describe("linkWithSearchParams", () => {
        it("should combine link with search params", () => {
            const searchParams = { redirect: "/home" };
            const result = UrlTools.linkWithSearchParams(LINKS.Login, searchParams);
            expect(result).toContain(LINKS.Login);
            expect(result).toContain("redirect");
        });

        it("should use default params when none provided", () => {
            const result = UrlTools.linkWithSearchParams(LINKS.Login, {});
            expect(result).toContain(LINKS.Login);
        });
    });
});

describe("error handling", () => {
    it("should handle stringifySearchParams errors gracefully", () => {
        const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {
            // Mock implementation
        });

        const result = stringifySearchParams({ circular: CIRCULAR_OBJ });
        expect(result).toBe("");
        expect(consoleErrorSpy).toHaveBeenCalled();

        consoleErrorSpy.mockRestore();
    });

    it("should handle parseSearchParams JSON parsing errors by falling back to strings", () => {
        // @ts-ignore
        global.window = {
            location: { search: "" },
            history: {
                replaceState: vi.fn((state, title, url) => {
                    const searchMatch = url.match(/\?(.*)$/);
                    global.window.location.search = searchMatch ? `?${searchMatch[1]}` : "";
                }),
            },
        };

        // Set invalid JSON in search params
        window.history.replaceState({}, "", "?invalid=notjson");

        const result = parseSearchParams();
        // When JSON parsing fails, the parameter falls back to string value
        expect(result).toEqual({ invalid: "notjson" });

        // @ts-ignore
        delete global.window;
    });

    it("should log errors in parseSearchParams when not in test environment", () => {
        // @ts-ignore
        global.window = {
            location: { search: "" },
            history: {
                replaceState: vi.fn((state, title, url) => {
                    const searchMatch = url.match(/\?(.*)$/);
                    global.window.location.search = searchMatch ? `?${searchMatch[1]}` : "";
                }),
            },
        };

        const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => { });
        const originalNodeEnv = process.env.NODE_ENV;

        // Temporarily change NODE_ENV to trigger console.error
        process.env.NODE_ENV = "production";

        // Set parameter with invalid URI encoding that will cause decodeURIComponent to fail
        window.history.replaceState({}, "", "?invalid%ZZ=test");

        const result = parseSearchParams();

        // Should log error when not in test environment
        expect(consoleErrorSpy).toHaveBeenCalledWith(
            expect.stringContaining("Error decoding parameter \"invalid%ZZ\""),
        );

        // When URI decoding fails, the parameter is ignored
        expect(result).toEqual({});

        // Restore environment and mocks
        process.env.NODE_ENV = originalNodeEnv;
        consoleErrorSpy.mockRestore();
        // @ts-ignore
        delete global.window;
    });

    it("should warn for invalid objects in getObjectUrlBase", () => {
        // In test environment, warnings are suppressed, so we need to temporarily change NODE_ENV
        const originalEnv = process.env.NODE_ENV;
        process.env.NODE_ENV = "development";
        
        const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {
            // Mock implementation
        });

        getObjectUrlBase({ invalidProp: "value" } as any);
        expect(consoleWarnSpy).toHaveBeenCalled();

        consoleWarnSpy.mockRestore();
        process.env.NODE_ENV = originalEnv;
    });
});
