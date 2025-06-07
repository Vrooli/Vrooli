import { describe, it, expect } from "vitest";
import { getReactionScore, removeModifiers } from "./emotions.js";

describe("Emoji Reactions Utility Functions", () => {
    describe("removeModifiers", () => {
        it("should remove skin tone from thumbs up emoji", () => {
            const reaction = "👍🏻";
            const result = removeModifiers(reaction);
            expect(result).to.equal("👍");
        });

        it("should return the same emoji if no modifiers are present", () => {
            const reaction = "🚀";
            const result = removeModifiers(reaction);
            expect(result).to.equal("🚀");
        });

        it("should handle non-string inputs gracefully", () => {
            const reaction = null;
            const result = removeModifiers(reaction as unknown as string);
            expect(result).to.equal("");
        });
    });

    describe("getReactionScore", () => {
        it("should return 1 for positive reactions", () => {
            const reaction = "👍";
            const score = getReactionScore(reaction);
            expect(score).to.equal(1);
        });

        it("should return -1 for negative reactions", () => {
            const reaction = "👎";
            const score = getReactionScore(reaction);
            expect(score).to.equal(-1);
        });

        it("should return 0 for neutral reactions", () => {
            const reaction = "🐰";
            const score = getReactionScore(reaction);
            expect(score).to.equal(0);
        });

        it("should handle reactions with modifiers correctly", () => {
            const reaction = "👍🏻";
            const score = getReactionScore(reaction);
            expect(score).to.equal(1);
        });

        it("should return 0 for null or undefined reactions", () => {
            const reaction = null;
            const score = getReactionScore(reaction);
            expect(score).to.equal(0);
        });
    });
});
