import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { noopSubmit, noop, noopAsync, funcTrue, funcFalse, preventFormSubmit } from "./noop.js";

describe("noop utilities", () => {
    describe("noopSubmit", () => {
        let consoleWarnStub: any;
        let setSubmittingStub: any;

        beforeEach(() => {
            // Spy on console.warn to prevent output during tests
            consoleWarnStub = vi.spyOn(console, "warn").mockImplementation(() => {});
            setSubmittingStub = vi.fn();
        });

        afterEach(() => {
            // Restore the original console.warn
            consoleWarnStub.mockRestore();
        });

        it("should call console.warn with the provided values", () => {
            const values = { username: "testuser", email: "test@example.com" };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledOnce();
            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", values);
        });

        it("should call setSubmitting with false", () => {
            const values = { field: "value" };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(setSubmittingStub).toHaveBeenCalledOnce();
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should handle null values", () => {
            const values = null;
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", null);
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should handle undefined values", () => {
            const values = undefined;
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", undefined);
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should handle empty object values", () => {
            const values = {};
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", {});
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should handle complex nested values", () => {
            const values = {
                user: {
                    name: "John Doe",
                    preferences: {
                        theme: "dark",
                        notifications: true,
                    },
                },
                items: [1, 2, 3],
            };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", values);
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should handle array values", () => {
            const values = [1, 2, 3, 4, 5];
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", values);
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should handle string values", () => {
            const values = "unexpected string submission";
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", values);
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should work when setSubmitting throws an error", () => {
            const values = { test: "value" };
            const errorMessage = "setSubmitting failed";
            const throwingSetSubmitting = vi.fn().mockImplementation(() => { throw new Error(errorMessage); });
            const helpers = { setSubmitting: throwingSetSubmitting };

            expect(() => noopSubmit(values, helpers)).toThrow(errorMessage);
            expect(consoleWarnStub).toHaveBeenCalledOnce();
        });

        it("should be called multiple times without issues", () => {
            const values1 = { call: 1 };
            const values2 = { call: 2 };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values1, helpers);
            noopSubmit(values2, helpers);

            expect(consoleWarnStub).toHaveBeenCalledTimes(2);
            expect(setSubmittingStub).toHaveBeenCalledTimes(2);
            expect(consoleWarnStub).toHaveBeenNthCalledWith(1, "Formik onSubmit called unexpectedly with values:", values1);
            expect(consoleWarnStub).toHaveBeenNthCalledWith(2, "Formik onSubmit called unexpectedly with values:", values2);
        });
    });

    describe("noop", () => {
        let consoleWarnSpy: any;

        beforeEach(() => {
            consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {
                // Mock implementation
            });
        });

        afterEach(() => {
            consoleWarnSpy.mockRestore();
        });

        it("should call console.warn with 'Noop called'", () => {
            noop();
            expect(consoleWarnSpy).toHaveBeenCalledWith("Noop called");
            expect(consoleWarnSpy).toHaveBeenCalledTimes(1);
        });

        it("should be callable multiple times", () => {
            noop();
            noop();
            noop();
            expect(consoleWarnSpy).toHaveBeenCalledTimes(3);
            expect(consoleWarnSpy).toHaveBeenCalledWith("Noop called");
        });

        it("should return undefined", () => {
            const result = noop();
            expect(result).toBeUndefined();
        });
    });

    describe("noopAsync", () => {
        let consoleWarnSpy: any;

        beforeEach(() => {
            consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {
                // Mock implementation
            });
        });

        afterEach(() => {
            consoleWarnSpy.mockRestore();
        });

        it("should be an async function", () => {
            const result = noopAsync();
            expect(result).toBeInstanceOf(Promise);
        });

        it("should call console.warn with 'NoopAsync called'", async () => {
            await noopAsync();
            expect(consoleWarnSpy).toHaveBeenCalledWith("NoopAsync called");
            expect(consoleWarnSpy).toHaveBeenCalledTimes(1);
        });

        it("should resolve to undefined", async () => {
            const result = await noopAsync();
            expect(result).toBeUndefined();
        });

        it("should be callable multiple times", async () => {
            await noopAsync();
            await noopAsync();
            expect(consoleWarnSpy).toHaveBeenCalledTimes(2);
            expect(consoleWarnSpy).toHaveBeenCalledWith("NoopAsync called");
        });
    });

    describe("funcTrue", () => {
        it("should always return true", () => {
            expect(funcTrue()).toBe(true);
            expect(funcTrue()).toBe(true);
            expect(funcTrue()).toBe(true);
        });

        it("should return boolean type", () => {
            const result = funcTrue();
            expect(typeof result).toBe("boolean");
        });

        it("should be idempotent", () => {
            const results = Array.from({ length: 100 }, () => funcTrue());
            expect(results.every(result => result === true)).toBe(true);
        });
    });

    describe("funcFalse", () => {
        it("should always return false", () => {
            expect(funcFalse()).toBe(false);
            expect(funcFalse()).toBe(false);
            expect(funcFalse()).toBe(false);
        });

        it("should return boolean type", () => {
            const result = funcFalse();
            expect(typeof result).toBe("boolean");
        });

        it("should be idempotent", () => {
            const results = Array.from({ length: 100 }, () => funcFalse());
            expect(results.every(result => result === false)).toBe(true);
        });
    });

    describe("preventFormSubmit", () => {
        let consoleLogSpy: any;
        let mockEvent: any;

        beforeEach(() => {
            consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => {
                // Mock implementation
            });
            mockEvent = {
                preventDefault: vi.fn(),
                stopPropagation: vi.fn(),
            };
        });

        afterEach(() => {
            consoleLogSpy.mockRestore();
        });

        it("should call console.log with 'in preventFormSubmit'", () => {
            preventFormSubmit(mockEvent);
            expect(consoleLogSpy).toHaveBeenCalledWith("in preventFormSubmit");
            expect(consoleLogSpy).toHaveBeenCalledTimes(1);
        });

        it("should call preventDefault on the event", () => {
            preventFormSubmit(mockEvent);
            expect(mockEvent.preventDefault).toHaveBeenCalledTimes(1);
        });

        it("should call stopPropagation on the event", () => {
            preventFormSubmit(mockEvent);
            expect(mockEvent.stopPropagation).toHaveBeenCalledTimes(1);
        });

        it("should call both preventDefault and stopPropagation", () => {
            preventFormSubmit(mockEvent);
            expect(mockEvent.preventDefault).toHaveBeenCalledTimes(1);
            expect(mockEvent.stopPropagation).toHaveBeenCalledTimes(1);
        });

        it("should handle multiple calls", () => {
            preventFormSubmit(mockEvent);
            preventFormSubmit(mockEvent);
            expect(mockEvent.preventDefault).toHaveBeenCalledTimes(2);
            expect(mockEvent.stopPropagation).toHaveBeenCalledTimes(2);
            expect(consoleLogSpy).toHaveBeenCalledTimes(2);
        });

        it("should work with real DOM event-like objects", () => {
            const realEvent = {
                preventDefault: vi.fn(),
                stopPropagation: vi.fn(),
                type: "submit",
                target: {},
            };

            preventFormSubmit(realEvent);
            expect(realEvent.preventDefault).toHaveBeenCalledTimes(1);
            expect(realEvent.stopPropagation).toHaveBeenCalledTimes(1);
        });

        it("should return undefined", () => {
            const result = preventFormSubmit(mockEvent);
            expect(result).toBeUndefined();
        });
    });
});
