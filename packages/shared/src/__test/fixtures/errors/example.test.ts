/**
 * Example tests demonstrating how to use error fixtures
 * 
 * This file shows practical examples of using error fixtures
 * in different testing scenarios.
 */

import { describe, expect, it } from "vitest";
import {
    apiErrorFixtures,
    authErrorFixtures,
    businessErrorFixtures,
    networkErrorFixtures,
    systemErrorFixtures,
    validationErrorFixtures,
} from "./index.js";

describe("Error Fixture Examples", () => {
    describe("API Error Handling", () => {
        it("should handle 400 Bad Request errors", () => {
            const error = apiErrorFixtures.badRequest.withDetails;
            
            // Simulate error response
            expect(error.status).toBe(400);
            expect(error.code).toBe("BAD_REQUEST");
            expect(error.details.fields).toHaveProperty("email");
            expect(error.details.fields).toHaveProperty("password");
        });

        it("should handle rate limiting", () => {
            const error = apiErrorFixtures.rateLimit.standard;
            
            expect(error.status).toBe(429);
            expect(error.retryAfter).toBe(60);
            expect(error.remaining).toBe(0);
            expect(new Date(error.reset!)).toBeInstanceOf(Date);
        });

        it("should create custom validation errors", () => {
            const error = apiErrorFixtures.factories.createValidationError({
                username: "Username already taken",
                email: "Invalid email format",
            });
            
            expect(error.status).toBe(400);
            expect(error.code).toBe("VALIDATION_ERROR");
            expect(error.details.fields).toHaveProperty("username");
        });
    });

    describe("Network Error Handling", () => {
        it("should handle timeout errors", () => {
            const error = networkErrorFixtures.timeout.client;
            
            expect(error.error).toBeInstanceOf(Error);
            expect(error.error.message).toContain("timeout");
            expect(error.display?.retry).toBe(true);
            expect(error.metadata?.duration).toBe(30000);
        });

        it("should handle offline state", () => {
            const error = networkErrorFixtures.networkOffline;
            
            expect(error.display?.title).toBe("You're Offline");
            expect(error.display?.icon).toBe("wifi_off");
            expect(error.display?.retry).toBe(false);
        });

        it("should create retryable errors", () => {
            const error = networkErrorFixtures.factories.createRetryableError(
                "Connection failed",
                3,
                2,
            );
            
            expect(error.display?.retry).toBe(true);
            expect(error.display?.message).toContain("Attempt 2 of 3");
            expect(error.metadata?.attempt).toBe(2);
        });
    });

    describe("Validation Error Handling", () => {
        it("should handle form validation errors", () => {
            const errors = validationErrorFixtures.formErrors.registration;
            
            expect(errors.fields.email).toBe("Email is required");
            expect(errors.fields.password).toContain("8 characters");
            expect(errors.fields.confirmPassword).toBe("Passwords do not match");
        });

        it("should handle nested validation errors", () => {
            const errors = validationErrorFixtures.nested.project;
            
            expect(errors.name).toBe("Project name is required");
            expect((errors.team as any).id).toBe("Invalid team ID");
            expect(errors.tags).toContain("Duplicate tag");
        });

        it("should handle array field errors", () => {
            const errors = validationErrorFixtures.arrayErrors.emails;
            
            expect(errors.fields.emails).toBeInstanceOf(Array);
            expect(errors.fields.emails[0]).toBe("Invalid email format");
            expect(errors.fields.emails[1]).toBeUndefined();
        });
    });

    describe("Auth Error Handling", () => {
        it("should handle login failures", () => {
            const error = authErrorFixtures.login.accountLocked;
            
            expect(error.code).toBe("ACCOUNT_LOCKED");
            expect(error.details?.lockoutDuration).toBe(3600);
            expect(error.action?.type).toBe("verify");
            expect(error.action?.url).toBe("/auth/unlock");
        });

        it("should handle permission errors", () => {
            const error = authErrorFixtures.permissions.insufficientRole;
            
            expect(error.details?.requiredRole).toBe("admin");
            expect(error.details?.currentRole).toBe("member");
        });

        it("should handle session expiration", () => {
            const error = authErrorFixtures.session.expired;
            
            expect(error.code).toBe("SESSION_EXPIRED");
            expect(error.action?.type).toBe("login");
            expect(error.action?.url).toBe("/auth/login");
        });
    });

    describe("Business Error Handling", () => {
        it("should handle resource limits", () => {
            const error = businessErrorFixtures.limits.creditLimit;
            
            expect(error.type).toBe("limit");
            expect(error.details?.required).toBe(100);
            expect(error.details?.current).toBe(25);
            expect(error.userAction?.action).toBe("purchase_credits");
        });

        it("should handle workflow errors", () => {
            const error = businessErrorFixtures.workflow.prerequisiteNotMet;
            
            expect(error.type).toBe("workflow");
            expect(error.details?.missingSteps).toContain("email_verification");
            expect(error.userAction?.url).toBe("/onboarding");
        });

        it("should handle data conflicts", () => {
            const error = businessErrorFixtures.conflicts.versionConflict;
            
            expect(error.type).toBe("conflict");
            expect(error.details?.yourVersion).toBe("v1.2.3");
            expect(error.details?.currentVersion).toBe("v1.2.4");
        });
    });

    describe("System Error Handling", () => {
        it("should handle database errors", () => {
            const error = systemErrorFixtures.database.connectionLost;
            
            expect(error.severity).toBe("critical");
            expect(error.component).toBe("PostgreSQL");
            expect(error.recovery?.automatic).toBe(true);
            expect(error.recovery?.retryable).toBe(true);
        });

        it("should handle external service errors", () => {
            const error = systemErrorFixtures.externalService.aiServiceDown;
            
            expect(error.component).toBe("LLM Service");
            expect(error.recovery?.fallback).toBe("queue_for_later");
            expect(error.recovery?.estimatedRecovery).toBeDefined();
        });

        it("should handle infrastructure errors", () => {
            const error = systemErrorFixtures.infrastructure.memoryExhausted;
            
            expect(error.severity).toBe("critical");
            expect(error.recovery?.fallback).toBe("restart_process");
            expect(error.details?.metadata?.availableMemory).toBe("10MB");
        });
    });

    describe("Factory Function Usage", () => {
        it("should create custom errors using factories", () => {
            // API error factory
            const apiError = apiErrorFixtures.factories.createApiError(
                418,
                "TEAPOT",
                "I'm a teapot",
                { brewing: true },
            );
            expect(apiError.status).toBe(418);
            expect(apiError.details.brewing).toBe(true);

            // Business error factory
            const limitError = businessErrorFixtures.factories.createLimitError(
                "api_calls",
                1000,
                1000,
                "/billing/upgrade",
            );
            expect(limitError.type).toBe("limit");
            expect(limitError.userAction?.url).toBe("/billing/upgrade");

            // System error factory
            const systemError = systemErrorFixtures.factories.createCriticalError(
                "CustomService",
                "Service initialization failed",
                "use_backup_service",
            );
            expect(systemError.severity).toBe("critical");
            expect(systemError.recovery?.fallback).toBe("use_backup_service");
        });
    });
});