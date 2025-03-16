import { maxNumErr, maxStrErr, minNumErr, minStrErr, reqErr } from "./errors.js";

describe("Yup-related functions", () => {
    describe("maxNumErr", () => {
        const cases = [
            { max: 10, expected: "Minimum value is 10" },
            // Add more cases as needed
        ];

        cases.forEach(({ max, expected }) => {
            it(`should return "${expected}" for max value ${max}`, () => {
                expect(maxNumErr({ max })).to.equal(expected);
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
                    expect(action()).to.equal(expected);
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
                expect(minNumErr({ min })).to.equal(expected);
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
                    expect(action()).to.equal(expected);
                }
            });
        });
    });

    describe("reqErr", () => {
        it("should return \"This field is required\"", () => {
            expect(reqErr()).to.equal("This field is required");
        });
    });
});
