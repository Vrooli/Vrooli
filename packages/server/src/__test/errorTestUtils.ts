/**
 * Error Testing Utilities
 * 
 * Provides standardized functions for testing CustomError and external errors
 * in the Vrooli test suite.
 */

import { expect } from "vitest";
import { CustomError } from "../events/error.js";
import type { TranslationKeyError } from "@vrooli/shared";

/**
 * Asserts that an error is a CustomError with the expected code and optional trace prefix
 * 
 * @param error - The error to check
 * @param code - The expected error code (TranslationKeyError)
 * @param tracePrefix - Optional trace prefix to match (e.g., "0323")
 * 
 * @example
 * try {
 *     await someFunction();
 *     expect.fail("Should have thrown");
 * } catch (error) {
 *     expectCustomError(error, "Unauthorized", "0323");
 * }
 */
export function expectCustomError(
    error: unknown,
    code: TranslationKeyError,
    tracePrefix?: string,
): void {
    expect(error).toBeInstanceOf(CustomError);
    const customError = error as CustomError;
    expect(customError.code).toBe(code);
    
    if (tracePrefix) {
        expect(customError.trace).toMatch(new RegExp(`^${tracePrefix}-`));
    }
    
    // Ensure the error has the expected structure
    expect(customError).toHaveProperty("trace");
    expect(customError).toHaveProperty("code");
    expect(customError).toHaveProperty("message");
}

/**
 * Asserts that an async function rejects with a CustomError
 * 
 * @param promise - The promise to check
 * @param code - The expected error code
 * @param tracePrefix - Optional trace prefix to match
 * 
 * @example
 * await expectCustomErrorAsync(
 *     auth.emailLogIn({ input }, context),
 *     "InvalidCredentials",
 *     "0062"
 * );
 */
export async function expectCustomErrorAsync(
    promise: Promise<any>,
    code: TranslationKeyError,
    tracePrefix?: string,
): Promise<void> {
    try {
        await promise;
        expect.fail(`Expected CustomError with code "${code}" but no error was thrown`);
    } catch (error) {
        expectCustomError(error, code, tracePrefix);
    }
}

/**
 * Asserts that a function throws a CustomError
 * 
 * @param fn - The function to execute
 * @param code - The expected error code
 * @param tracePrefix - Optional trace prefix to match
 * 
 * @example
 * expectCustomErrorSync(
 *     () => validateInput(invalidData),
 *     "InvalidArgs",
 *     "0321"
 * );
 */
export function expectCustomErrorSync(
    fn: () => any,
    code: TranslationKeyError,
    tracePrefix?: string,
): void {
    try {
        fn();
        expect.fail(`Expected CustomError with code "${code}" but no error was thrown`);
    } catch (error) {
        expectCustomError(error, code, tracePrefix);
    }
}

/**
 * Asserts that an error is an external (non-CustomError) error
 * Use this for errors from external libraries like Stripe, JWT, etc.
 * 
 * @param error - The error to check
 * @param messagePattern - String or RegExp to match against error message
 * @param errorType - Optional error constructor to check instanceof
 * 
 * @example
 * try {
 *     await stripeOperation();
 * } catch (error) {
 *     expectExternalError(error, /Invalid price ID/, StripeError);
 * }
 */
export function expectExternalError(
    error: unknown,
    messagePattern: string | RegExp,
    errorType?: new (...args: any[]) => Error,
): void {
    expect(error).toBeInstanceOf(Error);
    
    if (errorType) {
        expect(error).toBeInstanceOf(errorType);
    }
    
    const errorObj = error as Error;
    if (typeof messagePattern === "string") {
        expect(errorObj.message).toContain(messagePattern);
    } else {
        expect(errorObj.message).toMatch(messagePattern);
    }
}

/**
 * Asserts that an async function rejects with an external error
 * 
 * @param promise - The promise to check
 * @param messagePattern - String or RegExp to match against error message
 * @param errorType - Optional error constructor to check instanceof
 * 
 * @example
 * await expectExternalErrorAsync(
 *     jwt.sign(invalidPayload),
 *     "jwt.sign failed"
 * );
 */
export async function expectExternalErrorAsync(
    promise: Promise<any>,
    messagePattern: string | RegExp,
    errorType?: new (...args: any[]) => Error,
): Promise<void> {
    try {
        await promise;
        expect.fail(`Expected error matching "${messagePattern}" but no error was thrown`);
    } catch (error) {
        expectExternalError(error, messagePattern, errorType);
    }
}

/**
 * Asserts that a function does not throw any error
 * 
 * @param fn - The function to execute
 * 
 * @example
 * expectNoError(() => validateInput(validData));
 */
export function expectNoError(fn: () => any): void {
    expect(fn).not.toThrow();
}

/**
 * Asserts that an async function does not reject
 * 
 * @param promise - The promise to check
 * 
 * @example
 * await expectNoErrorAsync(processValidData(data));
 */
export async function expectNoErrorAsync(promise: Promise<any>): Promise<void> {
    await expect(promise).resolves.not.toThrow();
}

/**
 * Helper to check if an error has specific CustomError data
 * 
 * @param error - The CustomError to check
 * @param data - Expected data object
 * 
 * @example
 * expectCustomErrorData(error, { userId: "123", action: "delete" });
 */
export function expectCustomErrorData(
    error: CustomError,
    data: Record<string, any>,
): void {
    expect(error.data).toBeDefined();
    expect(error.data).toMatchObject(data);
}

/**
 * Helper to convert legacy string error checks to CustomError checks
 * This is a migration helper - use sparingly and replace with proper checks
 * 
 * @param fn - Function that might throw
 * @param legacyMessage - The legacy message being checked
 * @returns The caught error for further inspection
 * 
 * @example
 * const error = catchForMigration(() => someFunction(), "Invalid input");
 * // Now inspect error to determine proper CustomError code
 */
export function catchForMigration(fn: () => any, legacyMessage?: string): Error {
    try {
        fn();
        expect.fail(`Expected error${legacyMessage ? ` with message "${legacyMessage}"` : ""} but none was thrown`);
        throw new Error("Unreachable");
    } catch (error) {
        if (error instanceof Error && error.message.includes("Expected error")) {
            throw error;
        }
        return error as Error;
    }
}

/**
 * Helper for async version of catchForMigration
 */
export async function catchForMigrationAsync(
    promise: Promise<any>,
    legacyMessage?: string,
): Promise<Error> {
    try {
        await promise;
        expect.fail(`Expected error${legacyMessage ? ` with message "${legacyMessage}"` : ""} but none was thrown`);
        throw new Error("Unreachable");
    } catch (error) {
        if (error instanceof Error && error.message.includes("Expected error")) {
            throw error;
        }
        return error as Error;
    }
}
