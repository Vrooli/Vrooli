import { describe, it, expect } from "vitest";
import * as yup from "yup";
import { opt, req } from "./builders/optionality.js";
import { MAX_DOUBLE, MAX_INT, MIN_DOUBLE, MIN_INT, bigIntString, bool, config, double, doublePositiveOrZero, email, endDate, endTime, handle, hexColor, id, imageFile, index, int, intPositiveOrOne, intPositiveOrZero, minVersionTest, newEndTime, newStartTime, originalStartTime, publicId, pushNotificationKeys, startTime, timezone, url, versionLabel } from "./commonFields.js";

type Case = string | number | bigint | boolean | Date | null | undefined | { [x: string]: Case } | Case[];
type ValidatorSet = {
    schema: yup.AnySchema;
    valid?: Case[];
    invalid?: Case[];
    transforms?: [Case, Case][];
    context?: [string, yup.AnySchema, Case][];
};

const now = new Date();
const oneHourLater = new Date(now.getTime() + 3600000); // 1 hour later
const twoHoursLater = new Date(now.getTime() + 7200000); // 2 hours later

function safeStringify(value: unknown): string | undefined {
    try {
        return JSON.stringify(value, (key, value) => {
            if (typeof value === "bigint") {
                return value.toString();
            }
            return value;
        });
    } catch (e) {
        return String(value);
    }
}

describe("Yup validation tests", () => {
    const validators: Record<string, ValidatorSet> = {
        removeEmptyString: {
            schema: yup.string().removeEmptyString(),
            // Should leave strings the same, unless they are empty, whitespace, or invalid
            transforms: [
                ["", undefined],
                [" ", undefined],
                ["  ", undefined],
                [null, undefined],
                [undefined, undefined],
                [1, "1"], // yup automatically converts numbers to strings, so we have no choice but to accept them
                [true, "true"], // yup automatically converts booleans to strings, so we have no choice but to accept them
                [{}, "[object Object]"], // Objects without custom toString get their default representation
                ["a", "a"],
                [" a", " a"],
                ["a ", "a "],
                [" a ", " a "],
                [" a b ", " a b "],
            ],
        },
        toBool: {
            schema: yup.bool().toBool(),
            transforms: [
                ["true", true],
                [" true ", true],
                ["false", false],
                [" false ", false],
                [0, false],
                [1, true],
                ["0", false],
                ["1", true],
                ["", false],
                [" ", false],
                ["yes", true],
                [" no ", false],
                ["asdfsadf", false],
            ],
        },
        id: {
            schema: id,
            valid: [
                "1234567890123456789",
                BigInt("9876543210987654321").toString(),
            ],
            invalid: [
                "123e4567-e89b-12d3-a456-426614174000",
                "not-a-number",
                "123.45",
                "-100",
                "0",
                (2n ** 64n).toString(),
                "",
                " ",
            ],
            transforms: [
                [" 1234567890123456789  ", "1234567890123456789"],
                [9876543210987654321n, "9876543210987654321"],
                [123, "123"],
            ],
        },
        publicId: {
            schema: publicId,
            valid: [
                "abcdef0123",
                "abcdef0123xy",
            ],
            invalid: [
                "abc",
                "abcdef0123xyz",
                "ABCDEF0123",
                "abcdef-012",
                "abcdef!012",
                "",
                " ",
            ],
            transforms: [
                [" abcdef0123  ", "abcdef0123"],
                [1234567890, "1234567890"],
            ],
        },
        config: {
            schema: config,
            valid: [
                {},
                { key: "value" },
                { nested: { a: 1, b: "test" } },
                { arr: [1, "two", { c: true }] },
            ],
            invalid: [
                "not an object",
                123,
                null,
            ],
            transforms: [
            ],
        },
        email: {
            schema: email,
            valid: ["test@example.com", "valid_123@test.co", "another.valid@test.info"],
            invalid: [
                "test@",
                "invalidemail.com",
                "a".repeat(250) + "@example.com",
                "missingatsign.com",
                "missingdomain@.com",
            ],
            transforms: [[" boop@beep.com ", "boop@beep.com"]],
        },
        handle: {
            schema: handle,
            valid: ["user_123", "abc", "validHandle", "1234567890123456"],
            invalid: [
                "ab",
                "a".repeat(17),
                "user$123",
                "!@#$%^&*()",
                "user-name",
                "middle space",
            ],
            transforms: [["  user_123  ", "user_123"], ["  abc  ", "abc"]],
        },
        hexColor: {
            schema: hexColor,
            valid: ["#FFFFFF", "#000000", "#123ABC"],
            invalid: ["12345", "ZZZZZZ", "#1234567", ""],
            transforms: [["  #FFFFFF  ", "#FFFFFF"], ["  #000000  ", "#000000"]],
        },
        imageFile: {
            schema: imageFile,
            valid: ["image.jpg", "photo.png", "assets/image.gif", "a".repeat(250) + ".gif"],
            invalid: ["a".repeat(256) + ".jpg", "", "   "],
            transforms: [["  image.jpg  ", "image.jpg"], ["  photo.png  ", "photo.png"]],
        },
        pushNotificationKeys: {
            schema: pushNotificationKeys,
            valid: [{ p256dh: "validKey123", auth: "authKey123" }],
            invalid: [
                { p256dh: "a".repeat(257), auth: "valid" },
                { p256dh: "valid", auth: "a".repeat(257) },
                {},
            ],
        },
        urlProd: {
            schema: url({ env: "production" }),
            valid: ["https://www.example.com", "http://www.example.com", "https://example.com", "http://example.com", "ftp://example.com"],
            invalid: ["justastring", "http:/malformed.url", "https://192.168.1.1", "", "http://localhost"],
            transforms: [["  https://www.example.com  ", "https://www.example.com"]],
        },
        urlDev: {
            schema: url({ env: "development" }),
            valid: ["http://localhost:3000", "http://127.0.0.1:8080", "ftp://localhost:3000", "https://www.example.com", "http://www.example.com", "https://example.com", "http://example.com"],
            invalid: ["justastring", "http:/malformed.url", "", "https://192.168.1.1"],
            transforms: [["  http://localhost:3000  ", "http://localhost:3000"]],
        },
        bool: {
            schema: bool,
            valid: [true, false],
            transforms: [
                ["true", true],
                [" true ", true],
                ["false", false],
                [" false ", false],
                [0, false],
                [1, true],
                ["0", false],
                ["1", true],
                ["", false],
                [" ", false],
                ["yes", true],
                [" no ", false],
                ["asdfsadf", false],
            ],
        },
        bigIntString: {
            schema: bigIntString,
            valid: ["123", "0", "1000000000000000000", BigInt(1000000000000000000).toString()],
            invalid: [-2.1, "123.5", "notanumber", "123abc", NaN, null, undefined, {}, []],
            transforms: [[" 123 ", "123"], [123456789123, "123456789123"], [BigInt(-123456789123), "-123456789123"], [Number.MAX_SAFE_INTEGER, Number.MAX_SAFE_INTEGER.toString()], [0, "0"], [1, "1"]],
        },
        doublePositiveOrZero: {
            schema: doublePositiveOrZero,
            valid: [0, 10.5, MAX_DOUBLE],
            invalid: [-1, MAX_DOUBLE + 1, "string", NaN, Infinity],
            transforms: [[" 100 ", 100]],
        },
        intPositiveOrZero: {
            schema: intPositiveOrZero,
            valid: [0, 10, MAX_INT],
            invalid: [-1, MAX_INT + 1, 10.5, "string", NaN, Infinity],
            transforms: [[" 100 ", 100]],
        },
        intPositiveOrOne: {
            schema: intPositiveOrOne,
            valid: [1, 10, MAX_INT],
            invalid: [0, -1, MAX_INT + 1, 10.5, "string", NaN, Infinity],
            transforms: [[" 100 ", 100]],
        },
        double: {
            schema: double,
            valid: [MIN_DOUBLE, 0, MAX_DOUBLE],
            invalid: [MIN_DOUBLE - 1, MAX_DOUBLE + 1, "string", NaN, Infinity],
            transforms: [[" -100.5 ", -100.5]],
        },
        int: {
            schema: int,
            valid: [MIN_INT, 0, MAX_INT],
            invalid: [MIN_INT - 1, MAX_INT + 1, 10.5, "string", NaN, Infinity],
            transforms: [[" 100 ", 100]],
        },
        index: {
            schema: index,
            valid: [0, 10, MAX_INT],
            invalid: [-1, MAX_INT + 1, 10.5, "string", NaN, Infinity],
            transforms: [[" 100 ", 100]],
        },
        timezone: {
            schema: timezone,
            valid: ["America/New_York", "Europe/London", "Asia/Tokyo"],
            // invalid: ["Invalid/Timezone", "a".repeat(65), ""], // We currently don't validate more than just checking if it's a string
            transforms: [["  Europe/Berlin  ", "Europe/Berlin"]],
        },
        startTime: {
            schema: startTime,
            valid: [new Date(), new Date("2020-01-01")],
            invalid: ["invalid-date", ""],
        },
        endTime: {
            schema: endTime,
            context: [["startTime", startTime, now]],
            valid: [oneHourLater],
            invalid: [now, new Date(now.getTime() - 3600000)],
        },
        originalStartTime: {
            schema: originalStartTime,
            valid: [new Date(), new Date("2020-01-01")],
            invalid: ["invalid-date", ""],
        },
        newStartTime: {
            schema: newStartTime,
            valid: [new Date(), new Date("2020-01-01")],
            invalid: ["invalid-date", ""],
        },
        newEndTime: {
            schema: newEndTime,
            context: [["newStartTime", newStartTime, now]],
            valid: [twoHoursLater],
            invalid: [now, new Date(now.getTime() - 7200000)],
        },
        endDate: {
            schema: endDate,
            valid: [new Date()],
            invalid: ["invalid-date", ""],
        },
        versionLabel: {
            schema: versionLabel({ minVersion: "1.0.0" }),
            valid: [
                "1.0.0",
                "1.0.1",
                "1.1.0",
                "2.0.0",
            ],
            invalid: [
                "0.9.9",
                "0.0.9",
                "a.b.c",
                "1.0.0.0", // Assuming this format is invalid
                "".repeat(17), // Exceeds max length of 16
            ],
            transforms: [
                [" 1.0.0 ", "1.0.0"],
                ["1.0.0", "1.0.0"],
            ],
        },
    };

    Object.entries(validators).forEach(([key, { schema, valid, invalid, transforms, context }]) => {
        describe(`${key} validation`, () => {
            function generateTestSchema(isFieldRequired: boolean): {
                testSchema: yup.AnySchema,
                contextValues: Record<string, unknown>,
            } {
                let testSchema = isFieldRequired ? req(schema) : opt(schema);
                const contextValues: Record<string, unknown> = {};

                // When there is context, we have to test a yup object instead of a single field
                if (context) {
                    const contextSchema = {};
                    context.forEach(([fieldName, fieldSchema, fieldValue]) => {
                        contextSchema[fieldName] = fieldSchema;
                        contextValues[fieldName] = fieldValue;
                    });
                    testSchema = yup.object().shape({
                        ...contextSchema, // Spread the context schema
                        [key]: testSchema, // the target field
                    });
                }
                return { testSchema, contextValues };
            }

            // Testing valid cases
            valid?.forEach(v => {
                const keyString = key.length > 10 ? key.slice(0, 8) + "..." : key;
                let valueString = safeStringify(v);
                if (valueString && valueString.length > 100) {
                    valueString = valueString.slice(0, 100) + "...";
                }
                it(`should accept valid ${keyString}: ${valueString}`, async () => {
                    // Test the field as a required field
                    const { testSchema, contextValues } = generateTestSchema(true);
                    const testData = context ? { ...contextValues, [key]: v } : v;
                    const expected = context ? testData : v;
                    const result = await testSchema.validate(testData);
                    expect(result).to.deep.equal(expected);
                });
            });

            // Testing that null and undefined are accepted when the field is optional
            [null, undefined].forEach(v => {
                const keyString = key.length > 10 ? key.slice(0, 8) + "..." : key;
                it(`should accept null/undefined for optional ${keyString}`, async () => {
                    // Test the field as an optional field
                    const { testSchema, contextValues } = generateTestSchema(false);
                    const testData = context ? { ...contextValues, [key]: v } : v;

                    // Validate the testData and check the specific field in the result
                    const validationResult = await testSchema.validate(testData);
                    const fieldResult = context ? validationResult[key] : validationResult;
                    expect(fieldResult).to.not.be.ok; // Check if the specific field is falsy
                });
            });

            // Testing invalid cases
            invalid?.forEach(iv => {
                const keyString = key.length > 10 ? key.slice(0, 8) + "..." : key;
                let valueString = safeStringify(iv);
                if (valueString && valueString.length > 100) {
                    valueString = valueString.slice(0, 100) + "...";
                }
                it(`should reject invalid ${keyString}: ${valueString}`, async () => {
                    const { testSchema, contextValues } = generateTestSchema(true);
                    const testData = context ? { ...contextValues, [key]: iv } : iv;
                    try {
                        await testSchema.validate(testData);
                        expect.fail("Expected validation to reject");
                    } catch (e) {
                        expect(e).to.be.instanceOf(yup.ValidationError);
                    }
                });
            });

            // Testing transformation cases (if applicable)
            transforms?.forEach(([input, expectedOutput]) => {
                const keyString = key.length > 10 ? key.slice(0, 8) + "..." : key;
                let inputString = safeStringify(input);
                if (inputString && inputString.length > 100) {
                    inputString = inputString.slice(0, 100) + "...";
                }
                let expectedOutputString = safeStringify(expectedOutput);
                if (expectedOutputString && expectedOutputString.length > 100) {
                    expectedOutputString = expectedOutputString.slice(0, 100) + "...";
                }
                it(`should transform ${keyString}: ${inputString} to ${expectedOutputString}`, async () => {
                    const { testSchema, contextValues } = generateTestSchema(false); // We don't want the schema to be rejected, since we're only testing the transformation
                    const testData = context ? { ...contextValues, [key]: input } : input;
                    const expected = context ? { ...contextValues, [key]: expectedOutput } : expectedOutput;
                    const result = await testSchema.validate(testData);
                    expect(result).to.deep.equal(expected);
                });
            });
        });
    });
});

describe("minVersionTest function tests", () => {
    const minVersion = "1.0.0";

    it("version meets the minimum version requirement", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn("1.0.1")).to.equal(true);
    });

    it("version does not meet the minimum version requirement", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn("0.9.9")).to.equal(false);
    });

    it("undefined version", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(undefined)).to.equal(true);
    });

    it("minimum version as input version", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(minVersion)).to.equal(true);
    });
});
