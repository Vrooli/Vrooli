/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect, vi, beforeEach, afterEach, describe, it } from "vitest";
import { hashString, randomString, validateCode } from "./codes.js";

describe("randomString function", () => {
    it("should generate a string of default length (64) when no parameters provided", () => {
        const result = randomString();
        expect(result).toHaveLength(64);
        expect(typeof result).toBe("string");
    });

    it("should generate a string of the correct length", () => {
        const length = 10;
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
        const result = randomString(length, chars);
        expect(result).toHaveLength(length);
    });

    it("should generate a string using only the specified characters", () => {
        const length = 15;
        const chars = "ABC123";
        const result = randomString(length, chars);
        const isValid = [...result].every(char => chars.includes(char));
        expect(isValid).toBe(true);
    });

    it("should use default charset when none provided", () => {
        const result = randomString(100);
        const defaultChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
        // Every character in result should be from default charset
        for (const char of result) {
            expect(defaultChars).toContain(char);
        }
    });

    it("should generate different strings on subsequent calls", () => {
        const result1 = randomString(32);
        const result2 = randomString(32);
        expect(result1).not.toBe(result2);
    });

    it("should handle minimum and maximum valid parameters", () => {
        // Minimum length with minimum charset
        const minResult = randomString(1, "ab");
        expect(minResult).toHaveLength(1);
        expect(["a", "b"]).toContain(minResult);
        
        // Maximum length (test with smaller size for performance)
        const maxCharset = "a".repeat(256);
        const maxResult = randomString(100, maxCharset);
        expect(maxResult).toHaveLength(100);
        expect(maxResult).toBe("a".repeat(100));
    });

    // Test for error thrown on invalid length parameters
    const invalidLengths = [-1, 0, 2049, 2.3, null, "hello", 2.3, {}, [], NaN, Infinity];
    invalidLengths.forEach(length => {
        it(`should throw an error for invalid length: ${length}`, () => {
            // @ts-ignore: Testing runtime scenario
            expect(() => randomString(length)).toThrow("Length must be bewteen 1 and 2048.");
        });
    });

    // Test for error thrown on invalid chars parameters
    const invalidChars = ["", "A", "A".repeat(257)]; // Empty, too short, and too long
    invalidChars.forEach(chars => {
        it(`should throw an error for invalid chars set: ${chars.length} characters long`, () => {
            expect(() => randomString(10, chars)).toThrow("Chars must be bewteen 2 and 256 characters long.");
        });
    });

    it("should throw error for non-string charset", () => {
        // @ts-ignore: Testing runtime scenario
        expect(() => randomString(10, 123)).toThrow("Chars must be bewteen 2 and 256 characters long.");
    });
});

describe("validateCode function", () => {
    const validTimeout = 60000; // 1 minute
    
    it("should return true for valid matching codes within timeout", () => {
        const code = "TEST123";
        const dateRequested = new Date(Date.now() - 30000); // 30 seconds ago
        
        const result = validateCode(code, code, dateRequested, validTimeout);
        expect(result).toBe(true);
    });

    it("should return false for mismatched codes", () => {
        const providedCode = "TEST123";
        const storedCode = "DIFFERENT456";
        const dateRequested = new Date(Date.now() - 30000);
        
        const result = validateCode(providedCode, storedCode, dateRequested, validTimeout);
        expect(result).toBe(false);
    });

    it("should return false for expired codes", () => {
        const code = "TEST123";
        const dateRequested = new Date(Date.now() - 120000); // 2 minutes ago (beyond 1 minute timeout)
        
        const result = validateCode(code, code, dateRequested, validTimeout);
        expect(result).toBe(false);
    });

    it("should return false for null/undefined provided code", () => {
        const storedCode = "TEST123";
        const dateRequested = new Date();
        
        expect(validateCode(null, storedCode, dateRequested, validTimeout)).toBe(false);
        expect(validateCode(undefined as any, storedCode, dateRequested, validTimeout)).toBe(false);
        expect(validateCode("", storedCode, dateRequested, validTimeout)).toBe(false);
    });

    it("should return false for null/undefined stored code", () => {
        const providedCode = "TEST123";
        const dateRequested = new Date();
        
        expect(validateCode(providedCode, null, dateRequested, validTimeout)).toBe(false);
        expect(validateCode(providedCode, undefined as any, dateRequested, validTimeout)).toBe(false);
        expect(validateCode(providedCode, "", dateRequested, validTimeout)).toBe(false);
    });

    it("should return false for null/undefined date requested", () => {
        const code = "TEST123";
        
        expect(validateCode(code, code, null, validTimeout)).toBe(false);
        expect(validateCode(code, code, undefined as any, validTimeout)).toBe(false);
    });

    it("should return false for invalid timeout values", () => {
        const code = "TEST123";
        const dateRequested = new Date();
        
        // Zero timeout
        expect(validateCode(code, code, dateRequested, 0)).toBe(false);
        
        // Negative timeout
        expect(validateCode(code, code, dateRequested, -1000)).toBe(false);
        
        // Non-integer timeout
        expect(validateCode(code, code, dateRequested, 5.5)).toBe(false);
    });

    it("should handle edge case where code expires exactly at timeout boundary", () => {
        const code = "TEST123";
        const timeout = 60000; // 1 minute
        const dateRequested = new Date(Date.now() - timeout - 1); // 1ms beyond timeout
        
        const result = validateCode(code, code, dateRequested, timeout);
        expect(result).toBe(false);
    });

    it("should be case sensitive for code comparison", () => {
        const providedCode = "test123";
        const storedCode = "TEST123";
        const dateRequested = new Date();
        
        const result = validateCode(providedCode, storedCode, dateRequested, validTimeout);
        expect(result).toBe(false);
    });

    it("should handle exactly at timeout boundary (valid)", () => {
        const code = "TEST123";
        const timeout = 60000;
        const dateRequested = new Date(Date.now() - timeout + 1); // 1ms before timeout
        
        const result = validateCode(code, code, dateRequested, timeout);
        expect(result).toBe(true);
    });
});

describe("hashString", () => {
    it("should return a consistent hash for the same input", () => {
        const input = "test string";
        const hash1 = hashString(input);
        const hash2 = hashString(input);
        
        expect(hash1).toBe(hash2);
        expect(typeof hash1).toBe("string");
        expect(hash1.length).toBeGreaterThan(0);
    });

    it("should return different hashes for different inputs", () => {
        const input1 = "test string 1";
        const input2 = "test string 2";
        
        const hash1 = hashString(input1);
        const hash2 = hashString(input2);
        
        expect(hash1).not.toBe(hash2);
    });

    it("should return MD5 hex format (32 characters)", () => {
        const input = "test";
        const hash = hashString(input);
        
        // MD5 hash should be 32 hex characters
        expect(hash).toHaveLength(32);
        expect(hash).toMatch(/^[a-f0-9]{32}$/);
    });

    it("should handle empty string", () => {
        const hash = hashString("");
        expect(typeof hash).toBe("string");
        expect(hash).toHaveLength(32);
        // MD5 of empty string should be consistent
        expect(hash).toBe("d41d8cd98f00b204e9800998ecf8427e");
    });

    it("should handle special characters and unicode", () => {
        const inputs = [
            "Hello 世界!",
            "🎉✨🚀",
            "Special chars: !@#$%^&*()",
            "Line\\nBreak\\tTab",
        ];
        
        inputs.forEach(input => {
            const hash = hashString(input);
            expect(hash).toHaveLength(32);
            expect(hash).toMatch(/^[a-f0-9]{32}$/);
        });
    });

    it("should be deterministic across multiple calls", () => {
        const input = "deterministic test";
        const hashes = Array.from({ length: 10 }, () => hashString(input));
        
        // All hashes should be identical
        hashes.forEach(hash => {
            expect(hash).toBe(hashes[0]);
        });
    });

    it("should handle very long strings", () => {
        const longString = "a".repeat(10000);
        const hash = hashString(longString);
        
        expect(hash).toHaveLength(32);
        expect(hash).toMatch(/^[a-f0-9]{32}$/);
    });

    it("should produce known MD5 hashes for test vectors", () => {
        // Known MD5 test vectors
        const testVectors = [
            { input: "abc", expected: "900150983cd24fb0d6963f7d28e17f72" },
            { input: "hello", expected: "5d41402abc4b2a76b9719d911017c592" },
            { input: "The quick brown fox jumps over the lazy dog", expected: "9e107d9d372bb6826bd81d3542a419d6" },
        ];
        
        testVectors.forEach(({ input, expected }) => {
            expect(hashString(input)).toBe(expected);
        });
    });
});
