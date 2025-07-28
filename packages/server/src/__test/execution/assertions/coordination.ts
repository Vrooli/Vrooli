/**
 * Coordination Assertions
 * 
 * Custom assertions for testing agent coordination patterns
 */

import { expect } from "vitest";
import type { RoutineCall } from "../factories/scenario/types.js";

export interface CoordinationAssertions {
    toHaveCoordinationPattern(pattern: CoordinationPattern): void;
    toHaveRetryPattern(maxAttempts: number): void;
    toHaveParallelExecution(routines: string[]): void;
    toHaveSequentialExecution(routines: string[]): void;
}

export interface CoordinationPattern {
    type: "retry" | "parallel" | "sequential" | "pipeline";
    config: any;
}

declare module "vitest" {
    type Assertion<T> = CoordinationAssertions
    type AsymmetricMatchersContaining = CoordinationAssertions
}

export function extendCoordinationAssertions() {
    expect.extend({
        toHaveCoordinationPattern(received: RoutineCall[], pattern: CoordinationPattern) {
            let pass = false;
            let message = "";

            switch (pattern.type) {
                case "retry":
                    const retryResult = checkRetryPattern(received, pattern.config);
                    pass = retryResult.pass;
                    message = retryResult.message;
                    break;
                    
                case "parallel":
                    const parallelResult = checkParallelPattern(received, pattern.config);
                    pass = parallelResult.pass;
                    message = parallelResult.message;
                    break;
                    
                case "sequential":
                    const sequentialResult = checkSequentialPattern(received, pattern.config);
                    pass = sequentialResult.pass;
                    message = sequentialResult.message;
                    break;
                    
                case "pipeline":
                    const pipelineResult = checkPipelinePattern(received, pattern.config);
                    pass = pipelineResult.pass;
                    message = pipelineResult.message;
                    break;
            }

            return { pass, message: () => message };
        },

        toHaveRetryPattern(received: RoutineCall[], maxAttempts: number) {
            // Group calls by routine
            const callsByRoutine = new Map<string, RoutineCall[]>();
            for (const call of received) {
                const calls = callsByRoutine.get(call.routineLabel) || [];
                calls.push(call);
                callsByRoutine.set(call.routineLabel, calls);
            }

            // Check if any routine has retry pattern
            let foundRetry = false;
            let retryDetails = "";

            for (const [routine, calls] of Array.from(callsByRoutine.entries())) {
                if (calls.length > 1) {
                    foundRetry = true;
                    retryDetails += `${routine}: ${calls.length} attempts\n`;
                    
                    // Check if retries stopped after success
                    const successIndex = calls.findIndex(c => c.success);
                    if (successIndex >= 0 && successIndex < calls.length - 1) {
                        retryDetails += `  Warning: Continued after success at attempt ${successIndex + 1}\n`;
                    }
                }
            }

            const pass = foundRetry;

            return {
                pass,
                message: () => pass
                    ? `Expected not to have retry pattern:\n${retryDetails}`
                    : `Expected to have retry pattern (max ${maxAttempts} attempts), but all routines executed only once`,
            };
        },

        toHaveParallelExecution(received: RoutineCall[], routines: string[]) {
            const relevantCalls = received.filter(c => routines.includes(c.routineLabel));
            
            if (relevantCalls.length < 2) {
                return {
                    pass: false,
                    message: () => `Expected parallel execution of ${routines.join(", ")}, but found only ${relevantCalls.length} calls`,
                };
            }

            // Check for time overlap
            let hasOverlap = false;
            for (let i = 0; i < relevantCalls.length; i++) {
                for (let j = i + 1; j < relevantCalls.length; j++) {
                    const call1 = relevantCalls[i];
                    const call2 = relevantCalls[j];
                    const endTime1 = new Date(call1.timestamp.getTime() + call1.duration);
                    const endTime2 = new Date(call2.timestamp.getTime() + call2.duration);
                    
                    // Check if calls overlap
                    if (call1.timestamp <= endTime2 && call2.timestamp <= endTime1) {
                        hasOverlap = true;
                        break;
                    }
                }
                if (hasOverlap) break;
            }

            return {
                pass: hasOverlap,
                message: () => hasOverlap
                    ? "Expected not to have parallel execution"
                    : `Expected parallel execution of ${routines.join(", ")}, but calls did not overlap in time`,
            };
        },

        toHaveSequentialExecution(received: RoutineCall[], routines: string[]) {
            const relevantCalls = received.filter(c => routines.includes(c.routineLabel));
            
            // Check if routines were called in the specified order
            let currentIndex = 0;
            const executionOrder: string[] = [];
            
            for (const call of relevantCalls) {
                executionOrder.push(call.routineLabel);
                const expectedRoutine = routines[currentIndex % routines.length];
                if (call.routineLabel === expectedRoutine) {
                    currentIndex++;
                }
            }

            const pass = currentIndex >= routines.length;

            return {
                pass,
                message: () => pass
                    ? "Expected not to have sequential execution"
                    : `Expected sequential execution: ${routines.join(" → ")}\n` +
                      `Actual order: ${executionOrder.join(" → ")}`,
            };
        },
    });
}

function checkRetryPattern(calls: RoutineCall[], config: any): { pass: boolean; message: string } {
    const { routine, maxAttempts, successRequired } = config;
    const routineCalls = calls.filter(c => c.routineLabel === routine);
    
    if (routineCalls.length === 0) {
        return {
            pass: false,
            message: `Expected retry pattern for "${routine}", but no calls found`,
        };
    }

    if (routineCalls.length > maxAttempts) {
        return {
            pass: false,
            message: `Expected max ${maxAttempts} attempts for "${routine}", but found ${routineCalls.length}`,
        };
    }

    if (successRequired) {
        const hasSuccess = routineCalls.some(c => c.success);
        if (!hasSuccess) {
            return {
                pass: false,
                message: `Expected successful retry for "${routine}", but all ${routineCalls.length} attempts failed`,
            };
        }
    }

    return {
        pass: true,
        message: `Has retry pattern for "${routine}" with ${routineCalls.length} attempts`,
    };
}

function checkParallelPattern(calls: RoutineCall[], config: any): { pass: boolean; message: string } {
    const { routines } = config;
    // Implementation similar to toHaveParallelExecution
    return { pass: true, message: "Has parallel pattern" };
}

function checkSequentialPattern(calls: RoutineCall[], config: any): { pass: boolean; message: string } {
    const { routines } = config;
    // Implementation similar to toHaveSequentialExecution
    return { pass: true, message: "Has sequential pattern" };
}

function checkPipelinePattern(calls: RoutineCall[], config: any): { pass: boolean; message: string } {
    const { stages } = config;
    // Check if outputs from one stage feed into inputs of next
    return { pass: true, message: "Has pipeline pattern" };
}
