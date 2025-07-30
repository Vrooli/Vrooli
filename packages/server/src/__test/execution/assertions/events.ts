/**
 * Event Assertions
 * 
 * Custom assertions for testing event sequences
 */

import { expect } from "vitest";
import type { ScenarioEvent } from "../factories/scenario/types.js";

export interface EventAssertions {
    toMatchSequence(expectedSequence: string[]): void;
    toContainEvents(expectedEvents: string[]): void;
    toHaveEventCount(topic: string, count: number): void;
    toHaveEventWithData(topic: string, data: any): void;
}


export function extendEventAssertions() {
    expect.extend({
        toMatchSequence(received: ScenarioEvent[], expectedSequence: string[]) {
            const actualSequence = received.map(e => e.topic);

            // Check if events occurred in order
            let sequenceIndex = 0;
            const matchedIndices: number[] = [];

            for (let i = 0; i < actualSequence.length; i++) {
                if (actualSequence[i] === expectedSequence[sequenceIndex]) {
                    matchedIndices.push(i);
                    sequenceIndex++;
                    if (sequenceIndex === expectedSequence.length) break;
                }
            }

            const pass = sequenceIndex === expectedSequence.length;

            return {
                pass,
                message: () => {
                    if (pass) {
                        return `Expected events not to match sequence ${expectedSequence.join(" → ")}`;
                    }

                    const matched = expectedSequence.slice(0, sequenceIndex);
                    const unmatched = expectedSequence.slice(sequenceIndex);

                    return "Expected events to match sequence:\n" +
                        `✓ Matched: ${matched.join(" → ")}\n` +
                        `✗ Missing: ${unmatched.join(" → ")}\n` +
                        `Actual events: ${actualSequence.join(", ")}`;
                },
            };
        },

        toContainEvents(received: ScenarioEvent[], expectedEvents: string[]) {
            const actualTopics = received.map(e => e.topic);
            const missingEvents = expectedEvents.filter(e => !actualTopics.includes(e));
            const pass = missingEvents.length === 0;

            return {
                pass,
                message: () => pass
                    ? `Expected not to contain events ${expectedEvents.join(", ")}`
                    : "Expected to contain events:\n" +
                    `✓ Found: ${expectedEvents.filter(e => actualTopics.includes(e)).join(", ")}\n` +
                    `✗ Missing: ${missingEvents.join(", ")}`,
            };
        },

        toHaveEventCount(received: ScenarioEvent[], topic: string, expectedCount: number) {
            const actualCount = received.filter(e => e.topic === topic).length;
            const pass = actualCount === expectedCount;

            return {
                pass,
                message: () => pass
                    ? `Expected not to have ${expectedCount} "${topic}" events`
                    : `Expected ${expectedCount} "${topic}" events, but found ${actualCount}`,
            };
        },

        toHaveEventWithData(received: ScenarioEvent[], topic: string, expectedData: any) {
            const matchingEvents = received.filter(e =>
                e.topic === topic &&
                JSON.stringify(e.data) === JSON.stringify(expectedData),
            );
            const pass = matchingEvents.length > 0;

            return {
                pass,
                message: () => {
                    if (pass) {
                        return `Expected not to have "${topic}" event with data ${JSON.stringify(expectedData)}`;
                    }

                    const topicEvents = received.filter(e => e.topic === topic);
                    if (topicEvents.length === 0) {
                        return `Expected to have "${topic}" event with data, but no "${topic}" events found`;
                    }

                    return `Expected to have "${topic}" event with data:\n` +
                        `Expected: ${JSON.stringify(expectedData, null, 2)}\n` +
                        `Actual events data:\n${topicEvents.map(e => JSON.stringify(e.data, null, 2)).join("\n---\n")}`;
                },
            };
        },
    });
}
