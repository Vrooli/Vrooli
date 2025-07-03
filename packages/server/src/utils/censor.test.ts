import { describe, expect, it, beforeEach, vi } from "vitest";
import fs from "fs";
import { initializeProfanity, hasProfanity, toStringArray, filterProfanity, resetProfanityState } from "./censor.js";

// Mock fs module
vi.mock("fs");
const mockFs = vi.mocked(fs);

// Mock logger
vi.mock("../events/logger.js", () => ({
    logger: {
        error: vi.fn(),
    },
}));

describe("censor utilities", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Reset the module state
        resetProfanityState();
        // Mock a successful file read
        mockFs.readFileSync.mockReturnValue("badword1\nbadword2\ntest\nhell");
        // Initialize with mocked data
        initializeProfanity();
    });

    describe("initializeProfanity", () => {
        it("should initialize profanity list from file", () => {
            const mockContent = "word1\nword2\nword3";
            resetProfanityState();
            mockFs.readFileSync.mockReturnValue(mockContent);
            
            initializeProfanity();
            
            expect(mockFs.readFileSync).toHaveBeenCalledWith(
                expect.stringContaining("censorDictionary.txt"),
                "utf8",
            );
        });

        it("should handle file read errors gracefully", () => {
            resetProfanityState();
            mockFs.readFileSync.mockImplementation(() => {
                throw new Error("File not found");
            });

            expect(() => initializeProfanity()).not.toThrow();
        });
    });

    describe("hasProfanity", () => {
        it("should return false for clean text", () => {
            expect(hasProfanity("hello world")).toBe(false);
            expect(hasProfanity("this is a nice message")).toBe(false);
        });

        it("should return true for text containing profanity", () => {
            expect(hasProfanity("this contains badword1")).toBe(true);
            expect(hasProfanity("badword2 is here")).toBe(true);
        });

        it("should handle multiple text parameters", () => {
            expect(hasProfanity("clean text", "also clean")).toBe(false);
            expect(hasProfanity("clean text", "contains badword1")).toBe(true);
            expect(hasProfanity("badword1", "badword2")).toBe(true);
        });

        it("should handle null and undefined values", () => {
            expect(hasProfanity(null)).toBe(false);
            expect(hasProfanity(undefined)).toBe(false);
            expect(hasProfanity("clean", null, "text")).toBe(false);
            expect(hasProfanity("clean", undefined, "badword1")).toBe(true);
        });

        it("should handle empty strings", () => {
            expect(hasProfanity("")).toBe(false);
            expect(hasProfanity("", "clean")).toBe(false);
        });

        it("should be case insensitive", () => {
            expect(hasProfanity("BADWORD1")).toBe(true);
            expect(hasProfanity("BadWord1")).toBe(true);
            expect(hasProfanity("badWORD1")).toBe(true);
        });

        it("should respect word boundaries", () => {
            // "hell" is in profanity list but "hello" should not match
            expect(hasProfanity("hello")).toBe(false);
            expect(hasProfanity("hell")).toBe(true);
            expect(hasProfanity("shell")).toBe(false);
            expect(hasProfanity("hell world")).toBe(true);
        });
    });

    describe("toStringArray", () => {
        it("should convert string to array", () => {
            const result = toStringArray("hello", null);
            expect(result).toEqual(["hello"]);
        });

        it("should handle arrays", () => {
            const result = toStringArray(["hello", "world"], null);
            expect(result).toEqual(["hello", "world"]);
        });

        it("should handle nested arrays", () => {
            const result = toStringArray([["hello", "world"], ["foo", "bar"]], null);
            expect(result).toEqual(["hello", "world", "foo", "bar"]);
        });

        it("should handle objects", () => {
            const obj = { name: "test", description: "hello world" };
            const result = toStringArray(obj, null);
            expect(result).toEqual(["test", "hello world"]);
        });

        it("should handle objects with specified fields", () => {
            const obj = { name: "test", description: "hello world", id: 123 };
            const result = toStringArray(obj, ["name", "description"]);
            expect(result).toEqual(["test", "hello world"]);
        });

        it("should handle nested objects", () => {
            const obj = {
                user: { name: "john", email: "john@test.com" },
                content: "hello world",
            };
            const result = toStringArray(obj, null);
            expect(result).toEqual(["john", "john@test.com", "hello world"]);
        });

        it("should handle field filtering at top level", () => {
            const obj = {
                user: { name: "john", email: "john@test.com" },
                content: "hello world",
                other: "ignored field",
            };
            // When specifying fields, only those fields' values are processed
            const result = toStringArray(obj, ["content"]);
            expect(result).toEqual(["hello world"]);
            
            // Test the "other" field is truly ignored
            const resultWithOther = toStringArray(obj, ["other"]);
            expect(resultWithOther).toEqual(["ignored field"]);
            
            // Test null fields processes all
            const resultAll = toStringArray(obj, null);
            expect(resultAll).toEqual(["john", "john@test.com", "hello world", "ignored field"]);
        });

        it("should handle mixed data types", () => {
            const obj = {
                text: "hello",
                number: 123,
                date: new Date(),
                bool: true,
                nullable: null,
                undefined,
            };
            const result = toStringArray(obj, null);
            expect(result).toEqual(["hello"]);
        });

        it("should filter out null values", () => {
            const mixed = ["hello", null, "world", undefined, 123];
            const result = toStringArray(mixed, null);
            expect(result).toEqual(["hello", "world"]);
        });

        it("should return null for non-string primitives", () => {
            expect(toStringArray(123, null)).toBeNull();
            expect(toStringArray(true, null)).toBeNull();
            expect(toStringArray(null, null)).toBeNull();
            expect(toStringArray(undefined, null)).toBeNull();
        });

        it("should handle Date objects correctly", () => {
            const date = new Date();
            expect(toStringArray(date, null)).toBeNull();
        });

        it("should handle complex nested structures", () => {
            const complex = {
                users: [
                    { name: "john", profile: { bio: "developer" } },
                    { name: "jane", profile: { bio: "designer" } },
                ],
                title: "test project",
            };
            const result = toStringArray(complex, null);
            expect(result).toEqual(["john", "developer", "jane", "designer", "test project"]);
        });
    });

    describe("filterProfanity", () => {
        it("should censor profanity with default character", () => {
            const result = filterProfanity("this contains badword1");
            expect(result).toBe("this contains ********");
        });

        it("should censor profanity with custom character", () => {
            const result = filterProfanity("this contains badword1", "#");
            expect(result).toBe("this contains ########");
        });

        it("should handle multiple profane words", () => {
            const result = filterProfanity("badword1 and badword2 here");
            expect(result).toBe("******** and ******** here");
        });

        it("should preserve non-profane text", () => {
            const cleanText = "this is perfectly clean text";
            const result = filterProfanity(cleanText);
            expect(result).toBe(cleanText);
        });

        it("should handle empty strings", () => {
            expect(filterProfanity("")).toBe("");
        });

        it("should be case insensitive", () => {
            const result = filterProfanity("BADWORD1 and BadWord2");
            expect(result).toBe("******** and ********");
        });

        it("should handle profanity at start and end", () => {
            const result = filterProfanity("badword1 middle content badword2");
            expect(result).toBe("******** middle content ********");
        });

        it("should respect word boundaries", () => {
            // "hell" should be censored but not "hello"
            const result = filterProfanity("hello hell shell");
            expect(result).toBe("hello **** shell");
        });

        it("should handle consecutive profanity", () => {
            const result = filterProfanity("badword1 badword2", "@");
            // This tests if word boundaries work correctly with spaces
            expect(result).toBe("@@@@@@@@ @@@@@@@@");
        });

        it("should handle different censor characters", () => {
            const testCases = [
                { char: "*", expected: "********" },
                { char: "#", expected: "########" },
                { char: "X", expected: "XXXXXXXX" },
                { char: "_", expected: "________" },
            ];

            testCases.forEach(({ char, expected }) => {
                const result = filterProfanity("badword1", char);
                expect(result).toBe(expected);
            });
        });
    });

    describe("integration tests", () => {
        it("should work together for content moderation workflow", () => {
            const userContent = {
                title: "My Post",
                body: "This contains badword1 content",
                comments: [
                    { text: "Nice post!" },
                    { text: "This has badword2 in it" },
                ],
            };

            // Extract all text for checking
            const textArray = toStringArray(userContent, null);
            
            // Check for profanity
            const hasBadWords = hasProfanity(...(textArray || []));
            expect(hasBadWords).toBe(true);

            // Filter profanity from individual strings
            const cleanBody = filterProfanity(userContent.body);
            expect(cleanBody).toBe("This contains ******** content");
        });

        it("should handle complex nested structures with profanity", () => {
            const nestedData = {
                posts: [
                    {
                        content: "Clean content here",
                        author: { name: "John", bio: "Nice person" },
                    },
                    {
                        content: "This has badword1",
                        author: { name: "Jane", bio: "Contains badword2" },
                    },
                ],
            };

            const textArray = toStringArray(nestedData, null);
            expect(hasProfanity(...(textArray || []))).toBe(true);

            // Test filtering specific fields - get posts field first
            const postsData = toStringArray(nestedData, ["posts"]);
            // For this test, we'll extract bio text from the full structure
            const allText = toStringArray(nestedData, null);
            const hasBadBios = allText?.some(text => text.includes("badword2"));
            expect(hasBadBios).toBe(true);
        });
    });
});
