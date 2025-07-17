/**
 * Mock Controller
 * 
 * Central controller for all test mocks in scenarios
 */

import type { MockConfiguration, MockAIConfig, MockRoutineConfig } from "./types.js";

export class MockController {
    private mockRegistry = new Map<string, any>();
    private attemptCounters = new Map<string, number>();
    private currentConfig: MockConfiguration | null = null;

    async configure(config: MockConfiguration): Promise<void> {
        this.currentConfig = config;
        
        // Set up AI response mocks
        if (config.ai) {
            for (const [routineId, aiConfig] of Object.entries(config.ai)) {
                this.mockRegistry.set(`ai:${routineId}`, {
                    type: "sequence",
                    responses: aiConfig.responses,
                });
            }
        }

        // Set up routine mocks
        if (config.routines) {
            for (const [routineId, routineConfig] of Object.entries(config.routines)) {
                this.mockRegistry.set(`routine:${routineId}`, {
                    type: "sequence",
                    responses: routineConfig.responses,
                });
            }
        }

        // Set up timing mocks
        if (config.timing) {
            for (const [key, value] of Object.entries(config.timing)) {
                this.mockRegistry.set(`timing:${key}`, value);
            }
        }
    }

    override(key: string, response: any): void {
        this.mockRegistry.set(key, {
            type: "static",
            response,
        });
    }

    async getMockResponse(type: string, id: string, input?: any): Promise<any> {
        const key = `${type}:${id}`;
        const mock = this.mockRegistry.get(key);
        if (!mock) return null;

        const attemptKey = key;
        const attemptCount = this.attemptCounters.get(attemptKey) || 0;
        this.attemptCounters.set(attemptKey, attemptCount + 1);

        if (mock.type === "sequence") {
            const response = mock.responses.find((r: any) => 
                r.attempt === attemptCount + 1 ||
                (r.input && this.inputMatches(input, r.input)),
            ) || mock.responses[attemptCount % mock.responses.length];

            // Simulate delay if specified
            if (response.delay) {
                await new Promise(resolve => setTimeout(resolve, response.delay));
            }

            return response;
        }

        return mock.response;
    }

    getTimingOverride(key: string): number | undefined {
        return this.mockRegistry.get(`timing:${key}`);
    }

    getCallCount(type: string, id: string): number {
        const key = `${type}:${id}`;
        return this.attemptCounters.get(key) || 0;
    }

    clear(): void {
        this.mockRegistry.clear();
        this.attemptCounters.clear();
        this.currentConfig = null;
    }

    getConfig(): MockConfiguration | null {
        return this.currentConfig;
    }

    private inputMatches(actual: any, expected: any): boolean {
        return JSON.stringify(actual) === JSON.stringify(expected);
    }
}
