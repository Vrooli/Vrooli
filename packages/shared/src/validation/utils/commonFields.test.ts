import * as yup from "yup";
import { uuid } from "../../id";
import { opt, req } from "./builders";
import { MAX_DOUBLE, MAX_INT, MIN_DOUBLE, MIN_INT, bool, configCallData, double, doublePositiveOrZero, email, endDate, endTime, handle, hexColor, id, imageFile, index, int, intPositiveOrOne, intPositiveOrZero, minVersionTest, newEndTime, newStartTime, originalStartTime, pushNotificationKeys, startTime, timezone, url, versionLabel } from "./commonFields";

type Case = string | number | boolean | Date | null | undefined | { [x: string]: Case } | Case[];
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
                [{}, undefined],
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
                uuid(),
                uuid().toUpperCase(),
            ],
            invalid: [
                uuid().slice(0, -1), // One less character
                uuid() + "0", // One extra character
                uuid().replace("-", ""), // No hyphens
                uuid().replace("-", "_"), // Underscore instead of hyphen
                uuid().slice(0, 14) + "g" + uuid().slice(15), // Random 'g' inserted in the middle
                "123",
                "not-an-id",
                "a".repeat(257),
                "",
                " ",
            ],
            transforms: [[" 123e4567-e89b-12d3-a456-426614174000  ", "123e4567-e89b-12d3-a456-426614174000"]],
        },
        configCallData: {
            schema: configCallData,
            valid: ["{\"key\": \"value\"}", "{\"another\": \"item\"}", "{\"array\": [1, 2, 3]}"],
            invalid: ["a".repeat(8193), "", " ".repeat(100)],
            transforms: [["  {\"key\": \"value\"}  ", "{\"key\": \"value\"}"]],
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
            const generateTestSchema = (isFieldRequired: boolean): {
                testSchema: yup.AnySchema,
                contextValues: Record<string, unknown>,
            } => {
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
            };

            // Testing valid cases
            valid?.forEach(v => {
                it(`should accept valid ${key}: ${JSON.stringify(v)}`, async () => {
                    // Test the field as a required field
                    const { testSchema, contextValues } = generateTestSchema(true);
                    const testData = context ? { ...contextValues, [key]: v } : v;
                    const expected = context ? testData : v;
                    await expect(testSchema.validate(testData)).resolves.toEqual(expected);
                });
            });

            // Testing that null and undefined are accepted when the field is optional
            [null, undefined].forEach(v => {
                it(`should accept null/undefined for optional ${key}`, async () => {
                    // Test the field as an optional field
                    const { testSchema, contextValues } = generateTestSchema(false);
                    const testData = context ? { ...contextValues, [key]: v } : v;

                    // Validate the testData and check the specific field in the result
                    const validationResult = await testSchema.validate(testData);
                    const fieldResult = context ? validationResult[key] : validationResult;
                    expect(fieldResult).toBeFalsy(); // Check if the specific field is falsy
                });
            });

            // Testing invalid cases
            invalid?.forEach(iv => {
                it(`should reject invalid ${key}: ${JSON.stringify(iv)}`, async () => {
                    const { testSchema, contextValues } = generateTestSchema(true);
                    const testData = context ? { ...contextValues, [key]: iv } : iv;
                    await expect(testSchema.validate(testData)).rejects.toThrow();
                });
            });

            // Testing transformation cases (if applicable)
            transforms?.forEach(([input, expectedOutput]) => {
                it(`should transform ${key}: ${JSON.stringify(input)} to ${JSON.stringify(expectedOutput)}`, async () => {
                    const { testSchema, contextValues } = generateTestSchema(false); // We don't want the schema to be rejected, since we're only testing the transformation
                    const testData = context ? { ...contextValues, [key]: input } : input;
                    const expected = context ? { ...contextValues, [key]: expectedOutput } : expectedOutput;
                    await expect(testSchema.validate(testData)).resolves.toEqual(expected);
                });
            });
        });
    });
});

describe("minVersionTest function tests", () => {
    const minVersion = "1.0.0";

    test("version meets the minimum version requirement", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn("1.0.1")).toBe(true);
    });

    test("version does not meet the minimum version requirement", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn("0.9.9")).toBe(false);
    });

    test("undefined version", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(undefined)).toBe(true);
    });

    test("minimum version as input version", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(minVersion)).toBe(true);
    });
});