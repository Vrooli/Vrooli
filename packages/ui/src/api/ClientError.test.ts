/**
 * Tests for ClientError class and its integration with ServerResponseParser
 * 
 * This tests the UI-specific error handling implementation and validates
 * round-trip conversion between server and client error formats.
 */

import { type ServerError, type TranslationKeyError } from "@vrooli/shared";
import { describe, expect, it } from "vitest";
import { ClientError } from "./ClientError.js";
import { ServerResponseParser } from "./responseParser.js";

describe("ClientError", () => {
    describe("Construction and Basic Properties", () => {
        it("should create ClientError with all properties", () => {
            const error = new ClientError("TestError" as TranslationKeyError, "0001-ABCD", { test: true });

            expect(error.code).toBe("TestError");
            expect(error.trace).toBe("0001-ABCD");
            expect(error.data).toEqual({ test: true });
            expect(error.message).toBe("TestError: 0001-ABCD");
            expect(error.name).toBe("TestError");
        });

        it("should implement ParseableError interface", () => {
            const error = new ClientError("TestError" as TranslationKeyError, "0001-ABCD");

            expect(error.isParseableByUI()).toBe(true);
            expect(error.toServerError()).toBeDefined();
            expect(error.getSeverity()).toBeDefined();
            expect(error.getUserMessage()).toBeDefined();
        });
    });

    describe("ServerError Conversion", () => {
        it("should create ClientError from translated ServerError", () => {
            const serverError: ServerError = {
                code: "InvalidCredentials" as TranslationKeyError,
                trace: "0062-ABCD",
            };

            const clientError = ClientError.fromServerError(serverError);

            expect(clientError.code).toBe("InvalidCredentials");
            expect(clientError.trace).toBe("0062-ABCD");
            expect(clientError.isUntranslated()).toBe(false);
        });

        it("should create ClientError from untranslated ServerError", () => {
            const serverError: ServerError = {
                message: "Custom error message",
                trace: "0063-ABCD",
            };

            const clientError = ClientError.fromServerError(serverError);

            expect(clientError.code).toBe("ErrorUnknown");
            expect(clientError.trace).toBe("0063-ABCD");
            expect(clientError.isUntranslated()).toBe(true);
            expect(clientError.getOriginalMessage()).toBe("Custom error message");
        });

        it("should convert back to ServerError correctly", () => {
            const originalServerError: ServerError = {
                code: "SessionExpired" as TranslationKeyError,
                trace: "0064-ABCD",
            };

            const clientError = ClientError.fromServerError(originalServerError);
            const convertedServerError = clientError.toServerError();

            expect(convertedServerError).toEqual(originalServerError);
        });

        it("should preserve untranslated message in round-trip", () => {
            const originalServerError: ServerError = {
                message: "Database connection failed",
                trace: "0065-ABCD",
            };

            const clientError = ClientError.fromServerError(originalServerError);
            const convertedServerError = clientError.toServerError();

            expect("message" in convertedServerError && convertedServerError.message).toBe("Database connection failed");
            expect(convertedServerError.trace).toBe("0065-ABCD");
        });
    });

    describe("Severity Detection", () => {
        it("should detect Warning severity", () => {
            const warningError = new ClientError("WarningLowDiskSpace" as TranslationKeyError, "0001-ABCD");
            expect(warningError.getSeverity()).toBe("Warning");
        });

        it("should detect Info severity", () => {
            const infoError = new ClientError("InfoMaintenanceScheduled" as TranslationKeyError, "0002-ABCD");
            expect(infoError.getSeverity()).toBe("Info");
        });

        it("should default to Error severity", () => {
            const error = new ClientError("CriticalSystemFailure" as TranslationKeyError, "0003-ABCD");
            expect(error.getSeverity()).toBe("Error");
        });
    });

    describe("User Messages", () => {
        it("should return original message for untranslated errors", () => {
            const serverError: ServerError = {
                message: "Network timeout occurred",
                trace: "0066-ABCD",
            };

            const clientError = ClientError.fromServerError(serverError);
            expect(clientError.getUserMessage()).toBe("Network timeout occurred");
        });

        it("should return fallback message for translated errors", () => {
            const clientError = new ClientError("SessionExpired" as TranslationKeyError, "0067-ABCD");
            expect(clientError.getUserMessage()).toBe("Error: SessionExpired");
        });
    });

    describe("Integration with ServerResponseParser", () => {
        it("should work with ServerResponseParser error conversion", () => {
            const serverError: ServerError = {
                code: "InvalidCredentials" as TranslationKeyError,
                trace: "0068-ABCD",
            };

            const clientError = ServerResponseParser.serverErrorToClientError(serverError);

            expect(clientError).toBeInstanceOf(ClientError);
            expect(clientError.code).toBe("InvalidCredentials");
            expect(clientError.trace).toBe("0068-ABCD");
        });

        it("should work with ServerResponseParser processErrors", () => {
            const response = {
                errors: [
                    { code: "InvalidCredentials" as TranslationKeyError, trace: "0069-ABCD" },
                    { message: "Database error", trace: "0070-ABCD" },
                ],
            };

            const { clientErrors, messages, codes } = ServerResponseParser.processErrors(response, ["en"]);

            expect(clientErrors).toHaveLength(2);
            expect(clientErrors[0]).toBeInstanceOf(ClientError);
            expect(clientErrors[1]).toBeInstanceOf(ClientError);

            expect(codes[0]).toBe("InvalidCredentials");
            expect(codes[1]).toBe("ErrorUnknown");

            expect(messages).toHaveLength(2);
        });

        it("should handle multiple ServerErrors conversion", () => {
            const serverErrors: ServerError[] = [
                { code: "SessionExpired" as TranslationKeyError, trace: "0071-ABCD" },
                { code: "AccessDenied" as TranslationKeyError, trace: "0072-ABCD" },
            ];

            const clientErrors = ClientError.fromServerErrors(serverErrors);

            expect(clientErrors).toHaveLength(2);
            expect(clientErrors[0].code).toBe("SessionExpired");
            expect(clientErrors[1].code).toBe("AccessDenied");
        });
    });

    describe("Validation Methods", () => {
        it("should validate parser compatibility", () => {
            const clientError = new ClientError("TestError" as TranslationKeyError, "0075-ABCD");
            expect(clientError.isCompatibleWithParser()).toBe(true);
        });

        it("should validate VrooliError pattern", () => {
            const clientError = new ClientError("TestError" as TranslationKeyError, "0076-ABCD");
            expect(clientError.matchesVrooliErrorPattern()).toBe(true);
        });

        it("should handle validation errors gracefully", () => {
            const clientError = new ClientError("TestError" as TranslationKeyError, "0077-ABCD");
            // Mock a toServerError that throws
            clientError.toServerError = () => { throw new Error("Mock error"); };

            expect(clientError.isCompatibleWithParser()).toBe(false);
        });
    });

    describe("Error Handling Edge Cases", () => {
        it("should throw error for invalid ServerError format", () => {
            const invalidServerError = { trace: "0073-ABCD" } as any;

            expect(() => ClientError.fromServerError(invalidServerError)).toThrow(
                "Invalid ServerError format - must have either 'code' or 'message'",
            );
        });

        it("should handle empty data gracefully", () => {
            const clientError = new ClientError("TestError" as TranslationKeyError, "0074-ABCD");

            expect(clientError.data).toBeUndefined();
            expect(clientError.isUntranslated()).toBe(false);
            expect(clientError.getOriginalMessage()).toBeUndefined();
        });
    });
});
