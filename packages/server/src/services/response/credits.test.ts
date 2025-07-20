import { describe, it, expect } from "vitest";
import { calculateMaxCredits } from "./credits.js";

describe("calculateMaxCredits", () => {
    describe("basic functionality", () => {
        it("should return the user's remaining credits when less than effective task max", () => {
            const result = calculateMaxCredits(500000, 800000, 100000);
            expect(result).toBe(500000n);
        });

        it("should return effective task max when user has more credits", () => {
            const result = calculateMaxCredits(1500000, 800000, 100000);
            expect(result).toBe(700000n); // 800000 - 100000
        });

        it("should handle case where no credits have been spent", () => {
            const result = calculateMaxCredits(500000, 1000000, 0);
            expect(result).toBe(500000n);
        });

        it("should handle case where user credits equal effective task max", () => {
            const result = calculateMaxCredits(700000, 800000, 100000);
            expect(result).toBe(700000n);
        });
    });

    describe("type conversions", () => {
        it("should handle string inputs", () => {
            const result = calculateMaxCredits("750000", "800000", "50000");
            expect(result).toBe(750000n);
        });

        it("should handle bigint inputs", () => {
            const result = calculateMaxCredits(2000000n, 1000000n, 200000n);
            expect(result).toBe(800000n); // 1000000 - 200000
        });

        it("should handle mixed type inputs", () => {
            const result = calculateMaxCredits("1500000", 1000000n, "200000");
            expect(result).toBe(800000n);
        });

        it("should handle very large numbers as strings", () => {
            const result = calculateMaxCredits(
                "9999999999999999999",
                "10000000000000000000",
                "1000000000000000000",
            );
            expect(result).toBe(9000000000000000000n);
        });
    });

    describe("edge cases with zero values", () => {
        it("should return 0n when user has no remaining credits", () => {
            const result = calculateMaxCredits(0, 1000000, 0);
            expect(result).toBe(0n);
        });

        it("should return 0n when task max credits is 0", () => {
            const result = calculateMaxCredits(500000, 0, 0);
            expect(result).toBe(0n);
        });

        it("should return 0n when both user credits and task max are 0", () => {
            const result = calculateMaxCredits(0, 0, 0);
            expect(result).toBe(0n);
        });

        it("should handle string '0' values", () => {
            const result = calculateMaxCredits("0", "1000000", "0");
            expect(result).toBe(0n);
        });
    });

    describe("negative value handling", () => {
        it("should return 0n when user has negative credits", () => {
            const result = calculateMaxCredits(-500000, 1000000, 0);
            expect(result).toBe(0n);
        });

        it("should return 0n when task max is negative", () => {
            const result = calculateMaxCredits(500000, -1000000, 0);
            expect(result).toBe(0n);
        });

        it("should return 0n when credits spent is negative", () => {
            const result = calculateMaxCredits(500000, 1000000, -100000);
            expect(result).toBe(0n);
        });

        it("should handle negative string values", () => {
            const result = calculateMaxCredits("-500000", "1000000", "0");
            expect(result).toBe(0n);
        });
    });

    describe("credits already spent scenarios", () => {
        it("should return 0n when all task credits have been spent", () => {
            const result = calculateMaxCredits(500000, 1000000, 1000000);
            expect(result).toBe(0n);
        });

        it("should return 0n when more credits spent than task max", () => {
            const result = calculateMaxCredits(500000, 1000000, 1500000);
            expect(result).toBe(0n);
        });

        it("should correctly calculate remaining task budget", () => {
            const result = calculateMaxCredits(1000000, 500000, 100000);
            expect(result).toBe(400000n); // 500000 - 100000
        });
    });

    describe("undefined and null handling", () => {
        it("should treat undefined userRemainingCredits as 0", () => {
            const result = calculateMaxCredits(undefined as unknown, 1000000, 0);
            expect(result).toBe(0n);
        });

        it("should treat undefined taskMaxCredits as 0", () => {
            const result = calculateMaxCredits(500000, undefined as unknown, 0);
            expect(result).toBe(0n);
        });

        it("should treat undefined creditsSpent as 0", () => {
            const result = calculateMaxCredits(500000, 1000000, undefined);
            expect(result).toBe(500000n);
        });

        it("should handle null values by treating them as 0", () => {
            const result = calculateMaxCredits(null as unknown, 1000000, 0);
            expect(result).toBe(0n);
        });
    });

    describe("extreme values and overflow protection", () => {
        it("should handle maximum safe integer values", () => {
            const maxSafeInt = Number.MAX_SAFE_INTEGER.toString();
            const result = calculateMaxCredits(maxSafeInt, maxSafeInt, "0");
            expect(result).toBe(BigInt(maxSafeInt));
        });

        it("should handle values beyond MAX_SAFE_INTEGER", () => {
            const bigValue = "99999999999999999999999999999";
            const result = calculateMaxCredits(bigValue, bigValue, "0");
            expect(result).toBe(BigInt(bigValue));
        });

        it("should correctly subtract very large numbers", () => {
            const taskMax = "1000000000000000000000";
            const spent = "999999999999999999999";
            const userRemaining = "10000000000000000000000";
            const result = calculateMaxCredits(userRemaining, taskMax, spent);
            expect(result).toBe(1n); // taskMax - spent = 1
        });
    });

    describe("real-world scenarios", () => {
        it("should handle typical free tier limits", () => {
            const userCredits = 10000; // User has 10k credits
            const taskMax = 5000;       // Task allows max 5k
            const spent = 1000;         // Already used 1k
            const result = calculateMaxCredits(userCredits, taskMax, spent);
            expect(result).toBe(4000n);  // Can use 4k more
        });

        it("should handle premium user with high limits", () => {
            const userCredits = "1000000000"; // 1 billion credits
            const taskMax = "50000000";       // Task allows 50 million
            const spent = "10000000";         // Already used 10 million
            const result = calculateMaxCredits(userCredits, taskMax, spent);
            expect(result).toBe(40000000n);   // Can use 40 million more
        });

        it("should handle user running low on credits mid-task", () => {
            const userCredits = 100;      // Only 100 credits left
            const taskMax = 10000;        // Task allows 10k total
            const spent = 5000;           // Already used 5k
            const result = calculateMaxCredits(userCredits, taskMax, spent);
            expect(result).toBe(100n);    // Limited by user's remaining credits
        });
    });

    describe("documentation examples", () => {
        it("should match example 1: calculateMaxCredits(500000, 800000, 100000) => 500000n", () => {
            const result = calculateMaxCredits(500000, 800000, 100000);
            expect(result).toBe(500000n);
        });

        it("should match example 2: calculateMaxCredits('750000', 800000, '50000') => 750000n", () => {
            const result = calculateMaxCredits("750000", 800000, "50000");
            expect(result).toBe(750000n);
        });

        it("should match example 3: calculateMaxCredits(1500000n, '1000000', 200000) => 800000n", () => {
            const result = calculateMaxCredits(1500000n, "1000000", 200000);
            expect(result).toBe(800000n);
        });
    });

    describe("error handling and robustness", () => {
        it("should handle empty strings as 0", () => {
            const result = calculateMaxCredits("", "1000000", "0");
            expect(result).toBe(0n);
        });

        it("should handle whitespace strings as 0", () => {
            const result = calculateMaxCredits("   ", "1000000", "0");
            expect(result).toBe(0n);
        });

        it("should handle boolean false as 0", () => {
            const result = calculateMaxCredits(false as unknown, 1000000, 0);
            expect(result).toBe(0n);
        });

        it("should handle boolean true as 1", () => {
            const result = calculateMaxCredits(true as unknown, 1000000, 0);
            expect(result).toBe(1n); // true converts to 1n, and with no credits spent, userRemaining (1) < taskMax (1000000)
        });

        it("should handle when all inputs are falsy", () => {
            const result = calculateMaxCredits(null as unknown, undefined as unknown, "");
            expect(result).toBe(0n);
        });
    });
});
