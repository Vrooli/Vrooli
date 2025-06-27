/**
 * Error Fixture Test Utilities
 * 
 * Provides validation and testing utilities for error fixtures,
 * ensuring they match actual application error handling patterns.
 * 
 * Similar to validationTestUtils.ts but specifically designed for
 * error fixtures and their integration with the VrooliError interface.
 */

import { describe, expect, it } from "vitest";
import { standardErrorValidator } from "../../../index.js";
import { type TranslationKeyError } from "../../../types.js";

// Note: ClientError is now in UI package, so tests using it should be in UI package

/**
 * Interface for error fixtures that can be validated
 */
export interface ErrorFixture {
    code: string;
    trace: string;
    data?: Record<string, unknown>;
    toServerError?: () => { trace: string; code?: string; message?: string; data?: Record<string, unknown> };
    getSeverity?: () => "Error" | "Warning" | "Info";
}

/**
 * Type for error fixture collections
 */
export type ErrorFixtureCollection = Record<string, ErrorFixture>;

/**
 * Configuration options for error fixture validation tests.
 */
export interface ErrorFixtureValidationOptions {
    /** Validate trace format (XXXX-XXXX pattern) */
    validateTrace?: boolean;
    /** Validate translation keys exist and are valid */
    validateTranslationKeys?: boolean;
    /** Validate compatibility with ServerResponseParser */
    validateParserCompatibility?: boolean;
    /** Skip ServerError conversion tests */
    skipServerErrorConversion?: boolean;
    /** Custom validation functions */
    customValidations?: Array<(error: ErrorFixture) => void>;
}

/**
 * Standard validation tests for error fixtures.
 * Similar to runStandardValidationTests but for error fixtures.
 * 
 * @param fixtures Error fixtures object to validate
 * @param categoryName Name of the error category for test descriptions
 * @param options Validation options
 */
export function runErrorFixtureValidationTests<TFixtures extends ErrorFixtureCollection>(
    fixtures: TFixtures,
    categoryName: string,
    options: ErrorFixtureValidationOptions = {},
): void {
    const {
        validateTrace = true,
        validateTranslationKeys = true,
        validateParserCompatibility = true,
        skipServerErrorConversion = false,
        customValidations = [],
    } = options;

    describe(`${categoryName} Error Fixtures - Validation Tests`, () => {
        
        it("should have valid error structure for all fixtures", () => {
            Object.entries(fixtures).forEach(([key, fixture]) => {
                expect(fixture, `Fixture '${key}' should exist`).toBeDefined();
                expect(fixture, `Fixture '${key}' should have code property`).toHaveProperty("code");
                expect(fixture, `Fixture '${key}' should have trace property`).toHaveProperty("trace");
                
                expect(typeof fixture.code, `Fixture '${key}' code should be string`).toBe("string");
                expect(typeof fixture.trace, `Fixture '${key}' trace should be string`).toBe("string");
            });
        });

        if (validateTrace) {
            it("should have valid trace format for all fixtures", () => {
                Object.entries(fixtures).forEach(([key, fixture]) => {
                    const isValid = standardErrorValidator.isValidTrace(fixture.trace);
                    expect(isValid, `Fixture '${key}' should have valid trace format (XXXX-XXXX): ${fixture.trace}`).toBe(true);
                });
            });
        }

        if (validateTranslationKeys) {
            it("should have valid translation keys", () => {
                Object.entries(fixtures).forEach(([key, fixture]) => {
                    const isValid = standardErrorValidator.isValidErrorCode(fixture.code);
                    expect(isValid, `Fixture '${key}' should have valid error code: ${fixture.code}`).toBe(true);
                });
            });
        }

        it("should be compatible with VrooliError interface", () => {
            Object.entries(fixtures).forEach(([key, fixture]) => {
                const matchesPattern = standardErrorValidator.matchesPattern(fixture);
                expect(matchesPattern, `Fixture '${key}' should match VrooliError interface pattern`).toBe(true);
            });
        });

        if (!skipServerErrorConversion) {
            it("should convert to ServerError format correctly", () => {
                Object.entries(fixtures).forEach(([key, fixture]) => {
                    expect(typeof fixture.toServerError, `Fixture '${key}' should have toServerError method`).toBe("function");
                    
                    if (fixture.toServerError) {
                        const serverError = fixture.toServerError();
                        expect(serverError, `Fixture '${key}' should produce valid ServerError`).toBeDefined();
                        expect(serverError, `Fixture '${key}' ServerError should have trace`).toHaveProperty("trace");
                        expect(serverError.trace, `Fixture '${key}' ServerError trace should match`).toBe(fixture.trace);
                        
                        // Should have either code or message
                        const hasCode = "code" in serverError;
                        const hasMessage = "message" in serverError;
                        expect(hasCode || hasMessage, `Fixture '${key}' ServerError should have either code or message`).toBe(true);
                    }
                });
            });
        }

        if (validateParserCompatibility) {
            it("should be compatible with ServerResponseParser", () => {
                Object.entries(fixtures).forEach(([key, fixture]) => {
                    // Test that fixture can be converted to ServerError (UI tests will validate ClientError conversion)
                    if (fixture.toServerError) {
                        const serverError = fixture.toServerError();
                        
                        // Validate ServerError format matches expected parser patterns
                        expect(serverError, `Fixture '${key}' should produce valid ServerError`).toBeDefined();
                        expect(serverError, `Fixture '${key}' ServerError should have trace`).toHaveProperty("trace");
                        
                        // Should have either code or message (what ServerResponseParser expects)
                        const hasCode = "code" in serverError;
                        const hasMessage = "message" in serverError;
                        expect(hasCode || hasMessage, `Fixture '${key}' ServerError should have either code or message for parser compatibility`).toBe(true);
                    }
                });
            });

            it("should provide valid severity levels", () => {
                Object.entries(fixtures).forEach(([key, fixture]) => {
                    if ("getSeverity" in fixture && typeof fixture.getSeverity === "function") {
                        const severity = fixture.getSeverity();
                        expect(["Error", "Warning", "Info"], 
                               `Fixture '${key}' should have valid severity`).toContain(severity);
                    }
                });
            });
        }

        // Run custom validations
        if (customValidations.length > 0) {
            it("should pass custom validations", () => {
                Object.entries(fixtures).forEach(([key, fixture]) => {
                    customValidations.forEach((validation, index) => {
                        expect(() => validation(fixture), 
                               `Fixture '${key}' should pass custom validation ${index + 1}`).not.toThrow();
                    });
                });
            });
        }
    });
}

/**
 * Validate fixtures against actual translation keys.
 * Ensures all error codes used in fixtures exist in translation files.
 * 
 * @param fixtures Error fixtures to validate
 * @param translationKeys Array of valid translation keys
 * @param categoryName Category name for test descriptions
 */
export function validateTranslationKeys(
    fixtures: ErrorFixtureCollection, 
    translationKeys: TranslationKeyError[],
    categoryName = "Error Fixtures",
): void {
    describe(`${categoryName} - Translation Key Validation`, () => {
        it("should use valid translation keys", () => {
            Object.entries(fixtures).forEach(([key, fixture]) => {
                expect(translationKeys, 
                       `Fixture '${key}' should use valid translation key: ${fixture.code}`)
                    .toContain(fixture.code);
            });
        });
    });
}

/**
 * Comprehensive validation tests combining all validations.
 * Similar to runComprehensiveValidationTests but for error fixtures.
 * 
 * @param fixtures Error fixtures to validate
 * @param categoryName Category name for test descriptions
 * @param translationKeys Optional array of valid translation keys
 * @param options Additional validation options
 */
export function runComprehensiveErrorTests<TFixtures extends ErrorFixtureCollection>(
    fixtures: TFixtures,
    categoryName: string,
    translationKeys?: TranslationKeyError[],
    options: ErrorFixtureValidationOptions = {},
): void {
    // Run standard error fixture validation
    runErrorFixtureValidationTests(fixtures, categoryName, options);
    
    // Run translation key validation if keys provided
    if (translationKeys) {
        validateTranslationKeys(fixtures, translationKeys, categoryName);
    }
}

/**
 * Test individual error fixture properties and behavior.
 * Useful for testing specific error scenarios.
 * 
 * @param fixture Single error fixture to test
 * @param expectedProperties Expected properties the fixture should have
 * @param testName Optional test name
 */
export function testErrorFixture(
    fixture: ErrorFixture,
    expectedProperties: {
        code?: TranslationKeyError;
        trace?: string;
        data?: Record<string, unknown>;
        severity?: "Error" | "Warning" | "Info";
    },
    testName = "Error fixture",
): void {
    describe(testName, () => {
        it("should have expected structure", () => {
            expect(standardErrorValidator.matchesPattern(fixture)).toBe(true);
        });

        if (expectedProperties.code) {
            it("should have expected error code", () => {
                expect(fixture.code).toBe(expectedProperties.code);
            });
        }

        if (expectedProperties.trace) {
            it("should have expected trace", () => {
                expect(fixture.trace).toBe(expectedProperties.trace);
            });
        }

        if (expectedProperties.data) {
            it("should have expected data", () => {
                expect(fixture.data).toEqual(expectedProperties.data);
            });
        }

        if (expectedProperties.severity && "getSeverity" in fixture) {
            it("should have expected severity", () => {
                expect(fixture.getSeverity()).toBe(expectedProperties.severity);
            });
        }
    });
}

/**
 * Validate error fixture ServerError conversion consistency.
 * Tests that fixtures can be consistently converted to ServerError format.
 * 
 * Note: Full round-trip testing (including ClientError) should be done in UI package tests.
 * 
 * @param fixtures Error fixtures to test
 * @param categoryName Category name for test descriptions
 */
export function validateErrorFixtureServerConversion<TFixtures extends ErrorFixtureCollection>(
    fixtures: TFixtures,
    categoryName: string,
): void {
    describe(`${categoryName} - ServerError Conversion`, () => {
        it("should consistently convert to ServerError format", () => {
            Object.entries(fixtures).forEach(([key, fixture]) => {
                if (fixture.toServerError) {
                    // Test multiple conversions produce identical results
                    const serverError1 = fixture.toServerError();
                    const serverError2 = fixture.toServerError();
                    
                    // Validate consistency
                    expect(serverError2.trace, `Fixture '${key}' trace should be consistent`).toBe(serverError1.trace);
                    
                    if ("code" in serverError1) {
                        expect("code" in serverError2, `Fixture '${key}' should consistently have code`).toBe(true);
                        if ("code" in serverError2) {
                            expect(serverError2.code, `Fixture '${key}' code should be consistent`).toBe(serverError1.code);
                        }
                    }
                    
                    if ("message" in serverError1) {
                        expect("message" in serverError2, `Fixture '${key}' should consistently have message`).toBe(true);
                        if ("message" in serverError2) {
                            expect(serverError2.message, `Fixture '${key}' message should be consistent`).toBe(serverError1.message);
                        }
                    }
                }
            });
        });
    });
}
