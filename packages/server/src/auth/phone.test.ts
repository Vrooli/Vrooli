/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect, describe, it } from "vitest";
import { generatePhoneVerificationCode } from "./phone.js";

describe("generatePhoneVerificationCode function", () => {
    it("should generate codes with digits 0-9 only", () => {
        const iterations = 100; // Number of times to invoke generatePhoneVerificationCode
        const codes = new Array(iterations).fill(null).map(() => generatePhoneVerificationCode(100));

        const isValidCharacter = (char) => {
            const code = char.charCodeAt(0);
            return (code >= 48 && code <= 57); // 0-9
        };

        codes.forEach(code => {
            // Ensure every character in the code is a digit
            Array.from(code).forEach(char => {
                // @ts-ignore: expect-message
                expect(isValidCharacter(char), `Character is ${char}`).to.be.ok;
            });
        });
    });

    const validLengths = [1, 15, 27, 100];
    validLengths.forEach(length => {
        it(`should generate a code of exactly ${length} characters`, () => {
            const code = generatePhoneVerificationCode(length);
            expect(code.length).toBe(length);
        });
    });

    const invalidLengths = [-1, 0, "", NaN, Infinity, 99999, null, {}, []];
    invalidLengths.forEach(length => {
        it(`should throw an error when length is ${JSON.stringify(length)}`, () => {
            // @ts-ignore: Testing runtime scenario
            expect(() => generatePhoneVerificationCode(length)).toThrow();
        });
    });
});
