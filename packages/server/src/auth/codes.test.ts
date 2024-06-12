/* eslint-disable @typescript-eslint/ban-ts-comment */
import { hashString, randomString, validateCode } from "./codes";

describe("randomString function", () => {
    // Test for correct string length
    it("should generate a string of the correct length", () => {
        const length = 10;
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
        const result = randomString(length, chars);
        expect(result).toHaveLength(length);
    });

    // Test to ensure the string uses only specified characters
    it("should generate a string using only the specified characters", () => {
        const length = 15;
        const chars = "ABC123";
        const result = randomString(length, chars);
        const isValid = [...result].every(char => chars.includes(char));
        expect(isValid).toBeTruthy();
    });

    // Test for error thrown on invalid length parameters
    const invalidLengths = [-1, 0, 2049, 2.3, null, "hello", 2.3, {}, [], NaN, Infinity];
    invalidLengths.forEach(length => {
        it(`should throw an error for invalid length: ${length}`, () => {
            // @ts-ignore: Testing runtime scenario
            expect(() => randomString(length)).toThrow();
        });
    });

    // Test for error thrown on invalid chars parameters
    const invalidChars = ["", "A".repeat(257)]; // Empty string and string longer than 256 characters
    invalidChars.forEach(chars => {
        it(`should throw an error for invalid chars set: ${chars.length} characters long`, () => {
            expect(() => randomString(10, chars)).toThrow();
        });
    });

    // Optional: Test for randomness or distribution
    // This would involve generating a large number of strings, analyzing character frequencies, and possibly employing statistical tests to assess randomness.
    // Such tests can be complex and are often beyond the scope of unit testing.
});

describe("validateCode function", () => {
    const storedCode = "123456";
    const dateNowSpy = jest.spyOn(Date, "now");
    const timeoutInMs = 100000;

    // Cleanup after all tests are done
    afterAll(() => {
        dateNowSpy.mockRestore();
    });

    it("should return true if providedCode matches storedCode and is within timeout", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        dateNowSpy.mockReturnValue(new Date(dateRequested.getTime() + timeoutInMs - 1000).getTime());

        expect(validateCode(providedCode, storedCode, dateRequested, timeoutInMs)).toBe(true);
    });

    it("should return false if providedCode matches storedCode but is outside timeout", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        dateNowSpy.mockReturnValue(new Date(dateRequested.getTime() + timeoutInMs + 1000).getTime());

        expect(validateCode(providedCode, storedCode, dateRequested, timeoutInMs)).toBe(false);
    });

    it("should return false if providedCode does not match storedCode", () => {
        const providedCode = "654321";
        const dateRequested = new Date();

        expect(validateCode(providedCode, storedCode, dateRequested, timeoutInMs)).toBe(false);
    });

    it("should return false if either providedCode or storedCode is null", () => {
        const dateRequested = new Date();
        expect(validateCode(null, storedCode, dateRequested, timeoutInMs)).toBe(false);
        expect(validateCode(storedCode, null, dateRequested, timeoutInMs)).toBe(false);
    });

    it("should return false if dateRequested is null", () => {
        const providedCode = "123456";
        expect(validateCode(providedCode, storedCode, null, timeoutInMs)).toBe(false);
    });

    it("should return false is timeout is null", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        // @ts-ignore: Testing runtime scenario
        expect(validateCode(providedCode, storedCode, dateRequested, null)).toBe(false);
    });

    it("should return false if timeout is less than 0", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        expect(validateCode(providedCode, storedCode, dateRequested, -1)).toBe(false);
    });

    // Additional test for robustness: check behavior when dateRequested is significantly in the past
    it("should return false if dateRequested is far in the past (well outside timeoutInMs)", () => {
        const providedCode = "123456";
        const dateRequested = new Date(Date.now() - timeoutInMs * 2); // Double the timeout
        expect(validateCode(providedCode, storedCode, dateRequested, timeoutInMs)).toBe(false);
    });
});

describe("hashString", () => {
    // Test basic functionality
    test("returns a consistent hash for a specific string", () => {
        const testString = "Hello, world!";
        expect(hashString(testString)).toBe(hashString(testString));
    });

    // Test edge cases
    describe("handles edge cases", () => {
        test("correctly hashes an empty string", () => {
            const result = hashString("");
            expect(typeof result).toBe("string");
            expect(result.length).toBeGreaterThan(0);
            expect(result.length).toBeLessThanOrEqual(128);
        });

        test("correctly hashes a very long string", () => {
            const result = hashString("a".repeat(1000));
            expect(typeof result).toBe("string");
            expect(result.length).toBeGreaterThan(0);
            expect(result.length).toBeLessThanOrEqual(128);
        });

        test("correctly hashes a string with all sorts of characters", () => {
            const result = hashString("你好，世界！Привет, мир! こんにちは、世界！12345 😊🚀🌟@#&*");
            expect(typeof result).toBe("string");
            expect(result.length).toBeGreaterThan(0);
            expect(result.length).toBeLessThanOrEqual(128);
        });
    });

    // Test uniqueness
    test("generates different hashes for different strings", () => {
        const string1 = "hello";
        const string2 = "hello!";
        expect(hashString(string1)).not.toBe(hashString(string2));
    });

    // Optional: Test error handling (if applicable)
    test("throws an error when input is not a string", () => {
        const invalidInput = 123; // This would be a TypeScript error, but just in case of runtime JS usage
        expect(() => hashString(invalidInput as unknown as string)).toThrow();
    });
});
