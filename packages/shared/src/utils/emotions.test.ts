// AI_CHECK: TEST_COVERAGE=7 | LAST: 2025-06-18
import { describe, it, expect } from "vitest";
import { getReactionScore, removeModifiers } from "./emotions.js";

describe("Emoji Reactions Utility Functions", () => {
    describe("removeModifiers", () => {
        it("should remove skin tone from thumbs up emoji", () => {
            const reaction = "👍🏻";
            const result = removeModifiers(reaction);
            expect(result).toBe("👍");
        });

        it("should return the same emoji if no modifiers are present", () => {
            const reaction = "🚀";
            const result = removeModifiers(reaction);
            expect(result).toBe("🚀");
        });

        it("should handle non-string inputs gracefully", () => {
            const reaction = null;
            const result = removeModifiers(reaction as unknown as string);
            expect(result).toBe("");
        });
    });

    describe("getReactionScore", () => {
        it("should return 1 for positive reactions", () => {
            const reaction = "👍";
            const score = getReactionScore(reaction);
            expect(score).toBe(1);
        });

        it("should return -1 for negative reactions", () => {
            const reaction = "👎";
            const score = getReactionScore(reaction);
            expect(score).toBe(-1);
        });

        it("should return 0 for neutral reactions", () => {
            const reaction = "🐰";
            const score = getReactionScore(reaction);
            expect(score).toBe(0);
        });

        it("should handle reactions with modifiers correctly", () => {
            const reaction = "👍🏻";
            const score = getReactionScore(reaction);
            expect(score).toBe(1);
        });

        it("should return 0 for null or undefined reactions", () => {
            const reaction = null;
            const score = getReactionScore(reaction);
            expect(score).toBe(0);
        });

        it("should return 0 for undefined reactions", () => {
            const score = getReactionScore(undefined);
            expect(score).toBe(0);
        });

        it("should handle all positive reactions correctly", () => {
            const positiveEmojis = ["👍", "👏", "🎉", "🥳", "😊", "😃", "😄", "😁", "😇", "❤️", "🥰", "💖", "😍", "🚀", "👀", "🔥", "🎊", "🙌", "👌", "👊", "💯", "🤘", "🤙", "🤟", "🤝"];
            
            positiveEmojis.forEach(emoji => {
                expect(getReactionScore(emoji)).toBe(1);
            });
        });

        it("should handle all negative reactions correctly", () => {
            const negativeEmojis = ["👎", "😕", "😡", "😠", "🤬", "😞", "😟", "😨", "🤮", "🤢", "🤧", "🤒", "🤕", "🤡", "🤥", "🤦", "🙅‍♂️"];
            
            negativeEmojis.forEach(emoji => {
                expect(getReactionScore(emoji)).toBe(-1);
            });
        });

        it("should handle empty string", () => {
            expect(getReactionScore("")).toBe(0);
        });

        it("should handle random text that isn't an emoji", () => {
            expect(getReactionScore("hello")).toBe(0);
            expect(getReactionScore("123")).toBe(0);
            expect(getReactionScore("@#$%")).toBe(0);
        });
    });

    describe("removeModifiers edge cases", () => {
        it("should handle multiple different skin tone modifiers", () => {
            // Test all skin tone variants
            expect(removeModifiers("👍🏻")).toBe("👍"); // Light skin tone
            expect(removeModifiers("👍🏼")).toBe("👍"); // Medium-light skin tone
            expect(removeModifiers("👍🏽")).toBe("👍"); // Medium skin tone
            expect(removeModifiers("👍🏾")).toBe("👍"); // Medium-dark skin tone
            expect(removeModifiers("👍🏿")).toBe("👍"); // Dark skin tone
        });

        it("should handle multiple emojis in one string", () => {
            expect(removeModifiers("👍🏻👎🏿")).toBe("👍👎");
            expect(removeModifiers("🙌🏽🎉")).toBe("🙌🎉");
        });

        it("should handle empty string", () => {
            expect(removeModifiers("")).toBe("");
        });

        it("should handle undefined as string type", () => {
            expect(removeModifiers(undefined as unknown as string)).toBe("");
        });

        it("should handle numbers as string type", () => {
            expect(removeModifiers(123 as unknown as string)).toBe("");
        });

        it("should handle boolean as string type", () => {
            expect(removeModifiers(true as unknown as string)).toBe("");
            expect(removeModifiers(false as unknown as string)).toBe("");
        });

        it("should handle objects as string type", () => {
            expect(removeModifiers({} as unknown as string)).toBe("");
            expect(removeModifiers({ emoji: "👍" } as unknown as string)).toBe("");
        });

        it("should handle regular text without emojis", () => {
            expect(removeModifiers("hello world")).toBe("hello world");
            expect(removeModifiers("123456")).toBe("123456");
        });

        it("should preserve variation selectors and ZWJ sequences", () => {
            // The function should only remove skin tone modifiers, not other unicode sequences
            // This is complex Unicode behavior, but we test that the function doesn't break on them
            expect(removeModifiers("👨‍👩‍👧‍👦")).toBe("👨‍👩‍👧‍👦"); // Family emoji with ZWJ
            expect(removeModifiers("🏳️‍🌈")).toBe("🏳️‍🌈"); // Flag with variation selector
        });
    });
});
