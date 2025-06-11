import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { noopSubmit, noop, noopAsync, funcTrue, funcFalse, preventFormSubmit } from "./noop.js";

describe("noop utilities", () => {
    describe("noopSubmit", () => {
        let consoleWarnStub: any;
        let setSubmittingStub: any;

        beforeEach(() => {
            consoleWarnStub = vi.spyOn(console, "warn").mockImplementation(() => {});
            setSubmittingStub = vi.fn();
        });

        afterEach(() => {
            consoleWarnStub.mockRestore();
        });

        it("should warn when called and set submitting to false", () => {
            const values = { username: "testuser" };
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit(values, helpers);

            expect(consoleWarnStub).toHaveBeenCalledWith("Formik onSubmit called unexpectedly with values:", values);
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should always set submitting to false to prevent form hanging", () => {
            const helpers = { setSubmitting: setSubmittingStub };

            noopSubmit({}, helpers);
            noopSubmit(null, helpers);
            noopSubmit(undefined, helpers);

            expect(setSubmittingStub).toHaveBeenCalledTimes(3);
            expect(setSubmittingStub).toHaveBeenCalledWith(false);
        });

        it("should propagate errors from setSubmitting", () => {
            const errorMessage = "setSubmitting failed";
            const throwingSetSubmitting = vi.fn().mockImplementation(() => { throw new Error(errorMessage); });
            const helpers = { setSubmitting: throwingSetSubmitting };

            expect(() => noopSubmit({}, helpers)).toThrow(errorMessage);
            expect(consoleWarnStub).toHaveBeenCalled();
        });
    });

    describe("noop", () => {
        let consoleWarnSpy: any;

        beforeEach(() => {
            consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        });

        afterEach(() => {
            consoleWarnSpy.mockRestore();
        });

        it("should warn when called to help debug unexpected calls", () => {
            noop();
            expect(consoleWarnSpy).toHaveBeenCalledWith("Noop called");
        });

        it("should return undefined as expected for a noop function", () => {
            const result = noop();
            expect(result).toBeUndefined();
        });
    });

    describe("noopAsync", () => {
        let consoleWarnSpy: any;

        beforeEach(() => {
            consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        });

        afterEach(() => {
            consoleWarnSpy.mockRestore();
        });

        it("should return a Promise that resolves to undefined", async () => {
            const result = noopAsync();
            expect(result).toBeInstanceOf(Promise);
            expect(await result).toBeUndefined();
        });

        it("should warn when called to help debug unexpected async calls", async () => {
            await noopAsync();
            expect(consoleWarnSpy).toHaveBeenCalledWith("NoopAsync called");
        });
    });

    describe("funcTrue", () => {
        it("should always return true for use as a constant predicate", () => {
            expect(funcTrue()).toBe(true);
            // Useful for testing or as a default callback that always passes
            const items = [1, 2, 3];
            expect(items.filter(() => funcTrue()).length).toBe(items.length);
        });
    });

    describe("funcFalse", () => {
        it("should always return false for use as a constant predicate", () => {
            expect(funcFalse()).toBe(false);
            // Useful for testing or as a default callback that always fails
            const items = [1, 2, 3];
            expect(items.filter(() => funcFalse()).length).toBe(0);
        });
    });

    describe("preventFormSubmit", () => {
        let consoleLogSpy: any;
        let mockEvent: any;

        beforeEach(() => {
            consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => {});
            mockEvent = {
                preventDefault: vi.fn(),
                stopPropagation: vi.fn(),
            };
        });

        afterEach(() => {
            consoleLogSpy.mockRestore();
        });

        it("should prevent form submission and stop event propagation", () => {
            preventFormSubmit(mockEvent);
            
            expect(mockEvent.preventDefault).toHaveBeenCalled();
            expect(mockEvent.stopPropagation).toHaveBeenCalled();
        });

        it("should log for debugging form submission issues", () => {
            preventFormSubmit(mockEvent);
            expect(consoleLogSpy).toHaveBeenCalledWith("in preventFormSubmit");
        });

        it("should work with browser submit events", () => {
            const submitEvent = {
                preventDefault: vi.fn(),
                stopPropagation: vi.fn(),
                type: "submit",
                target: { tagName: "FORM" },
            };

            preventFormSubmit(submitEvent);
            expect(submitEvent.preventDefault).toHaveBeenCalled();
            expect(submitEvent.stopPropagation).toHaveBeenCalled();
        });
    });
});
