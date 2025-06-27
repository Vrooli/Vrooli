/**
 * Comprehensive tests for ContextValidator
 * Tests the fixed validation logic against actual RunContext interface
 */

import { describe, it, expect, beforeEach } from "vitest";
import { ContextValidator, ContextValidationError } from "./contextValidator.js";
import type { RunContext, ContextScope } from "@vrooli/shared";
import type { ExecutionRunContext } from "../tier3/context/runContext.js";

describe("ContextValidator", () => {
    let validator: ContextValidator;

    beforeEach(() => {
        // Create a test logger that doesn't output during tests
        const logger = {
            debug: (): void => { /* test logger */ },
            info: (): void => { /* test logger */ },
            warn: (): void => { /* test logger */ },
            error: (): void => { /* test logger */ },
        };
        validator = new ContextValidator(logger);
    });

    describe("validateRunContext", () => {
        it("should validate a minimal valid RunContext", () => {
            const validContext: RunContext = {
                variables: {},
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(validContext);

            expect(result.valid).toBe(true);
            expect(result.errors).toEqual([]);
            expect(result.metadata.contextType).toBe("RunContext");
        });

        it("should validate a complete valid RunContext", () => {
            const scope: ContextScope = {
                id: "global",
                name: "Global Scope",
                variables: {
                    theme: "dark",
                    language: "en",
                },
            };

            const validContext: RunContext = {
                variables: {
                    userInput: "Hello world",
                    stepCount: 5,
                    isComplete: false,
                },
                blackboard: {
                    intermediateResult: { status: "processing", data: [1, 2, 3] },
                    lastAction: "user_input_received",
                },
                scopes: [scope],
            };

            const result = validator.validateRunContext(validContext);

            expect(result.valid).toBe(true);
            expect(result.errors).toEqual([]);
            expect(result.warnings?.length || 0).toBeGreaterThanOrEqual(0);
            expect(result.metadata.itemsValidated).toBeGreaterThan(0);
        });

        it("should reject null or undefined context", () => {
            const result1 = validator.validateRunContext(null as unknown as RunContext);
            const result2 = validator.validateRunContext(undefined as unknown as RunContext);

            expect(result1.valid).toBe(false);
            expect(result1.errors[0].type).toBe(ContextValidationError.INVALID_STRUCTURE);
            expect(result1.errors[0].severity).toBe("critical");

            expect(result2.valid).toBe(false);
            expect(result2.errors[0].type).toBe(ContextValidationError.INVALID_STRUCTURE);
        });

        it("should reject non-object context", () => {
            const result1 = validator.validateRunContext("not an object" as unknown as RunContext);
            const result2 = validator.validateRunContext(42 as unknown as RunContext);

            expect(result1.valid).toBe(false);
            expect(result1.errors[0].type).toBe(ContextValidationError.INVALID_STRUCTURE);

            expect(result2.valid).toBe(false);
            expect(result2.errors[0].type).toBe(ContextValidationError.INVALID_STRUCTURE);
        });

        it("should detect missing required fields", () => {
            const incompleteContext = {
                variables: {},
                // Missing blackboard and scopes
            } as RunContext;

            const result = validator.validateRunContext(incompleteContext);

            expect(result.valid).toBe(false);
            const missingFields = result.errors
                .filter(e => e.type === ContextValidationError.MISSING_REQUIRED_FIELD)
                .map(e => e.field);
            
            expect(missingFields).toContain("blackboard");
            expect(missingFields).toContain("scopes");
        });

        it("should validate variables Record type", () => {
            const invalidContext = {
                variables: ["not", "a", "record"] as unknown as Record<string, unknown>, // Array instead of Record
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(invalidContext);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => 
                e.type === ContextValidationError.INVALID_DATA_TYPE && 
                e.field === "variables",
            )).toBe(true);
        });

        it("should validate blackboard Record type", () => {
            const invalidContext = {
                variables: {},
                blackboard: "not a record" as unknown as Record<string, unknown>, // String instead of Record
                scopes: [],
            };

            const result = validator.validateRunContext(invalidContext);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => 
                e.type === ContextValidationError.INVALID_DATA_TYPE && 
                e.field === "blackboard",
            )).toBe(true);
        });

        it("should validate scopes array type", () => {
            const invalidContext = {
                variables: {},
                blackboard: {},
                scopes: "not an array" as unknown as ContextScope[], // String instead of Array
            };

            const result = validator.validateRunContext(invalidContext);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => 
                e.type === ContextValidationError.INVALID_DATA_TYPE && 
                e.field === "scopes",
            )).toBe(true);
        });

        it("should detect sensitive data in variable keys", () => {
            const contextWithSensitiveData: RunContext = {
                variables: {
                    userPassword: "secret123",
                    apiKey: "sk-123456",
                    secretToken: "hidden",
                    normalData: "safe",
                },
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(contextWithSensitiveData);

            // Should pass validation but have warnings/errors about sensitive data
            const sensitiveErrors = result.errors.filter(e => 
                e.type === ContextValidationError.SENSITIVE_DATA_EXPOSED,
            );

            expect(sensitiveErrors.length).toBeGreaterThan(0);
            expect(sensitiveErrors.some(e => e.field.includes("userPassword"))).toBe(true);
            expect(sensitiveErrors.some(e => e.field.includes("apiKey"))).toBe(true);
            expect(sensitiveErrors.some(e => e.field.includes("secretToken"))).toBe(true);
        });

        it("should warn about large objects", () => {
            const largeObject = {
                data: "x".repeat(150000), // Large string > 100KB
            };

            const contextWithLargeData: RunContext = {
                variables: {
                    largeDataSet: largeObject,
                },
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(contextWithLargeData);

            expect(result.warnings).toBeDefined();
            if (result.warnings) {
                expect(result.warnings.some(w => 
                    w.field.includes("largeDataSet") && 
                    w.message.includes("Large object"),
                )).toBe(true);
            }
        });

        it("should detect circular references", () => {
            const circularObject: Record<string, unknown> = { name: "test" };
            circularObject.self = circularObject; // Create circular reference

            const contextWithCircularRef: RunContext = {
                variables: {
                    circularData: circularObject,
                },
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(contextWithCircularRef);

            expect(result.errors.some(e => 
                e.type === ContextValidationError.CIRCULAR_REFERENCE &&
                e.field.includes("circularData"),
            )).toBe(true);
        });
    });

    describe("validateScopes", () => {
        it("should validate valid scopes", () => {
            const validScopes: ContextScope[] = [
                {
                    id: "global",
                    name: "Global Scope",
                    variables: { theme: "dark" },
                },
                {
                    id: "session",
                    name: "Session Scope",
                    parentId: "global",
                    variables: { userId: "123" },
                },
            ];

            const contextWithValidScopes: RunContext = {
                variables: {},
                blackboard: {},
                scopes: validScopes,
            };

            const result = validator.validateRunContext(contextWithValidScopes);

            expect(result.valid).toBe(true);
        });

        it("should detect duplicate scope IDs", () => {
            const duplicateScopes: ContextScope[] = [
                {
                    id: "duplicate",
                    name: "First Scope",
                    variables: {},
                },
                {
                    id: "duplicate", // Duplicate ID
                    name: "Second Scope", 
                    variables: {},
                },
            ];

            const contextWithDuplicates: RunContext = {
                variables: {},
                blackboard: {},
                scopes: duplicateScopes,
            };

            const result = validator.validateRunContext(contextWithDuplicates);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => 
                e.type === ContextValidationError.SCOPE_CONFLICT &&
                e.message.includes("Duplicate scope ID"),
            )).toBe(true);
        });

        it("should require scope ID", () => {
            const invalidScopes: ContextScope[] = [
                {
                    id: "", // Empty ID
                    name: "Invalid Scope",
                    variables: {},
                } as ContextScope,
            ];

            const contextWithInvalidScopes: RunContext = {
                variables: {},
                blackboard: {},
                scopes: invalidScopes,
            };

            const result = validator.validateRunContext(contextWithInvalidScopes);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => 
                e.type === ContextValidationError.MISSING_REQUIRED_FIELD &&
                e.field.includes("id"),
            )).toBe(true);
        });

        it("should warn about missing global scope", () => {
            const scopesWithoutGlobal: ContextScope[] = [
                {
                    id: "session",
                    name: "Session Scope",
                    variables: {},
                },
            ];

            const contextWithoutGlobal: RunContext = {
                variables: {},
                blackboard: {},
                scopes: scopesWithoutGlobal,
            };

            const result = validator.validateRunContext(contextWithoutGlobal);

            expect(result.warnings).toBeDefined();
            if (result.warnings) {
                expect(result.warnings.some(w => 
                    w.message.includes("No global scope found"),
                )).toBe(true);
            }
        });

        it("should warn about excessive scope depth", () => {
            const manyScopes: ContextScope[] = [];
            for (let i = 0; i < 15; i++) { // Exceeds maxScopeDepth of 10
                manyScopes.push({
                    id: `scope_${i}`,
                    name: `Scope ${i}`,
                    variables: {},
                });
            }

            const contextWithManyScopes: RunContext = {
                variables: {},
                blackboard: {},
                scopes: manyScopes,
            };

            const result = validator.validateRunContext(contextWithManyScopes);

            expect(result.warnings).toBeDefined();
            if (result.warnings) {
                expect(result.warnings.some(w => 
                    w.message.includes("Scope depth") && 
                    w.message.includes("exceeds recommended maximum"),
                )).toBe(true);
            }
        });
    });

    // Modern context validation tests for RunExecutionContext and ExecutionRunContext
    // Use RunExecutionContext validation instead

    describe("ExecutionRunContext validation", () => {
        it("should validate ExecutionRunContext with required fields", () => {
            const executionContext: ExecutionRunContext = {
                runId: "run_123",
                routineId: "routine_456",
                routineName: "Test Routine",
                userData: {
                    id: "user_789",
                    languages: ["en", "es"],
                },
                environment: {
                    NODE_ENV: "test",
                    LOG_LEVEL: "debug",
                },
                getAllMetadata: () => ({
                    startTime: Date.now(),
                    version: "1.0.0",
                }),
            };

            const result = validator.validateExecutionRunContext(executionContext);

            expect(result.valid).toBe(true);
            expect(result.metadata.contextType).toBe("ExecutionRunContext");
        });

        it("should require essential fields for ExecutionRunContext", () => {
            const incompleteExecutionContext = {
                runId: "run_123",
                // Missing routineId, routineName, userData
            } as ExecutionRunContext;

            const result = validator.validateExecutionRunContext(incompleteExecutionContext);

            expect(result.valid).toBe(false);
            
            const missingFields = result.errors
                .filter(e => e.type === ContextValidationError.MISSING_REQUIRED_FIELD)
                .map(e => e.field);
            
            expect(missingFields).toContain("routineId");
            expect(missingFields).toContain("routineName");
            expect(missingFields).toContain("userData");
        });

        it("should validate userData structure", () => {
            const executionContextInvalidUser: ExecutionRunContext = {
                runId: "run_123",
                routineId: "routine_456", 
                routineName: "Test Routine",
                userData: {
                    // Missing id
                    languages: "not an array" as any, // Invalid type
                },
                environment: {},
                getAllMetadata: () => ({}),
            };

            const result = validator.validateExecutionRunContext(executionContextInvalidUser);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => 
                e.field === "userData.id" &&
                e.type === ContextValidationError.MISSING_REQUIRED_FIELD,
            )).toBe(true);
            expect(result.errors.some(e => 
                e.field === "userData.languages" &&
                e.type === ContextValidationError.INVALID_DATA_TYPE,
            )).toBe(true);
        });

        it("should validate environment variables are strings", () => {
            const executionContextInvalidEnv: ExecutionRunContext = {
                runId: "run_123",
                routineId: "routine_456",
                routineName: "Test Routine", 
                userData: { id: "user_123" },
                environment: {
                    NODE_ENV: "test",
                    PORT: 3000 as unknown as string, // Should be string, not number
                    DEBUG: true as unknown as string, // Should be string, not boolean
                },
                getAllMetadata: () => ({}),
            };

            const result = validator.validateExecutionRunContext(executionContextInvalidEnv);

            expect(result.errors.some(e => 
                e.field.includes("environment.PORT") &&
                e.type === ContextValidationError.INVALID_DATA_TYPE,
            )).toBe(true);
            expect(result.errors.some(e => 
                e.field.includes("environment.DEBUG") &&
                e.type === ContextValidationError.INVALID_DATA_TYPE,
            )).toBe(true);
        });
    });

    describe("validation helpers", () => {
        it("should detect sensitive patterns correctly", () => {
            const contextWithVarious: RunContext = {
                variables: {
                    userName: "safe", // Not sensitive
                    userPassword: "secret", // Sensitive
                    apiKey: "key123", // Sensitive  
                    secretValue: "hidden", // Sensitive
                    regularData: "normal", // Not sensitive
                    tokenData: "bearer123", // Sensitive
                    credentials: "auth", // Sensitive
                    privateKey: "pem", // Sensitive
                },
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(contextWithVarious);

            const sensitiveErrors = result.errors.filter(e => 
                e.type === ContextValidationError.SENSITIVE_DATA_EXPOSED,
            );

            // Should detect all sensitive patterns
            expect(sensitiveErrors.length).toBeGreaterThan(0);
            
            const sensitiveFields = sensitiveErrors.map(e => e.field);
            expect(sensitiveFields.some(f => f.includes("userPassword"))).toBe(true);
            expect(sensitiveFields.some(f => f.includes("apiKey"))).toBe(true);
            expect(sensitiveFields.some(f => f.includes("secretValue"))).toBe(true);
            expect(sensitiveFields.some(f => f.includes("tokenData"))).toBe(true);
            expect(sensitiveFields.some(f => f.includes("credentials"))).toBe(true);
            expect(sensitiveFields.some(f => f.includes("privateKey"))).toBe(true);
            
            // Should not flag non-sensitive fields
            expect(sensitiveFields.some(f => f.includes("userName"))).toBe(false);
            expect(sensitiveFields.some(f => f.includes("regularData"))).toBe(false);
        });

        it("should handle variable size validation correctly", () => {
            const smallObject = { data: "small" };
            const largeObject = { data: "x".repeat(2000000) }; // > 1MB limit

            const contextWithSizes: RunContext = {
                variables: {
                    small: smallObject,
                    large: largeObject,
                },
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(contextWithSizes);

            expect(result.errors.some(e => 
                e.type === ContextValidationError.VARIABLE_SIZE_EXCEEDED &&
                e.field.includes("large"),
            )).toBe(true);

            // Small object should not trigger size error
            expect(result.errors.some(e => 
                e.type === ContextValidationError.VARIABLE_SIZE_EXCEEDED &&
                e.field.includes("small"),
            )).toBe(false);
        });
    });

    describe("validation result structure", () => {
        it("should return consistent validation result structure", () => {
            const validContext: RunContext = {
                variables: { test: "data" },
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(validContext);

            // Check result structure
            expect(result).toHaveProperty("valid");
            expect(result).toHaveProperty("errors");
            expect(result).toHaveProperty("warnings");
            expect(result).toHaveProperty("metadata");

            // Check metadata structure
            expect(result.metadata).toHaveProperty("validatedAt");
            expect(result.metadata).toHaveProperty("contextType");
            expect(result.metadata).toHaveProperty("itemsValidated");

            expect(result.metadata.validatedAt).toBeInstanceOf(Date);
            expect(typeof result.metadata.contextType).toBe("string");
            expect(typeof result.metadata.itemsValidated).toBe("number");
        });

        it("should provide helpful error messages with suggestions", () => {
            const invalidContext = {
                variables: ["not", "an", "object"] as unknown as Record<string, unknown>,
                blackboard: {},
                scopes: [],
            };

            const result = validator.validateRunContext(invalidContext);

            const dataTypeError = result.errors.find(e => 
                e.type === ContextValidationError.INVALID_DATA_TYPE,
            );

            expect(dataTypeError).toBeDefined();
            if (dataTypeError) {
                expect(dataTypeError.message).toContain("Record<string, unknown>");
                expect(dataTypeError.severity).toBeDefined();
            }
        });
    });
});
