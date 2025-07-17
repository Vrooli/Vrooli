/**
 * Scenario Validator
 * 
 * Validates scenario outcomes against expectations
 */

import type { ScenarioContext, ScenarioResult } from "./types.js";

export interface ValidationResult {
    valid: boolean;
    errors: ValidationError[];
    warnings: ValidationWarning[];
}

export interface ValidationError {
    type: string;
    message: string;
    expected?: any;
    actual?: any;
}

export interface ValidationWarning {
    type: string;
    message: string;
}

export class ScenarioValidator {
    validate(context: ScenarioContext, result: ScenarioResult): ValidationResult {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];

        // Validate blackboard state
        this.validateBlackboard(context, result, errors);

        // Validate event sequence
        this.validateEventSequence(context, result, errors);

        // Validate routine calls
        this.validateRoutineCalls(context, result, errors);

        // Validate duration
        this.validateDuration(context, result, warnings);

        // Validate outcomes
        this.validateOutcomes(context, result, errors);

        return {
            valid: errors.length === 0,
            errors,
            warnings,
        };
    }

    private validateBlackboard(
        context: ScenarioContext, 
        result: ScenarioResult,
        errors: ValidationError[],
    ): void {
        if (!context.expectations.finalBlackboard) return;

        for (const [key, expectedValue] of Object.entries(context.expectations.finalBlackboard)) {
            const actualValue = result.blackboard[key];
            
            if (!this.deepEqual(actualValue, expectedValue)) {
                errors.push({
                    type: "blackboard_mismatch",
                    message: `Blackboard key "${key}" has unexpected value`,
                    expected: expectedValue,
                    actual: actualValue,
                });
            }
        }
    }

    private validateEventSequence(
        context: ScenarioContext,
        result: ScenarioResult,
        errors: ValidationError[],
    ): void {
        if (!context.expectations.eventSequence) return;

        const expectedSequence = context.expectations.eventSequence;
        const actualSequence = result.events.map(e => e.topic);

        // Check if all expected events occurred
        for (const expectedEvent of expectedSequence) {
            if (!actualSequence.includes(expectedEvent)) {
                errors.push({
                    type: "missing_event",
                    message: `Expected event "${expectedEvent}" did not occur`,
                    expected: expectedEvent,
                });
            }
        }

        // Check sequence order
        let sequenceIndex = 0;
        for (const actualEvent of actualSequence) {
            if (actualEvent === expectedSequence[sequenceIndex]) {
                sequenceIndex++;
                if (sequenceIndex === expectedSequence.length) break;
            }
        }

        if (sequenceIndex !== expectedSequence.length) {
            errors.push({
                type: "event_sequence_mismatch",
                message: "Events did not occur in expected order",
                expected: expectedSequence,
                actual: actualSequence,
            });
        }
    }

    private validateRoutineCalls(
        context: ScenarioContext,
        result: ScenarioResult,
        errors: ValidationError[],
    ): void {
        if (!context.expectations.routineCalls) return;

        for (const expected of context.expectations.routineCalls) {
            const actualCalls = result.routineCalls.filter(
                call => call.routineLabel === expected.routine,
            );

            if (actualCalls.length !== expected.times) {
                errors.push({
                    type: "routine_call_count_mismatch",
                    message: `Routine "${expected.routine}" called ${actualCalls.length} times, expected ${expected.times}`,
                    expected: expected.times,
                    actual: actualCalls.length,
                });
            }
        }
    }

    private validateDuration(
        context: ScenarioContext,
        result: ScenarioResult,
        warnings: ValidationWarning[],
    ): void {
        if (!context.expectations.duration) return;

        const { min, max } = context.expectations.duration;

        if (min && result.duration < min) {
            warnings.push({
                type: "duration_too_short",
                message: `Scenario completed in ${result.duration}ms, faster than expected minimum ${min}ms`,
            });
        }

        if (max && result.duration > max) {
            warnings.push({
                type: "duration_too_long",
                message: `Scenario took ${result.duration}ms, longer than expected maximum ${max}ms`,
            });
        }
    }

    private validateOutcomes(
        context: ScenarioContext,
        result: ScenarioResult,
        errors: ValidationError[],
    ): void {
        if (!context.expectations.outcomes) return;

        // Check if final status matches expected outcomes
        if (result.finalStatus && !context.expectations.outcomes.includes(result.finalStatus)) {
            errors.push({
                type: "unexpected_outcome",
                message: `Scenario ended with status "${result.finalStatus}", which is not in expected outcomes`,
                expected: context.expectations.outcomes,
                actual: result.finalStatus,
            });
        }
    }

    private deepEqual(a: any, b: any): boolean {
        if (a === b) return true;
        
        if (typeof a !== typeof b) return false;
        
        if (a === null || b === null) return false;
        
        if (typeof a !== "object") return false;
        
        if (Array.isArray(a) !== Array.isArray(b)) return false;
        
        if (Array.isArray(a)) {
            if (a.length !== b.length) return false;
            return a.every((item, index) => this.deepEqual(item, b[index]));
        }
        
        const keysA = Object.keys(a);
        const keysB = Object.keys(b);
        
        if (keysA.length !== keysB.length) return false;
        
        return keysA.every(key => this.deepEqual(a[key], b[key]));
    }
}
