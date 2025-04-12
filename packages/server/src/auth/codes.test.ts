/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import sinon from "sinon";
import { hashString, randomString, validateCode } from "./codes.js";

describe("randomString function", () => {
    // Test for correct string length
    it("should generate a string of the correct length", () => {
        const length = 10;
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
        const result = randomString(length, chars);
        expect(result).to.have.lengthOf(length);
    });

    // Test to ensure the string uses only specified characters
    it("should generate a string using only the specified characters", () => {
        const length = 15;
        const chars = "ABC123";
        const result = randomString(length, chars);
        const isValid = [...result].every(char => chars.includes(char));
        expect(isValid).to.be.true;
    });

    // Test for error thrown on invalid length parameters
    const invalidLengths = [-1, 0, 2049, 2.3, null, "hello", 2.3, {}, [], NaN, Infinity];
    invalidLengths.forEach(length => {
        it(`should throw an error for invalid length: ${length}`, () => {
            // @ts-ignore: Testing runtime scenario
            expect(() => randomString(length)).to.throw();
        });
    });

    // Test for error thrown on invalid chars parameters
    const invalidChars = ["", "A".repeat(257)]; // Empty string and string longer than 256 characters
    invalidChars.forEach(chars => {
        it(`should throw an error for invalid chars set: ${chars.length} characters long`, () => {
            expect(() => randomString(10, chars)).to.throw();
        });
    });

    // Optional: Test for randomness or distribution
    // This would involve generating a large number of strings, analyzing character frequencies, and possibly employing statistical tests to assess randomness.
    // Such tests can be complex and are often beyond the scope of unit testing.
});

describe("validateCode function", () => {
    const storedCode = "123456";
    let clock: sinon.SinonFakeTimers;
    const timeoutInMs = 100000;

    beforeEach(() => {
        clock = sinon.useFakeTimers(new Date());
    });

    afterEach(() => {
        clock.restore();
    });

    it("should return true if providedCode matches storedCode and is within timeout", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        clock.tick(timeoutInMs - 1000);

        expect(validateCode(providedCode, storedCode, dateRequested, timeoutInMs)).to.be.true;
    });

    it("should return false if providedCode matches storedCode but is outside timeout", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        clock.tick(timeoutInMs + 1000);

        expect(validateCode(providedCode, storedCode, dateRequested, timeoutInMs)).to.be.false;
    });

    it("should return false if providedCode does not match storedCode", () => {
        const providedCode = "654321";
        const dateRequested = new Date();

        expect(validateCode(providedCode, storedCode, dateRequested, timeoutInMs)).to.be.false;
    });

    it("should return false if either providedCode or storedCode is null", () => {
        const dateRequested = new Date();
        expect(validateCode(null, storedCode, dateRequested, timeoutInMs)).to.be.false;
        expect(validateCode(storedCode, null, dateRequested, timeoutInMs)).to.be.false;
    });

    it("should return false if dateRequested is null", () => {
        const providedCode = "123456";
        expect(validateCode(providedCode, storedCode, null, timeoutInMs)).to.be.false;
    });

    it("should return false is timeout is null", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        // @ts-ignore: Testing runtime scenario
        expect(validateCode(providedCode, storedCode, dateRequested, null)).to.be.false;
    });

    it("should return false if timeout is less than 0", () => {
        const providedCode = "123456";
        const dateRequested = new Date();
        expect(validateCode(providedCode, storedCode, dateRequested, -1)).to.be.false;
    });

    // Additional test for robustness: check behavior when dateRequested is significantly in the past
    it("should return false if dateRequested is far in the past (well outside timeoutInMs)", () => {
        const providedCode = "123456";
        const pastDate = new Date();
        clock.tick(timeoutInMs * 2); // Double the timeout

        expect(validateCode(providedCode, storedCode, pastDate, timeoutInMs)).to.be.false;
    });
});

describe("hashString", () => {
    // Test basic functionality
    it("returns a consistent hash for a specific string", () => {
        const testString = "Hello, world!";
        expect(hashString(testString)).to.equal(hashString(testString));
    });

    // Test edge cases
    describe("handles edge cases", () => {
        it("correctly hashes an empty string", () => {
            const result = hashString("");
            expect(typeof result).to.equal("string");
            expect(result.length).to.be.greaterThan(0);
            expect(result.length).to.be.at.most(128);
        });

        it("correctly hashes a very long string", () => {
            const result = hashString("a".repeat(1000));
            expect(typeof result).to.equal("string");
            expect(result.length).to.be.greaterThan(0);
            expect(result.length).to.be.at.most(128);
        });

        it("correctly hashes a string with all sorts of characters", () => {
            const result = hashString("你好，世界！Привет, мир! こんにちは、世界！12345 😊🚀🌟@#&*");
            expect(typeof result).to.equal("string");
            expect(result.length).to.be.greaterThan(0);
            expect(result.length).to.be.at.most(128);
        });
    });

    // Test uniqueness
    it("generates different hashes for different strings", () => {
        const string1 = "hello";
        const string2 = "hello!";
        expect(hashString(string1)).not.to.equal(hashString(string2));
    });

    // Optional: Test error handling (if applicable)
    it("throws an error when input is not a string", () => {
        const invalidInput = 123; // This would be a TypeScript error, but just in case of runtime JS usage
        // @ts-ignore: Testing runtime scenario
        expect(() => hashString(invalidInput)).to.throw();
    });
});
