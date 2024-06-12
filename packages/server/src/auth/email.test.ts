/* eslint-disable @typescript-eslint/ban-ts-comment */
import { generateEmailVerificationCode } from "./email";

describe("generateEmailVerificationCode function", () => {
    it("should generate URL-safe codes without slashes across multiple invocations", () => {
        const iterations = 100; // Number of times to invoke generateEmailVerificationCode
        const codes = new Array(iterations).fill(null).map(() => generateEmailVerificationCode(100));

        // Define the allowed character ranges and characters for URL safety
        const isUrlSafeCharacter = (char) => {
            const code = char.charCodeAt(0);
            return (code >= 48 && code <= 57)  // 0-9
                || (code >= 65 && code <= 90)  // A-Z
                || (code >= 97 && code <= 122) // a-z
                || ["-", "_", ".", "~", "$"].includes(char);
        };

        codes.forEach(code => {
            // Ensure the code doesn't contain slashes
            expect(code).not.toMatch(/\//);

            // Ensure every character in the code is URL-safe
            Array.from(code).forEach(char => {
                // @ts-ignore: expect-message
                expect(isUrlSafeCharacter(char), `Character is ${char}`).toBeTruthy();
            });
        });
    });

    const validLengths = [1, 15, 27, 100];
    validLengths.forEach(length => {
        it(`should generate a code of exactly ${length} characters`, () => {
            const code = generateEmailVerificationCode(length);
            expect(code.length).toBe(length);
        });
    });

    const invalidLengths = [-1, 0, "", NaN, Infinity, 99999, null, {}, []];
    invalidLengths.forEach(length => {
        it(`should throw an error when length is ${JSON.stringify(length)}`, () => {
            // @ts-ignore: Testing runtime scenario
            expect(() => generateEmailVerificationCode(length)).toThrow();
        });
    });
});
