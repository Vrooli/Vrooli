// AI_CHECK: TEST_COVERAGE=1 | TEST_QUALITY=1 | LAST: 2025-06-19
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { chatMatchHash, weakHash, randomString } from "./codes.js";

describe("weakHash", () => {
    it("generates a consistent hash for the given string", () => {
        const inputString = "test-string";
        const hash = weakHash(inputString);
        expect(typeof hash).toBe("string");
        expect(hash).toBe(weakHash(inputString)); // Should be consistent
    });

    it("returns '0' for empty string", () => {
        const hash = weakHash("");
        expect(hash).toBe("0");
    });

    it("generates different hashes for different strings", () => {
        const hash1 = weakHash("string1");
        const hash2 = weakHash("string2");
        expect(hash1).not.toBe(hash2);
    });

    it("handles unicode characters", () => {
        const hash = weakHash("🚀✨");
        expect(typeof hash).toBe("string");
        expect(hash).toBe(weakHash("🚀✨"));
    });
});

describe("chatMatchHash", () => {
    it("generates the same hash regardless of the order of participant user IDs", () => {
        // These two arrays contain the same IDs in different orders
        const participantIds1 = ["alice", "bob", "charlie"];
        const participantIds2 = ["charlie", "alice", "bob"];

        const hash1 = chatMatchHash(participantIds1);
        const hash2 = chatMatchHash(participantIds2);

        expect(hash1).toBe(hash2);
    });

    it("generates different hashes for different sets of participant user IDs", () => {
        const participantIds1 = ["alice", "bob", "charlie"];
        const participantIds2 = ["alice", "bob", "david"];

        const hash1 = chatMatchHash(participantIds1);
        const hash2 = chatMatchHash(participantIds2);

        expect(hash1).not.toBe(hash2);
    });

    it("generates a consistent hash for a given set of participant user IDs", () => {
        const participantIds = ["alice", "bob", "charlie"];

        const hash1 = chatMatchHash(participantIds);
        const hash2 = chatMatchHash(participantIds);

        expect(hash1).toBe(hash2);
    });

    it("handles empty array", () => {
        const hash = chatMatchHash([]);
        expect(typeof hash).toBe("string");
        expect(hash).toBe(chatMatchHash([]));
    });

    it("handles single participant", () => {
        const hash = chatMatchHash(["alice"]);
        expect(typeof hash).toBe("string");
        expect(hash).toBe(chatMatchHash(["alice"]));
    });
});

describe("randomString", () => {
    it("generates a string of default length by default", () => {
        const result = randomString();
        expect(typeof result).toBe("string");
        expect(result.length).toBe(7); // DEFAULT_RANDOM_STRING_LENGTH
    });

    it("generates a string of specified length", () => {
        const result = randomString(5);
        expect(result.length).toBe(5);
    });

    it("handles invalid length parameters", () => {
        const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        
        // Negative length
        const result1 = randomString(-1);
        expect(result1.length).toBe(7); // Falls back to default
        
        // Zero length
        const result2 = randomString(0);
        expect(result2.length).toBe(7); // Falls back to default
        
        // Too large length
        const result3 = randomString(15);
        expect(result3.length).toBe(7); // Falls back to default

        // Non-integer length
        const result4 = randomString(3.5);
        expect(result4.length).toBe(7); // Falls back to default

        expect(consoleErrorSpy).toHaveBeenCalledTimes(4);
        consoleErrorSpy.mockRestore();
    });

    it("handles invalid radix parameters", () => {
        const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        
        // Too small radix
        const result1 = randomString(5, 1);
        expect(result1.length).toBe(5);
        
        // Too large radix
        const result2 = randomString(5, 300);
        expect(result2.length).toBe(5);

        // Non-integer radix
        const result3 = randomString(5, 10.5);
        expect(result3.length).toBe(5);

        expect(consoleErrorSpy).toHaveBeenCalledTimes(3);
        consoleErrorSpy.mockRestore();
    });

    it("generates different strings on subsequent calls", () => {
        const result1 = randomString();
        const result2 = randomString();
        // Very low probability of collision with 7-character random strings
        expect(result1).not.toBe(result2);
    });

    it("respects custom radix", () => {
        const result = randomString(5, 16); // Hexadecimal
        expect(result.length).toBe(5);
        expect(/^[0-9a-f]+$/.test(result)).toBe(true);
    });

    it("works with binary radix", () => {
        const result = randomString(5, 2); // Binary
        expect(result.length).toBe(5);
        expect(/^[01]+$/.test(result)).toBe(true);
    });
});
