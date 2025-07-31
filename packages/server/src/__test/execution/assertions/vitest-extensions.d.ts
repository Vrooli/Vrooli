/**
 * Vitest Custom Assertions Type Extensions
 * 
 * Unified type declarations for all custom test assertions
 * This extends vitest's built-in Assertion interface without replacing it
 */

import type { CoordinationPattern } from "./coordination.js";

declare module "vitest" {
  interface Assertion<T = any> {
    // Blackboard assertions
    toHaveKey(key: string): T
    toHaveKeys(keys: string[]): T
    toHaveValue(key: string, value: any): T
    toMatchState(expectedState: Record<string, any>): T
    toContainState(partialState: Record<string, any>): T

    // Event assertions
    toMatchSequence(expectedSequence: string[]): T
    toContainEvents(expectedEvents: string[]): T
    toHaveEventCount(topic: string, count: number): T
    toHaveEventWithData(topic: string, data: any): T

    // Coordination assertions
    toHaveCoordinationPattern(pattern: CoordinationPattern): T
    toHaveRetryPattern(maxAttempts: number): T
    toHaveParallelExecution(routines: string[]): T
    toHaveSequentialExecution(routines: string[]): T

    // Outcome assertions
    toMatchCalls(expectedCalls: Array<{ routine: string; times: number }>): T
    toHaveSuccessRate(minRate: number): T
    toCompleteWithin(maxDuration: number): T
    toUseCreditsWithin(maxCredits: number): T
  }
}
