/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect, describe, it } from "vitest";
import { PasswordAuthService } from "./email.js";
import { randomString } from "./codes.js";

describe("PasswordAuthService", () => {
    describe("generateEmailVerificationCode", () => {
        it("should generate URL-safe codes without slashes", () => {
            const testPublicId = "test-user-123";
            const result = PasswordAuthService.generateEmailVerificationCode(testPublicId);

            // Check that result has the expected structure
            expect(result).toHaveProperty("code");
            expect(result).toHaveProperty("link");
            expect(typeof result.code).toBe("string");
            expect(typeof result.link).toBe("string");
            
            // Check that the code doesn't contain slashes
            expect(result.code).not.toMatch(/\//);
            
            // Check that the link includes the public ID and code
            expect(result.link).toContain(testPublicId);
            expect(result.link).toContain(result.code);
        });

        it("should generate different codes for the same user on multiple invocations", () => {
            const testPublicId = "test-user-123";
            const codes = new Set();
            
            // Generate 10 codes
            for (let i = 0; i < 10; i++) {
                const result = PasswordAuthService.generateEmailVerificationCode(testPublicId);
                codes.add(result.code);
            }
            
            // All codes should be unique
            expect(codes.size).toBe(10);
        });
    });

    describe("randomString", () => {
        it("should generate URL-safe codes without slashes across multiple invocations", () => {
            const iterations = 100;
            const codes = new Array(iterations).fill(null).map(() => randomString(100));

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
                expect(code).not.to.match(/\//);

                // Ensure every character in the code is URL-safe
                Array.from(code).forEach(char => {
                    // @ts-ignore: expect-message
                    expect(isUrlSafeCharacter(char), `Character is ${char}`).to.be.ok;
                });
            });
        });

        const validLengths = [1, 15, 27, 100];
        validLengths.forEach(length => {
            it(`should generate a code of exactly ${length} characters`, () => {
                const code = randomString(length);
                expect(code.length).toBe(length);
            });
        });

        const invalidLengths = [-1, 0, "", NaN, Infinity, 99999, {}, []];
        invalidLengths.forEach(length => {
            it(`should throw an error when length is ${JSON.stringify(length)}`, () => {
                // @ts-ignore: Testing runtime scenario
                expect(() => randomString(length)).toThrow();
            });
        });
    });
});
