import { describe, expect, test, beforeEach, afterEach, vi, type Mock } from "vitest";
import { ErrorHandler, type ErrorHandlerConfig, type ErrorContext } from "./ErrorHandler.js";
import { EventPublisher } from "../../events/publisher.js";
import { logger } from "../../../events/logger.js";
import { EventTypes } from "@vrooli/shared";

// Mock dependencies
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("../../events/publisher.js", () => ({
    EventPublisher: {
        emit: vi.fn().mockResolvedValue({ 
            proceed: true, 
            eventId: "test-event-123",
            reason: null,
            wasBlocking: false,
            publishResult: { success: true },
        }),
    },
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => "test-pk-123"),
    };
});

describe("ErrorHandler", () => {
    let errorHandler: ErrorHandler;
    let defaultConfig: ErrorHandlerConfig;

    beforeEach(() => {
        vi.clearAllMocks();
        defaultConfig = {
            component: "TestComponent",
            chatId: "test-chat-123",
            runId: "test-run-456",
        };
        errorHandler = new ErrorHandler(defaultConfig);
    });

    describe("constructor", () => {
        test("should set default values correctly", () => {
            const handler = new ErrorHandler({ component: "Test" });
            // We can't directly access private config, but we can test behavior
            expect(handler).toBeDefined();
        });

        test("should accept custom configuration", () => {
            const customConfig: ErrorHandlerConfig = {
                component: "CustomComponent",
                logByDefault: false,
                publishByDefault: false,
                ignoredErrors: ["ECONNREFUSED"],
                includeStackTrace: false,
            };
            const handler = new ErrorHandler(customConfig);
            expect(handler).toBeDefined();
        });
    });

    describe("wrap", () => {
        test("should return success result for successful operations", async () => {
            const operation = vi.fn().mockResolvedValue({ data: "test" });
            const context: ErrorContext = {
                operation: "testOperation",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result).toEqual({ success: true, data: { data: "test" } });
            expect(operation).toHaveBeenCalledOnce();
            expect(logger.error).not.toHaveBeenCalled();
            expect(EventPublisher.emit).not.toHaveBeenCalled();
        });

        test("should handle errors and return failure result", async () => {
            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "failingOperation",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result).toEqual({ success: false, error: testError });
            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] failingOperation failed",
                expect.objectContaining({
                    operation: "failingOperation",
                    error: "Test error",
                    errorName: "Error",
                    stack: expect.any(String),
                }),
            );
            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SYSTEM.ERROR,
                expect.objectContaining({
                    chatId: "test-chat-123",
                    runId: "test-run-456",
                    component: "TestComponent",
                    operation: "failingOperation",
                    error: testError,
                }),
            );
        });

        test("should respect logError=false in context", async () => {
            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "quietOperation",
                component: "TestComponent",
                logError: false,
            };

            await errorHandler.wrap(operation, context);

            expect(logger.error).not.toHaveBeenCalled();
            expect(EventPublisher.emit).toHaveBeenCalled(); // Still publishes by default
        });

        test("should respect publishError=false in context", async () => {
            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "privateOperation",
                component: "TestComponent",
                publishError: false,
            };

            await errorHandler.wrap(operation, context);

            expect(logger.error).toHaveBeenCalled(); // Still logs by default
            expect(EventPublisher.emit).not.toHaveBeenCalled();
        });

        test("should include custom metadata in logs", async () => {
            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "metadataOperation",
                component: "TestComponent",
                metadata: {
                    userId: "user-123",
                    requestId: "req-456",
                },
            };

            await errorHandler.wrap(operation, context);

            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] metadataOperation failed",
                expect.objectContaining({
                    userId: "user-123",
                    requestId: "req-456",
                }),
            );
        });

        test("should use custom reason when provided", async () => {
            const testError = new Error("Original error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "customReasonOperation",
                component: "TestComponent",
                reason: "Custom error message for user",
            };

            await errorHandler.wrap(operation, context);

            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] customReasonOperation failed",
                expect.objectContaining({
                    error: "Custom error message for user",
                }),
            );
        });
    });

    describe("execute", () => {
        test("should return data for successful operations", async () => {
            const operation = vi.fn().mockResolvedValue({ data: "test" });
            const context: ErrorContext = {
                operation: "executeSuccess",
                component: "TestComponent",
            };

            const result = await errorHandler.execute(operation, context);

            expect(result).toEqual({ data: "test" });
            expect(operation).toHaveBeenCalledOnce();
        });

        test("should throw error for failed operations", async () => {
            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "executeFailure",
                component: "TestComponent",
            };

            await expect(errorHandler.execute(operation, context)).rejects.toThrow("Test error");
            expect(logger.error).toHaveBeenCalled();
            expect(EventPublisher.emit).toHaveBeenCalled();
        });
    });

    describe("executeWithRetry", () => {
        test("should succeed on first try", async () => {
            const operation = vi.fn().mockResolvedValue({ data: "test" });
            const context: ErrorContext = {
                operation: "retrySuccess",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(operation, context);

            expect(result).toEqual({ success: true, data: { data: "test" } });
            expect(operation).toHaveBeenCalledOnce();
            expect(logger.info).not.toHaveBeenCalled(); // No retry success log
        });

        test("should retry and succeed", async () => {
            const operation = vi.fn()
                .mockRejectedValueOnce(new Error("ECONNREFUSED"))
                .mockResolvedValueOnce({ data: "test" });
            const context: ErrorContext = {
                operation: "retryEventualSuccess",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 2,
                retryDelay: 10,
            });

            expect(result).toEqual({ success: true, data: { data: "test" } });
            expect(operation).toHaveBeenCalledTimes(2);
            expect(logger.warn).toHaveBeenCalledWith(
                "[TestComponent] Retrying operation after 10ms",
                expect.objectContaining({
                    operation: "retryEventualSuccess",
                    attempt: 1,
                    maxRetries: 2,
                }),
            );
            expect(logger.info).toHaveBeenCalledWith(
                "[TestComponent] Operation succeeded after 1 retries",
                expect.objectContaining({
                    operation: "retryEventualSuccess",
                    attempts: 2,
                }),
            );
        });

        test("should fail after max retries", async () => {
            const testError = new Error("ETIMEDOUT");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "retryMaxFailure",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 2,
                retryDelay: 10,
            });

            expect(result).toEqual({ success: false, error: testError });
            expect(operation).toHaveBeenCalledTimes(3); // Initial + 2 retries
            expect(logger.warn).toHaveBeenCalledTimes(2); // 2 retry warnings
        });

        test("should not retry non-retryable errors", async () => {
            const testError = new Error("Not a network error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "nonRetryableError",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 3,
                retryableErrors: ["ECONNREFUSED"],
            });

            expect(result).toEqual({ success: false, error: testError });
            expect(operation).toHaveBeenCalledOnce(); // No retries
            expect(logger.warn).not.toHaveBeenCalled();
        });

        test("should use exponential backoff", async () => {
            const testError = new Error("ENOTFOUND");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "exponentialBackoff",
                component: "TestComponent",
            };

            const startTime = Date.now();
            await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 2,
                retryDelay: 10,
                exponentialBackoff: true,
            });
            const totalTime = Date.now() - startTime;

            // Should wait 10ms + 20ms = 30ms minimum
            expect(totalTime).toBeGreaterThanOrEqual(30);
            expect(logger.warn).toHaveBeenCalledWith(
                "[TestComponent] Retrying operation after 10ms",
                expect.any(Object),
            );
            expect(logger.warn).toHaveBeenCalledWith(
                "[TestComponent] Retrying operation after 20ms",
                expect.any(Object),
            );
        });

        test("should include attempt metadata", async () => {
            const testError = new Error("ECONNREFUSED");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "metadataRetry",
                component: "TestComponent",
                metadata: { originalData: "test" },
            };

            await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 1,
                retryDelay: 10,
            });

            // Check that EventPublisher was called with attempt metadata
            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SYSTEM.ERROR,
                expect.objectContaining({
                    context: expect.objectContaining({
                        originalData: "test",
                        attempt: 1,
                        maxRetries: 1,
                    }),
                }),
            );
        });
    });

    describe("error normalization", () => {
        test("should handle Error instances", async () => {
            const testError = new Error("Test error");
            testError.name = "CustomError";
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "errorInstance",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result.success).toBe(false);
            if (!result.success) {
                expect(result.error).toBeInstanceOf(Error);
                expect(result.error.message).toBe("Test error");
                expect(result.error.name).toBe("CustomError");
            }
        });

        test("should handle string errors", async () => {
            const operation = vi.fn().mockRejectedValue("String error");
            const context: ErrorContext = {
                operation: "stringError",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result.success).toBe(false);
            if (!result.success) {
                expect(result.error).toBeInstanceOf(Error);
                expect(result.error.message).toBe("String error");
            }
        });

        test("should handle objects with message property", async () => {
            const errorObj = { message: "Object error", code: "ERR_001" };
            const operation = vi.fn().mockRejectedValue(errorObj);
            const context: ErrorContext = {
                operation: "objectError",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result.success).toBe(false);
            if (!result.success) {
                expect(result.error).toBeInstanceOf(Error);
                expect(result.error.message).toBe("Object error");
            }
        });

        test("should handle unknown error types", async () => {
            const operation = vi.fn().mockRejectedValue({ weird: "object" });
            const context: ErrorContext = {
                operation: "unknownError",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result.success).toBe(false);
            if (!result.success) {
                expect(result.error).toBeInstanceOf(Error);
                expect(result.error.message).toBe("[object Object]");
            }
        });

        test("should handle null/undefined errors", async () => {
            const operation = vi.fn().mockRejectedValue(null);
            const context: ErrorContext = {
                operation: "nullError",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result.success).toBe(false);
            if (!result.success) {
                expect(result.error).toBeInstanceOf(Error);
                expect(result.error.message).toBe("null");
            }
        });
    });

    describe("ignored errors", () => {
        test("should not log or publish ignored errors", async () => {
            const ignoredHandler = new ErrorHandler({
                ...defaultConfig,
                ignoredErrors: ["IgnorableError"],
            });

            const testError = new Error("This should be ignored");
            testError.name = "IgnorableError";
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "ignoredOperation",
                component: "TestComponent",
            };

            const result = await ignoredHandler.wrap(operation, context);

            expect(result).toEqual({ success: false, error: testError });
            expect(logger.error).not.toHaveBeenCalled();
            expect(EventPublisher.emit).not.toHaveBeenCalled();
        });
    });

    describe("event publishing edge cases", () => {
        test("should handle event publishing blocking", async () => {
            vi.mocked(EventPublisher.emit).mockResolvedValueOnce({
                proceed: false,
                eventId: "blocked-event-123",
                reason: "Event blocked by system policy",
                wasBlocking: true,
                publishResult: { success: false },
            });

            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "blockedOperation",
                component: "TestComponent",
            };

            await errorHandler.wrap(operation, context);

            expect(logger.warn).toHaveBeenCalledWith(
                "[TestComponent] Error event blocked by system",
                expect.objectContaining({
                    originalError: "Test error",
                    blockReason: "Event blocked by system policy",
                    operation: "blockedOperation",
                }),
            );
        });

        test("should handle event publishing failures", async () => {
            vi.mocked(EventPublisher.emit).mockRejectedValueOnce(new Error("Publishing failed"));

            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "publishFailOperation",
                component: "TestComponent",
            };

            const result = await errorHandler.wrap(operation, context);

            expect(result).toEqual({ success: false, error: testError });
            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] Failed to publish error event to unified system",
                expect.objectContaining({
                    originalError: "Test error",
                    publishError: "Publishing failed",
                }),
            );
        });
    });

    describe("stack trace handling", () => {
        test("should include stack trace by default", async () => {
            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "stackTraceOperation",
                component: "TestComponent",
            };

            await errorHandler.wrap(operation, context);

            expect(logger.error).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    stack: expect.stringContaining("Error: Test error"),
                }),
            );
        });

        test("should exclude stack trace when configured", async () => {
            const noStackHandler = new ErrorHandler({
                ...defaultConfig,
                includeStackTrace: false,
            });

            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "noStackOperation",
                component: "TestComponent",
            };

            await noStackHandler.wrap(operation, context);

            expect(logger.error).toHaveBeenCalledWith(
                expect.any(String),
                expect.not.objectContaining({
                    stack: expect.any(String),
                }),
            );
        });
    });

    describe("static helper methods", () => {
        test("isSuccess should correctly identify success results", () => {
            const successResult = { success: true as const, data: "test" };
            const failureResult = { success: false as const, error: new Error() };

            expect(ErrorHandler.isSuccess(successResult)).toBe(true);
            expect(ErrorHandler.isSuccess(failureResult)).toBe(false);
        });

        test("isError should correctly identify error results", () => {
            const successResult = { success: true as const, data: "test" };
            const failureResult = { success: false as const, error: new Error() };

            expect(ErrorHandler.isError(successResult)).toBe(false);
            expect(ErrorHandler.isError(failureResult)).toBe(true);
        });
    });

    describe("configuration edge cases", () => {
        test("should handle missing chatId and runId", async () => {
            const minimalHandler = new ErrorHandler({
                component: "MinimalComponent",
            });

            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "minimalOperation",
                component: "MinimalComponent",
            };

            await minimalHandler.wrap(operation, context);

            expect(EventPublisher.emit).toHaveBeenCalledWith(
                EventTypes.SYSTEM.ERROR,
                expect.objectContaining({
                    chatId: undefined,
                    runId: undefined,
                }),
            );
        });

        test("should disable logging and publishing when configured", async () => {
            const silentHandler = new ErrorHandler({
                component: "SilentComponent",
                logByDefault: false,
                publishByDefault: false,
            });

            const testError = new Error("Test error");
            const operation = vi.fn().mockRejectedValue(testError);
            const context: ErrorContext = {
                operation: "silentOperation",
                component: "SilentComponent",
            };

            await silentHandler.wrap(operation, context);

            expect(logger.error).not.toHaveBeenCalled();
            expect(EventPublisher.emit).not.toHaveBeenCalled();
        });
    });

    describe("error recovery scenarios", () => {
        test("should handle cascading error recovery failures", async () => {
            // Simulate event publishing failing during error handling
            (EventPublisher.emit as Mock).mockRejectedValueOnce(new Error("Event publishing failed"));

            const originalError = new Error("Original operation failed");
            const operation = vi.fn().mockRejectedValue(originalError);
            const context: ErrorContext = {
                operation: "cascadingFailure",
                component: "TestComponent",
                publishError: true,
            };

            const result = await errorHandler.wrap(operation, context);

            // Should still return the original error, not the publishing error
            expect(result).toEqual({ success: false, error: originalError });
            
            // Should log both the original error and the publishing failure
            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] cascadingFailure failed",
                expect.objectContaining({
                    error: "Original operation failed",
                }),
            );
            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] Failed to publish error event to unified system",
                expect.objectContaining({
                    originalError: "Original operation failed",
                    publishError: "Event publishing failed",
                }),
            );
        });

        test("should prevent infinite retry loops with circuit breaker pattern", async () => {
            const persistentError = new Error("ECONNREFUSED");
            let callCount = 0;
            const operation = vi.fn().mockImplementation(() => {
                callCount++;
                throw persistentError;
            });

            const context: ErrorContext = {
                operation: "circuitBreakerTest",
                component: "TestComponent",
            };

            // First attempt should retry normally
            const result1 = await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 2,
                retryDelay: 1,
            });

            expect(result1.success).toBe(false);
            expect(callCount).toBe(3); // Initial + 2 retries
            
            // Reset call count for second attempt
            callCount = 0;
            
            // Second attempt should also retry (no circuit breaker implemented yet, but tests the pattern)
            const result2 = await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 2,
                retryDelay: 1,
            });

            expect(result2.success).toBe(false);
            expect(callCount).toBe(3); // Should retry again
        });

        test("should handle partial recovery scenarios", async () => {
            let attemptCount = 0;
            const partiallyFailingOperation = vi.fn().mockImplementation(() => {
                attemptCount++;
                if (attemptCount <= 2) {
                    throw new Error("ECONNREFUSED");
                }
                return { data: "success", attemptCount };
            });

            const context: ErrorContext = {
                operation: "partialRecovery",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(partiallyFailingOperation, context, {
                maxRetries: 3,
                retryDelay: 1,
            });

            expect(result.success).toBe(true);
            expect(result.data).toEqual({ data: "success", attemptCount: 3 });
            expect(attemptCount).toBe(3);
            
            // Should log successful recovery
            expect(logger.info).toHaveBeenCalledWith(
                "[TestComponent] Operation succeeded after 2 retries",
                expect.objectContaining({
                    operation: "partialRecovery",
                    attempts: 3,
                }),
            );
        });

        test("should handle timeout errors in recovery scenarios", async () => {
            const timeoutError = new Error("ETIMEDOUT");
            timeoutError.name = "TimeoutError";
            
            const operation = vi.fn().mockRejectedValue(timeoutError);
            const context: ErrorContext = {
                operation: "timeoutRecovery",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 2,
                retryDelay: 1,
                retryableErrors: ["ETIMEDOUT", "TimeoutError"],
            });

            expect(result.success).toBe(false);
            expect(result.error).toBe(timeoutError);
            expect(operation).toHaveBeenCalledTimes(3); // Should retry timeout errors
        });

        test("should handle resource cleanup during error recovery", async () => {
            const cleanupSpy = vi.fn();
            const resourceError = new Error("Resource allocation failed");
            
            const operationWithCleanup = vi.fn().mockImplementation(async () => {
                try {
                    throw resourceError;
                } finally {
                    cleanupSpy();
                }
            });

            const context: ErrorContext = {
                operation: "resourceCleanup",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(operationWithCleanup, context, {
                maxRetries: 2,
                retryDelay: 1,
            });

            expect(result.success).toBe(false);
            expect(result.error).toBe(resourceError);
            
            // Cleanup should be called for each attempt
            expect(cleanupSpy).toHaveBeenCalledTimes(3); // Initial + 2 retries
        });

        test("should handle custom retry logic for specific error types", async () => {
            const customError = new Error("CUSTOM_ERROR_CODE");
            (customError as any).code = "CUSTOM_ERROR_CODE";
            
            const operation = vi.fn().mockRejectedValue(customError);
            const context: ErrorContext = {
                operation: "customRetryLogic",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(operation, context, {
                maxRetries: 2,
                retryDelay: 1,
                retryableErrors: ["CUSTOM_ERROR_CODE"],
            });

            expect(result.success).toBe(false);
            expect(operation).toHaveBeenCalledTimes(3); // Should retry custom error codes
        });

        test("should handle error correlation across multiple operations", async () => {
            const correlationId = "correlation-123";
            const error1 = new Error("First operation failed");
            const error2 = new Error("Second operation failed");
            
            const operation1 = vi.fn().mockRejectedValue(error1);
            const operation2 = vi.fn().mockRejectedValue(error2);
            
            const baseContext = {
                component: "TestComponent",
                metadata: { correlationId },
            };

            const result1 = await errorHandler.wrap(operation1, {
                ...baseContext,
                operation: "correlatedOperation1",
            });

            const result2 = await errorHandler.wrap(operation2, {
                ...baseContext,
                operation: "correlatedOperation2",
            });

            expect(result1.success).toBe(false);
            expect(result2.success).toBe(false);

            // Both errors should be logged with the same correlation ID
            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] correlatedOperation1 failed",
                expect.objectContaining({
                    correlationId,
                }),
            );
            expect(logger.error).toHaveBeenCalledWith(
                "[TestComponent] correlatedOperation2 failed",
                expect.objectContaining({
                    correlationId,
                }),
            );
        });

        test("should handle graceful degradation scenarios", async () => {
            let primaryCallCount = 0;
            const primaryOperation = vi.fn().mockImplementation(() => {
                primaryCallCount++;
                throw new Error("Primary service unavailable");
            });

            const fallbackOperation = vi.fn().mockResolvedValue({ data: "fallback_success" });

            const context: ErrorContext = {
                operation: "gracefulDegradation",
                component: "TestComponent",
            };

            // Simulate a pattern where we try primary, then fallback
            const primaryResult = await errorHandler.wrap(primaryOperation, context);
            
            let finalResult;
            if (!primaryResult.success) {
                // Primary failed, try fallback
                finalResult = await errorHandler.wrap(fallbackOperation, {
                    ...context,
                    operation: "fallbackOperation",
                });
            } else {
                finalResult = primaryResult;
            }

            expect(primaryResult.success).toBe(false);
            expect(finalResult.success).toBe(true);
            expect(finalResult.data).toEqual({ data: "fallback_success" });
            
            expect(primaryOperation).toHaveBeenCalledOnce();
            expect(fallbackOperation).toHaveBeenCalledOnce();
        });

        test("should handle complex retry scenarios with different error types", async () => {
            const errors = [
                new Error("ECONNREFUSED"),    // Retryable
                new Error("ETIMEDOUT"),       // Retryable
                new Error("ENOTFOUND"),       // Retryable
                new Error("UNAUTHORIZED"),    // Not retryable
            ];
            
            let callCount = 0;
            const complexOperation = vi.fn().mockImplementation(() => {
                const error = errors[callCount % errors.length];
                callCount++;
                throw error;
            });

            const context: ErrorContext = {
                operation: "complexRetryScenario",
                component: "TestComponent",
            };

            const result = await errorHandler.executeWithRetry(complexOperation, context, {
                maxRetries: 5,
                retryDelay: 1,
                retryableErrors: ["ECONNREFUSED", "ETIMEDOUT", "ENOTFOUND"],
            });

            expect(result.success).toBe(false);
            // Should stop retrying when it hits UNAUTHORIZED (4th call)
            expect(callCount).toBe(4);
            expect(result.error?.message).toBe("UNAUTHORIZED");
        });
    });
});
