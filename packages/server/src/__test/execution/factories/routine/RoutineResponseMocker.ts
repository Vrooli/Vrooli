/**
 * Routine Response Mocker
 * 
 * Manages mock responses for routine executions in tests
 */

import type { MockResponse } from "../../types.js";

export class RoutineResponseMocker {
    private mockRegistry = new Map<string, MockResponse[]>();
    private attemptCounters = new Map<string, number>();

    async register(routineId: string, responses: MockResponse[]): Promise<void> {
        this.mockRegistry.set(routineId, responses);
        this.attemptCounters.set(routineId, 0);
    }

    async getMockResponse(routineId: string, input?: Record<string, unknown>): Promise<Record<string, unknown> | null> {
        const responses = this.mockRegistry.get(routineId);
        if (!responses || responses.length === 0) {
            return null;
        }

        const attemptCount = this.attemptCounters.get(routineId) || 0;
        this.attemptCounters.set(routineId, attemptCount + 1);

        // Find matching response based on attempt number or input
        let matchingResponse = responses.find(r => 
            r.attempt === attemptCount + 1 ||
            (r.input && this.inputMatches(input, r.input)),
        );

        // Fallback to first response if no match
        if (!matchingResponse) {
            matchingResponse = responses[attemptCount % responses.length];
        }

        // Simulate delay if specified
        if (matchingResponse.delay) {
            await new Promise(resolve => setTimeout(resolve, matchingResponse.delay));
        }

        // Throw error if specified
        if (matchingResponse.error) {
            throw matchingResponse.error;
        }

        return matchingResponse.response;
    }

    clear(routineId?: string): void {
        if (routineId) {
            this.mockRegistry.delete(routineId);
            this.attemptCounters.delete(routineId);
        } else {
            this.mockRegistry.clear();
            this.attemptCounters.clear();
        }
    }

    getCallCount(routineId: string): number {
        return this.attemptCounters.get(routineId) || 0;
    }

    getAllCallCounts(): Map<string, number> {
        return new Map(this.attemptCounters);
    }

    private inputMatches(actual: Record<string, unknown> | undefined, expected: Record<string, unknown>): boolean {
        if (!actual && !expected) return true;
        if (!actual || !expected) return false;
        
        // Simple deep equality check
        return JSON.stringify(actual) === JSON.stringify(expected);
    }
}
