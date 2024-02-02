/* eslint-disable @typescript-eslint/ban-ts-comment */
import { nextRewardThreshold, reputationDeltaStar, reputationDeltaVote } from "./reputation";

describe("nextRewardThreshold", () => {
    test("positive direction, increasing values", () => {
        expect(nextRewardThreshold(1, 1)).toBe(2);
        expect(nextRewardThreshold(9, 1)).toBe(10);
        expect(nextRewardThreshold(10, 1)).toBe(20);
        expect(nextRewardThreshold(98, 1)).toBe(100);
        expect(nextRewardThreshold(90, 1)).toBe(100);
        expect(nextRewardThreshold(100, 1)).toBe(200);
        expect(nextRewardThreshold(101, 1)).toBe(200);
        expect(nextRewardThreshold(999, 1)).toBe(1000);
    });

    test("positive direction, zero and negative values", () => {
        expect(nextRewardThreshold(0, 1)).toBe(1);
        expect(nextRewardThreshold(-1, 1)).toBe(0);
        expect(nextRewardThreshold(-10, 1)).toBe(-9);
        expect(nextRewardThreshold(-11, 1)).toBe(-10);
        expect(nextRewardThreshold(-20, 1)).toBe(-10);
        expect(nextRewardThreshold(-19, 1)).toBe(-10);
        expect(nextRewardThreshold(-999, 1)).toBe(-900);
        expect(nextRewardThreshold(-1001, 1)).toBe(-1000);
    });

    test("negative direction, negative values", () => {
        expect(nextRewardThreshold(-1001, -1)).toBe(-2000);
        expect(nextRewardThreshold(-1000, -1)).toBe(-2000);
        expect(nextRewardThreshold(-999, -1)).toBe(-1000);
    });

    test("negative direction, zero and positive values", () => {
        expect(nextRewardThreshold(0, -1)).toBe(-1);
        expect(nextRewardThreshold(-9, -1)).toBe(-10);
        expect(nextRewardThreshold(-10, -1)).toBe(-20);
        expect(nextRewardThreshold(1, -1)).toBe(0);
        expect(nextRewardThreshold(9, -1)).toBe(8);
        expect(nextRewardThreshold(99, -1)).toBe(90);
        expect(nextRewardThreshold(1000, -1)).toBe(900);
    });
});

describe("reputationDeltaVote", () => {
    // No change in votes
    test("should return 0 when vote count does not change", () => {
        expect(reputationDeltaVote(5, 5)).toBe(0);
    });

    // Small vote counts change
    test("should return 1 when vote increases within -9 to 9 range", () => {
        expect(reputationDeltaVote(5, 6)).toBe(1);
    });

    test("should return -1 when vote decreases within -9 to 9 range", () => {
        expect(reputationDeltaVote(-5, -6)).toBe(-1);
    });

    // Medium vote counts change
    test("should return 1 when vote increases from 9 to 10", () => {
        expect(reputationDeltaVote(9, 10)).toBe(1);
    });

    test("should return 0 when vote increases within 10 to 99 range but not to a multiple of 10", () => {
        expect(reputationDeltaVote(20, 21)).toBe(0);
    });

    // Large vote counts change
    test("should return 1 when vote increases from 99 to 100", () => {
        expect(reputationDeltaVote(99, 100)).toBe(1);
    });

    test("should return 0 when vote increases within 100 to 999 range but not to a multiple of 100", () => {
        expect(reputationDeltaVote(200, 201)).toBe(0);
    });

    // Edge Cases
    test("should handle edge case of transitioning from -9 to -10", () => {
        expect(reputationDeltaVote(-9, -10)).toBe(-1);
    });

    // Very large numbers
    test("should return 1 when vote increases to a multiple of 1000", () => {
        expect(reputationDeltaVote(999, 1000)).toBe(1);
    });

    // Invalid inputs - Assuming the function is intended to handle only numeric inputs
    test("should throw error on non-numeric inputs", () => {
        // @ts-ignore: Testing runtime scenario
        expect(reputationDeltaVote("a", 10)).toBe(NaN);
    });

    // Large delta
    test("should return +10 when vote goes from 0 to 10", () => {
        expect(reputationDeltaVote(0, 10)).toBe(10);
    });

    test("should return +10 when vote goes from 0 to 19", () => {
        expect(reputationDeltaVote(0, 19)).toBe(10);
    });

    test("should return +11 when vote goes from 0 to 20", () => {
        expect(reputationDeltaVote(0, 20)).toBe(11);
    });

    test("should return +18 when vote goes from 0 to 99", () => {
        expect(reputationDeltaVote(0, 99)).toBe(18);
    });

    test("should return +19 when vote goes from 0 to 100", () => {
        expect(reputationDeltaVote(0, 100)).toBe(19);
    });

    // Single large increment followed by multiple small decrements
    test("should have no net reputation change after a large increase and multiple small decreases", () => {
        let reputation = 0;
        reputation += reputationDeltaVote(0, 100);
        expect(reputation).toBe(19);
        let currentVote = 100;
        let nextVote = 95;
        for (let i = 0; i < 20; i++) {
            // Go down by 5 each time, so the net change is 0
            reputation += reputationDeltaVote(currentVote, nextVote);
            currentVote = nextVote;
            nextVote -= 5;
        }
        expect(currentVote).toBe(0); // Original vote count
        expect(reputation).toBe(0); // Net zero change
    });

    // Multiple small increments followed by a small decrement
    test("should have no net reputation change after small increments to a threshold and a small decrement - positive numbers", () => {
        let reputation = 0;
        for (let i = 0; i < 10; i++) {
            reputation += reputationDeltaVote(i, i + 1); // +1 for reaching 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, for a total of +10
        }
        expect(reputation).toBe(10); // Net change of +10
        reputation += reputationDeltaVote(10, 9); // -1 for decrementing from 10
        expect(reputation).toBe(9); // Net change of +9
    });

    test("should have no net reputation change after small increments to a threshold and a small decrement - negative numbers", () => {
        let reputation = 0;
        for (let i = 0; i > -10; i--) {
            reputation += reputationDeltaVote(i, i - 1); // +1 for reaching -1, -2, -3, -4, -5, -6, -7, -8, -9, -10, for a total of +10
        }
        expect(reputation).toBe(-10); // Net change of -10
        reputation += reputationDeltaVote(-10, -9); // +1 for decrementing from -10
        expect(reputation).toBe(-9); // Net change of -9
    });

    // Fluctuations around a multiple of 10
    test("should have no net reputation change with fluctuations around a multiple of 10 - positive numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaVote(9, 10); // +1 for reaching 10
        reputation += reputationDeltaVote(10, 9); // -1 for decrementing from 10
        expect(reputation).toBe(0); // Net zero change
    });

    test("should have no net reputation change with fluctuations around a multiple of 10 - negative numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaVote(-9, -10); // -1 for reaching -10
        expect(reputation).toBe(-1); // Net change of -1
        reputation += reputationDeltaVote(-10, -9); // +1 for incrementing from -10
        expect(reputation).toBe(0); // Net zero change
    });

    // Single large decrement followed by multiple small increments
    test("should have no net reputation change after a large decrease and multiple small increases - positive numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaVote(100, 0);
        expect(reputation).toBe(-19);
        let currentVote = 0;
        let nextVote = 5;
        for (let i = 0; i < 20; i++) {
            // Go up by 5 each time, so the net change is 0
            reputation += reputationDeltaVote(currentVote, nextVote);
            currentVote = nextVote;
            nextVote += 5;
        }
        expect(currentVote).toBe(100); // Original vote count
        expect(reputation).toBe(0); // Net zero change
    });

    test("should have no net reputation change after a large decrease and multiple small increases - negative numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaVote(-100, 0);
        expect(reputation).toBe(19);
        let currentVote = 0;
        let nextVote = -5;
        for (let i = 0; i < 20; i++) {
            // Go down by 5 each time, so the net change is 0
            reputation += reputationDeltaVote(currentVote, nextVote);
            currentVote = nextVote;
            nextVote -= 5;
        }
        expect(currentVote).toBe(-100); // Original vote count
        expect(reputation).toBe(0); // Net zero change
    });

    test("should handle going between positive and negative numbers correctly", () => {
        let reputation = 0;
        reputation += reputationDeltaVote(0, 100);
        expect(reputation).toBe(19);
        reputation += reputationDeltaVote(100, -100);
        expect(reputation).toBe(-19);
        reputation += reputationDeltaVote(-100, 0);
        expect(reputation).toBe(0); // Net zero change
    });
});

// NOTE: These are the same tests as reputationDeltaVote, but reputation 
// is doubled
describe("reputationDeltaStar", () => {
    // No change in votes
    test("should return 0 when vote count does not change", () => {
        expect(reputationDeltaStar(5, 5)).toBe(0);
    });

    // Small vote counts change
    test("should return 2 when vote increases within -9 to 9 range", () => {
        expect(reputationDeltaStar(5, 6)).toBe(2);
    });

    test("should return -2 when vote decreases within -9 to 9 range", () => {
        expect(reputationDeltaStar(-5, -6)).toBe(-2);
    });

    // Medium vote counts change
    test("should return 2 when vote increases from 9 to 10", () => {
        expect(reputationDeltaStar(9, 10)).toBe(2);
    });

    test("should return 0 when vote increases within 10 to 99 range but not to a multiple of 10", () => {
        expect(reputationDeltaStar(20, 21)).toBe(0);
    });

    // Large vote counts change
    test("should return 2 when vote increases from 99 to 100", () => {
        expect(reputationDeltaStar(99, 100)).toBe(2);
    });

    test("should return 0 when vote increases within 100 to 999 range but not to a multiple of 100", () => {
        expect(reputationDeltaStar(200, 201)).toBe(0);
    });

    // Edge Cases
    test("should handle edge case of transitioning from -9 to -10", () => {
        expect(reputationDeltaStar(-9, -10)).toBe(-2);
    });

    // Very large numbers
    test("should return 2 when vote increases to a multiple of 1000", () => {
        expect(reputationDeltaStar(999, 1000)).toBe(2);
    });

    // Invalid inputs - Assuming the function is intended to handle only numeric inputs
    test("should throw error on non-numeric inputs", () => {
        // @ts-ignore: Testing runtime scenario
        expect(reputationDeltaStar("a", 10)).toBe(NaN);
    });

    // Large delta
    test("should return +20 when vote goes from 0 to 10", () => {
        expect(reputationDeltaStar(0, 10)).toBe(20);
    });

    test("should return +20 when vote goes from 0 to 19", () => {
        expect(reputationDeltaStar(0, 19)).toBe(20);
    });

    test("should return +22 when vote goes from 0 to 20", () => {
        expect(reputationDeltaStar(0, 20)).toBe(22);
    });

    test("should return +36 when vote goes from 0 to 99", () => {
        expect(reputationDeltaStar(0, 99)).toBe(36);
    });

    test("should return +38 when vote goes from 0 to 100", () => {
        expect(reputationDeltaStar(0, 100)).toBe(38);
    });

    // Single large increment followed by multiple small decrements
    test("should have no net reputation change after a large increase and multiple small decreases", () => {
        let reputation = 0;
        reputation += reputationDeltaStar(0, 100);
        expect(reputation).toBe(38);
        let currentVote = 100;
        let nextVote = 95;
        for (let i = 0; i < 20; i++) {
            // Go down by 5 each time, so the net change is 0
            reputation += reputationDeltaStar(currentVote, nextVote);
            currentVote = nextVote;
            nextVote -= 5;
        }
        expect(currentVote).toBe(0); // Original vote count
        expect(reputation).toBe(0); // Net zero change
    });

    // Multiple small increments followed by a small decrement
    test("should have no net reputation change after small increments to a threshold and a small decrement - positive numbers", () => {
        let reputation = 0;
        for (let i = 0; i < 10; i++) {
            reputation += reputationDeltaStar(i, i + 1); // +1 for reaching 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, for a total of +10
        }
        expect(reputation).toBe(20); // Net change of +20
        reputation += reputationDeltaStar(10, 9); // -2 for decrementing from 10
        expect(reputation).toBe(18); // Net change of +18
    });

    test("should have no net reputation change after small increments to a threshold and a small decrement - negative numbers", () => {
        let reputation = 0;
        for (let i = 0; i > -10; i--) {
            reputation += reputationDeltaStar(i, i - 1); // +1 for reaching -1, -2, -3, -4, -5, -6, -7, -8, -9, -10, for a total of +10
        }
        expect(reputation).toBe(-20); // Net change of -20
        reputation += reputationDeltaStar(-10, -9); // +2 for decrementing from -10
        expect(reputation).toBe(-18); // Net change of -18
    });

    // Fluctuations around a multiple of 10
    test("should have no net reputation change with fluctuations around a multiple of 10 - positive numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaStar(9, 10); // +2 for reaching 10
        reputation += reputationDeltaStar(10, 9); // -2 for decrementing from 10
        expect(reputation).toBe(0); // Net zero change
    });

    test("should have no net reputation change with fluctuations around a multiple of 10 - negative numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaStar(-9, -10); // -2 for reaching -10
        expect(reputation).toBe(-2);
        reputation += reputationDeltaStar(-10, -9); // +2 for incrementing from -10
        expect(reputation).toBe(0); // Net zero change
    });

    // Single large decrement followed by multiple small increments
    test("should have no net reputation change after a large decrease and multiple small increases - positive numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaStar(100, 0);
        expect(reputation).toBe(-38);
        let currentVote = 0;
        let nextVote = 5;
        for (let i = 0; i < 20; i++) {
            // Go up by 5 each time, so the net change is 0
            reputation += reputationDeltaStar(currentVote, nextVote);
            currentVote = nextVote;
            nextVote += 5;
        }
        expect(currentVote).toBe(100); // Original vote count
        expect(reputation).toBe(0); // Net zero change
    });

    test("should have no net reputation change after a large decrease and multiple small increases - negative numbers", () => {
        let reputation = 0;
        reputation += reputationDeltaStar(-100, 0);
        expect(reputation).toBe(38);
        let currentVote = 0;
        let nextVote = -5;
        for (let i = 0; i < 20; i++) {
            // Go down by 5 each time, so the net change is 0
            reputation += reputationDeltaStar(currentVote, nextVote);
            currentVote = nextVote;
            nextVote -= 5;
        }
        expect(currentVote).toBe(-100); // Original vote count
        expect(reputation).toBe(0); // Net zero change
    });

    test("should handle going between positive and negative numbers correctly", () => {
        let reputation = 0;
        reputation += reputationDeltaStar(0, 100);
        expect(reputation).toBe(38);
        reputation += reputationDeltaStar(100, -100);
        expect(reputation).toBe(-38);
        reputation += reputationDeltaStar(-100, 0);
        expect(reputation).toBe(0); // Net zero change
    });
});
