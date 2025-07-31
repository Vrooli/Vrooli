/**
 * Outcome Assertions
 * 
 * Custom assertions for validating scenario outcomes
 */

import { expect } from "vitest";
import type { RoutineCall } from "../factories/scenario/types.js";

export interface OutcomeAssertions {
    toMatchCalls(expectedCalls: Array<{ routine: string; times: number }>): void;
    toHaveSuccessRate(minRate: number): void;
    toCompleteWithin(maxDuration: number): void;
    toUseCreditsWithin(maxCredits: number): void;
}


export function extendOutcomeAssertions() {
    expect.extend({
        toMatchCalls(received: RoutineCall[], expectedCalls: Array<{ routine: string; times: number }>) {
            const callCounts = new Map<string, number>();
            
            // Count actual calls
            for (const call of received) {
                const count = callCounts.get(call.routineLabel) || 0;
                callCounts.set(call.routineLabel, count + 1);
            }

            // Check expectations
            const mismatches: string[] = [];
            for (const expected of expectedCalls) {
                const actualCount = callCounts.get(expected.routine) || 0;
                if (actualCount !== expected.times) {
                    mismatches.push(
                        `${expected.routine}: expected ${expected.times} calls, got ${actualCount}`,
                    );
                }
            }

            const pass = mismatches.length === 0;

            return {
                pass,
                message: () => pass
                    ? "Expected routine calls not to match"
                    : `Routine call count mismatches:\n${mismatches.join("\n")}`,
            };
        },

        toHaveSuccessRate(received: RoutineCall[], minRate: number) {
            if (received.length === 0) {
                return {
                    pass: false,
                    message: () => `Expected success rate of ${minRate}%, but no routine calls found`,
                };
            }

            const successCount = received.filter(c => c.success).length;
            const actualRate = (successCount / received.length) * 100;
            const pass = actualRate >= minRate;

            return {
                pass,
                message: () => pass
                    ? `Expected success rate less than ${minRate}%, but got ${actualRate.toFixed(1)}%`
                    : `Expected success rate of at least ${minRate}%, but got ${actualRate.toFixed(1)}% (${successCount}/${received.length} successful)`,
            };
        },

        toCompleteWithin(received: { duration: number }, maxDuration: number) {
            const pass = received.duration <= maxDuration;

            return {
                pass,
                message: () => pass
                    ? `Expected to take longer than ${maxDuration}ms, but completed in ${received.duration}ms`
                    : `Expected to complete within ${maxDuration}ms, but took ${received.duration}ms`,
            };
        },

        toUseCreditsWithin(received: { creditsUsed?: string }, maxCredits: number) {
            const creditsUsed = parseInt(received.creditsUsed || "0", 10);
            const pass = creditsUsed <= maxCredits;

            return {
                pass,
                message: () => pass
                    ? `Expected to use more than ${maxCredits} credits, but used ${creditsUsed}`
                    : `Expected to use at most ${maxCredits} credits, but used ${creditsUsed}`,
            };
        },
    });
}
