import { base36ToUuid, decodeValue, encodeValue, parseSearchParams, stringifySearchParams, uuidToBase36 } from "./url";

describe("encodeValue and decodeValue", () => {
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

    testCases.forEach(({ description, input, encoded, decoded }) => {
        test(description, () => {
            const encodedValue = encodeValue(input);
            expect(encodedValue).toEqual(encoded);

            const decodedValue = decodeValue(encodedValue);
            expect(decodedValue).toEqual(decoded);
        });
    });
});

describe("parseSearchParams", () => {
    // Save the original window.location
    const originalLocation = window.location;

    // Helper function to set window.location.search
    function setWindowSearch(search) {
        Object.defineProperty(window, "location", {
            value: {
                search,
            },
            writable: true,
        });
    }

    afterEach(() => {
        // Restore the original window.location after each test
        Object.defineProperty(window, "location", {
            value: originalLocation,
        });
    });

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

    it("returns an empty object for invalid format", () => {
        // Mock console.error
        const originalConsoleError = console.error;
        console.error = jest.fn();

        setWindowSearch("?invalid");
        expect(parseSearchParams()).toEqual({});

        // Restore console.error
        console.error = originalConsoleError;
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

describe("stringifySearchParams", () => {
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
    const originalLocation = window.location;

    const setWindowSearch = (search: string) => {
        Object.defineProperty(window, "location", {
            value: {
                ...originalLocation,
                search,
            },
            writable: true,
        });
    };

    afterEach(() => {
        Object.defineProperty(window, "location", {
            value: originalLocation,
        });
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
            description: "handles what looks like URL-encoded characters originally, by not changing them",
            input: { "key_one": "value%2Fone%22%3A%22nestedValue%22%7D" },
            expected: { "key_one": "value%2Fone%22%3A%22nestedValue%22%7D" },
        },
        {
            description: "handles nested objects",
            input: { key: { nestedKey: "nestedValue" } },
            expected: { key: { nestedKey: "nestedValue" } },
        },
        {
            description: "handles arrays",
            input: { key: ["value1", "value2", "value3"] },
            expected: { key: ["value1", "value2", "value3"] },
        },
        {
            description: "handles mixed types",
            input: { string: "value", number: 123, boolean: true, null: null },
            expected: { string: "value", number: 123, boolean: true },
        },
        {
            description: "handles complex nested structures",
            input: {
                key1: { nestedKey1: "nestedValue1" },
                key2: [{ id: 1, name: "Item1" }, { id: 2, name: "Item2" }],
            },
            expected: {
                key1: { nestedKey1: "nestedValue1" },
                key2: [{ id: 1, name: "Item1" }, { id: 2, name: "Item2" }],
            },
        },
        {
            description: "Handles negatives and decimals",
            input: {
                "positiveInteger": 123,
                "negativeInteger": -123,
                "positiveFloat": 123.456,
                "negativeFloat": -123.456,
                "zero": 0,
            },
            expected: {
                "positiveInteger": 123,
                "negativeInteger": -123,
                "positiveFloat": 123.456,
                "negativeFloat": -123.456,
                "zero": 0,
            },
        },
    ];

    testCases.forEach(({ description, input, expected }) => {
        it(description, () => {
            const searchParams = stringifySearchParams(input);
            setWindowSearch(searchParams);
            const parsedParams = parseSearchParams();
            expect(parsedParams).toEqual(expected);
        });
    });
});

describe("uuidToBase36 and base36ToUuid", () => {

    test("converts a standard UUID to base36 and back to UUID correctly", () => {
        const uuid = "123e4567-e89b-12d3-a456-426614174000";
        const base36 = uuidToBase36(uuid);
        const convertedUuid = base36ToUuid(base36);
        expect(convertedUuid).toBe(uuid);
    });

    test("handles a UUID with all zeros", () => {
        const uuid = "00000000-0000-0000-0000-000000000000";
        const base36 = uuidToBase36(uuid);
        const convertedUuid = base36ToUuid(base36);
        expect(convertedUuid).toBe(uuid);
    });

    test("handles a UUID with maximal values", () => {
        const uuid = "ffffffff-ffff-ffff-ffff-ffffffffffff";
        const base36 = uuidToBase36(uuid);
        const convertedUuid = base36ToUuid(base36);
        expect(convertedUuid).toBe(uuid);
    });

    test("returns empty string when invalid UUID is input to uuidToBase36", () => {
        const invalidUuid = "invalid-uuid-format";
        const result = uuidToBase36(invalidUuid);
        expect(result).toBe("");
    });

    test("returns empty string when invalid base36 is input to base36ToUuid", () => {
        const invalidBase36 = "invalid-base36-format";
        const result = base36ToUuid(invalidBase36);
        expect(result).toBe("");
    });
});
