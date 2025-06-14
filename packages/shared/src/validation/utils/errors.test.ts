import { describe, it, expect } from "vitest";
import { maxNumErr, maxStrErr, minNumErr, minStrErr, reqErr } from "./errors.js";

describe("Yup-related functions", () => {
    describe("maxNumErr", () => {
        const cases = [
            { max: 10, expected: "Maximum value is 10" },
            { max: 100, expected: "Maximum value is 100" },
            { max: 0, expected: "Maximum value is 0" },
            { max: -5, expected: "Maximum value is -5" },
        ];

        cases.forEach(({ max, expected }) => {
            it(`should return "${expected}" for max value ${max}`, () => {
                expect(maxNumErr({ max })).toBe(expected);
            });
        });
    });

    describe("maxStrErr", () => {
        const cases = [
            { max: 5, value: 123, error: "Value must be a string" },
            { max: 3, value: "test", expected: "1 character over the limit" },
            { max: 1, value: "hello", expected: "4 characters over the limit" },
            // Add more cases as needed
        ];

        cases.forEach(({ max, value, expected, error }) => {
            const testLabel = error ? `throw "${error}"` : `return "${expected}"`;
            it(`should ${testLabel} for value "${value}" with max ${max}`, () => {
                function action() {
                    return maxStrErr({ max, value });
                }
                if (error) {
                    expect(action).toThrow(error);
                } else {
                    expect(action()).toBe(expected);
                }
            });
        });
    });

    describe("minNumErr", () => {
        const cases = [
            { min: 5, expected: "Minimum value is 5" },
            // Add more cases as needed
        ];

        cases.forEach(({ min, expected }) => {
            it(`should return "${expected}" for min value ${min}`, () => {
                expect(minNumErr({ min })).toBe(expected);
            });
        });
    });

    describe("minStrErr", () => {
        const cases = [
            { min: 5, value: 123, error: "Value must be a string" },
            { min: 5, value: "test", expected: "1 character under the limit" },
            { min: 10, value: "hello", expected: "5 characters under the limit" },
            // Add more cases as needed
        ];

        cases.forEach(({ min, value, expected, error }) => {
            const testLabel = error ? `throw "${error}"` : `return "${expected}"`;
            it(`should ${testLabel} for value "${value}" with min ${min}`, () => {
                function action() {
                    return minStrErr({ min, value });
                }
                if (error) {
                    expect(action).toThrow(error);
                } else {
                    expect(action()).toBe(expected);
                }
            });
        });
    });

    describe("reqErr", () => {
        it("should return \"This field is required\"", () => {
            expect(reqErr()).toBe("This field is required");
        });
    });
});
