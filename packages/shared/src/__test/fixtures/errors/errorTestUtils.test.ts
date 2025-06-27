/**
 * Test file for error fixture validation utilities
 * 
 * This tests the new VrooliError interface integration and validates
 * that error fixtures work with ServerResponseParser.
 */

import { describe, expect, it } from "vitest";
import { 
    runErrorFixtureValidationTests, 
    validateTranslationKeys,
    runComprehensiveErrorTests,
    validateErrorFixtureServerConversion, 
} from "./errorTestUtils.js";
import { authErrorFixtures } from "./authErrors.js";
import { BaseErrorFixture } from "./types.js";
import { standardErrorValidator } from "../../../index.js";

describe("Error Test Utils", () => {
    describe("VrooliError Interface Validation", () => {
        it("should validate BaseErrorFixture implements VrooliError interface", () => {
            const fixture = new BaseErrorFixture("TestError", "0001-TEST");
            
            expect(standardErrorValidator.matchesPattern(fixture)).toBe(true);
            expect(standardErrorValidator.isCompatibleWithParser(fixture)).toBe(true);
            expect(fixture.isParseableByUI()).toBe(true);
        });

        it("should convert fixture to ServerError correctly", () => {
            const fixture = new BaseErrorFixture("TestError", "0001-TEST", { test: true });
            
            // Fixture → ServerError
            const serverError = fixture.toServerError();
            expect(serverError).toEqual({
                trace: "0001-TEST",
                code: "TestError",
            });

            // Validate ServerError has required properties
            expect(serverError.trace).toBe("0001-TEST");
            expect("code" in serverError && serverError.code).toBe("TestError");
        });

        it("should provide valid severity levels", () => {
            const warningFixture = new BaseErrorFixture("WarningTest", "0002-TEST");
            const infoFixture = new BaseErrorFixture("InfoNotice", "0003-TEST");
            const errorFixture = new BaseErrorFixture("ErrorTest", "0004-TEST");

            expect(warningFixture.getSeverity()).toBe("Warning");
            expect(infoFixture.getSeverity()).toBe("Info");
            expect(errorFixture.getSeverity()).toBe("Error");
        });
    });

    describe("Auth Error Fixtures Validation", () => {
        runErrorFixtureValidationTests(authErrorFixtures, "Auth Error Fixtures");
        
        it("should have all required auth error fixtures", () => {
            expect(authErrorFixtures.invalidCredentials).toBeDefined();
            expect(authErrorFixtures.sessionExpired).toBeDefined();
            expect(authErrorFixtures.accessDenied).toBeDefined();
            expect(authErrorFixtures.accountLocked).toBeDefined();
            expect(authErrorFixtures.twoFactorRequired).toBeDefined();
        });

        validateErrorFixtureServerConversion(authErrorFixtures, "Auth Error Fixtures");
    });

    describe("Translation Key Validation", () => {
        const mockTranslationKeys = [
            "InvalidCredentials",
            "SessionExpired", 
            "AccessDenied",
            "AccountLocked",
            "TwoFactorRequired",
        ];

        validateTranslationKeys(authErrorFixtures, mockTranslationKeys, "Auth Error Fixtures");
    });

    describe("Comprehensive Error Tests", () => {
        const mockTranslationKeys = [
            "InvalidCredentials",
            "SessionExpired", 
            "AccessDenied",
            "AccountLocked",
            "TwoFactorRequired",
        ];

        runComprehensiveErrorTests(
            authErrorFixtures, 
            "Auth Error Fixtures - Comprehensive",
            mockTranslationKeys,
        );
    });
});
