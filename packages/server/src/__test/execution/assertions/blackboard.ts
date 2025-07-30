/**
 * Blackboard Assertions
 * 
 * Custom assertions for testing blackboard state
 */

import { expect } from "vitest";

export interface BlackboardAssertions {
    toHaveKey(key: string): void;
    toHaveKeys(keys: string[]): void;
    toHaveValue(key: string, value: any): void;
    toMatchState(expectedState: Record<string, any>): void;
    toContainState(partialState: Record<string, any>): void;
}


export function extendBlackboardAssertions() {
    expect.extend({
        toHaveKey(received: Map<string, any>, key: string) {
            const pass = received.has(key);
            return {
                pass,
                message: () => pass
                    ? `Expected blackboard not to have key "${key}"`
                    : `Expected blackboard to have key "${key}"`,
            };
        },

        toHaveKeys(received: Map<string, any>, keys: string[]) {
            const missingKeys = keys.filter(key => !received.has(key));
            const pass = missingKeys.length === 0;

            return {
                pass,
                message: () => pass
                    ? `Expected blackboard not to have keys ${keys.join(", ")}`
                    : `Expected blackboard to have keys ${keys.join(", ")}, missing: ${missingKeys.join(", ")}`,
            };
        },

        toHaveValue(received: Map<string, any>, key: string, expectedValue: any) {
            const hasKey = received.has(key);
            const actualValue = received.get(key);
            const valuesMatch = JSON.stringify(actualValue) === JSON.stringify(expectedValue);
            const pass = hasKey && valuesMatch;

            return {
                pass,
                message: () => {
                    if (!hasKey) {
                        return `Expected blackboard to have key "${key}"`;
                    }
                    if (!valuesMatch) {
                        return `Expected blackboard["${key}"] to be ${JSON.stringify(expectedValue)}, but got ${JSON.stringify(actualValue)}`;
                    }
                    return `Expected blackboard["${key}"] not to be ${JSON.stringify(expectedValue)}`;
                },
            };
        },

        toMatchState(received: Map<string, any>, expectedState: Record<string, any>) {
            const actualState: Record<string, any> = {};
            for (const [key, value] of Array.from(received.entries())) {
                actualState[key] = value;
            }

            const pass = JSON.stringify(actualState) === JSON.stringify(expectedState);

            return {
                pass,
                message: () => pass
                    ? "Expected blackboard state not to match"
                    : `Expected blackboard state to match:\n${JSON.stringify(expectedState, null, 2)}\n\nActual:\n${JSON.stringify(actualState, null, 2)}`,
            };
        },

        toContainState(received: Map<string, any>, partialState: Record<string, any>) {
            const mismatches: string[] = [];

            for (const [key, expectedValue] of Object.entries(partialState)) {
                if (!received.has(key)) {
                    mismatches.push(`Missing key: ${key}`);
                } else {
                    const actualValue = received.get(key);
                    if (JSON.stringify(actualValue) !== JSON.stringify(expectedValue)) {
                        mismatches.push(`${key}: expected ${JSON.stringify(expectedValue)}, got ${JSON.stringify(actualValue)}`);
                    }
                }
            }

            const pass = mismatches.length === 0;

            return {
                pass,
                message: () => pass
                    ? "Expected blackboard not to contain state"
                    : `Expected blackboard to contain state:\n${mismatches.join("\n")}`,
            };
        },
    });
}
