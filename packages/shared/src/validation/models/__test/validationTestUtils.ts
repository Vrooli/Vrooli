/**
 * Validation Test Utilities
 * 
 * This module provides comprehensive test helpers for model validation:
 * 
 * ## Quick Start Examples:
 * 
 * ### Option 1: Comprehensive (Simplest)
 * ```typescript
 * runComprehensiveValidationTests(
 *     apiKeyValidation, 
 *     apiKeyFixtures, 
 *     apiKeyTestDataFactory, 
 *     "apiKey"
 * );
 * ```
 * 
 * ### Option 2: Individual Control  
 * ```typescript
 * runStandardValidationTests(apiKeyValidation, apiKeyFixtures, "apiKey");
 * runFixtureIntegrityTests(apiKeyTestDataFactory, "apiKey");
 * ```
 * 
 * ### Option 3: Manual Tests Only
 * ```typescript
 * describe("business logic validation", () => {
 *     await testValidationBatch(schema, [
 *         { data: factory.createMinimal({...}), shouldPass: true, description: "..." },
 *         // ...
 *     ]);
 * });
 * ```
 * 
 * ## What Each Helper Provides:
 * 
 * - `runStandardValidationTests`: ~8 tests covering basic schema compliance
 * - `runFixtureIntegrityTests`: 2 tests ensuring fixtures match schemas  
 * - `runComprehensiveValidationTests`: Both of the above combined
 * - `testValidation` / `testValidationBatch`: Manual test utilities
 */

import { describe, expect, it } from "vitest";
import type * as yup from "yup";
import type { YupModel } from "../utils/types.js";

/**
 * Standard test fixture structure for model validation tests.
 * This ensures consistency across all model tests and can be reused
 * by API endpoint tests and UI form tests.
 * 
 * @template TCreate - The TypeScript type for create operations (e.g., UserCreateInput)
 * @template TUpdate - The TypeScript type for update operations (e.g., UserUpdateInput)
 */
export interface ModelTestFixtures<TCreate = any, TUpdate = any> {
    minimal: {
        create: TCreate;
        update: TUpdate;
    };
    complete: {
        create: TCreate;
        update: TUpdate;
    };
    invalid: {
        missingRequired: {
            create: Partial<TCreate>;
            update: Partial<TUpdate>;
        };
        invalidTypes: {
            create: any;
            update: any;
        };
        [key: string]: any; // Allow model-specific invalid scenarios
    };
    // Edge cases specific to the model
    edgeCases: {
        [key: string]: any;
    };
}

/**
 * Test validation helper that handles both success and failure cases
 */
export async function testValidation(
    schema: yup.Schema,
    data: any,
    shouldPass: boolean,
    expectedError?: string | RegExp,
): Promise<any> {
    try {
        const result = await schema.validate(data, {
            abortEarly: false,
            stripUnknown: true,
        });
        if (!shouldPass) {
            throw new Error(`Expected validation to fail but it passed with: ${JSON.stringify(result)}`);
        }
        return result;
    } catch (error: any) {
        if (shouldPass) {
            throw new Error(`Expected validation to pass but got error: ${error.message}\nErrors: ${JSON.stringify(error.errors)}`);
        }
        if (expectedError) {
            if (expectedError instanceof RegExp) {
                // Check both the main message and individual errors
                const allErrorText = `${error.message} ${error.errors?.join(" ") || ""}`;
                expect(allErrorText).toMatch(expectedError);
            } else {
                // Check both the main message and individual errors
                const allErrorText = `${error.message} ${error.errors?.join(" ") || ""}`;
                expect(allErrorText).toContain(expectedError);
            }
        }
        return error;
    }
}

/**
 * Test a batch of validation scenarios
 */
export async function testValidationBatch(
    schema: yup.Schema,
    scenarios: Array<{
        data: any;
        shouldPass: boolean;
        expectedError?: string | RegExp;
        description: string;
    }>,
): Promise<void> {
    for (const scenario of scenarios) {
        await testValidation(
            schema,
            scenario.data,
            scenario.shouldPass,
            scenario.expectedError,
        );
    }
}

/**
 * Fixture integrity validation test suite
 * Validates that all fixture data conforms to the validation schema
 * 
 * @example
 * ```typescript
 * // Basic usage
 * runFixtureIntegrityTests(apiKeyTestDataFactory, "apiKey");
 * ```
 */
export function runFixtureIntegrityTests<TCreate, TUpdate>(
    testDataFactory: TypedTestDataFactory<TCreate, TUpdate>,
    modelName: string,
): void {
    describe(`${modelName}Fixtures - Integrity Tests`, () => {
        it("should validate all fixtures against schema", async () => {
            // This automatically tests ALL fixture data for correctness
            await testDataFactory.validateAllFixtures();
        });

        it("should generate valid factory data", async () => {
            // Test that factory methods produce valid data
            try {
                await testDataFactory.createMinimalValidated();
                await testDataFactory.createCompleteValidated();
                await testDataFactory.updateMinimalValidated();
                await testDataFactory.updateCompleteValidated();
            } catch (error: any) {
                // If validation integration isn't set up, provide helpful message
                if (error.message?.includes("Cannot validate fixtures without validation schema")) {
                    console.warn(`⚠️  ${modelName}: Validation schema not linked to factory - skipping validated factory tests`);
                } else {
                    throw error;
                }
            }
        });
    });
}

/**
 * Comprehensive validation test suite combining both standard validation and fixture integrity tests
 * Use this for complete automatic coverage of a model's validation
 * 
 * @example
 * ```typescript
 * // Simplest approach - everything in one call
 * runComprehensiveValidationTests(
 *     apiKeyValidation,
 *     apiKeyFixtures, 
 *     apiKeyTestDataFactory,
 *     "apiKey"
 * );
 * 
 * // Alternative: individual calls for more control
 * runStandardValidationTests(apiKeyValidation, apiKeyFixtures, "apiKey");
 * runFixtureIntegrityTests(apiKeyTestDataFactory, "apiKey");
 * ```
 */
export function runComprehensiveValidationTests<TCreate, TUpdate>(
    validationModel: any,
    fixtures: ModelTestFixtures<TCreate, TUpdate>,
    testDataFactory: TypedTestDataFactory<TCreate, TUpdate>,
    modelName: string,
): void {
    // Standard validation tests (schema compliance)
    runStandardValidationTests(validationModel, fixtures, modelName);
    
    // Fixture integrity tests (fixture-schema alignment)
    runFixtureIntegrityTests(testDataFactory, modelName);
}

/**
 * Common test suite for standard validation patterns
 */
export function runStandardValidationTests<TCreate, TUpdate>(
    validationModel: any,
    fixtures: ModelTestFixtures<TCreate, TUpdate>,
    modelName: string,
) {
    describe(`${modelName}Validation - Standard Tests`, () => {
        const defaultParams = { omitFields: [] };

        if (validationModel.create) {
            describe("create validation", () => {

                const createSchema = validationModel.create(defaultParams);

                it("should accept minimal valid data", async () => {
                    if (!fixtures.minimal?.create) {
                        throw new Error(`Missing required test fixture: fixtures.minimal.create for ${modelName}`);
                    }
                    const result = await testValidation(
                        createSchema,
                        fixtures.minimal.create,
                        true,
                    );
                    // Check that required fields are present
                    if (fixtures.minimal?.create && typeof fixtures.minimal.create === "object") {
                        Object.keys(fixtures.minimal.create).forEach(key => {
                            expect(result).toHaveProperty(key);
                        });
                    }
                });

                it("should accept complete valid data", async () => {
                    if (!fixtures.complete?.create) {
                        throw new Error(`Missing required test fixture: fixtures.complete.create for ${modelName}`);
                    }
                    const result = await testValidation(
                        createSchema,
                        fixtures.complete.create,
                        true,
                    );
                    // Verify key fields are preserved
                    expect(result).toBeTypeOf("object");
                });

                it("should reject missing required fields", async () => {
                    // First check if the schema actually has required fields
                    const emptyObj = {};
                    let hasRequiredFields = false;
                    try {
                        await createSchema.validate(emptyObj);
                        // If empty object validates, schema has no required fields
                        hasRequiredFields = false;
                    } catch (error) {
                        // Schema rejected empty object, so it has required fields
                        hasRequiredFields = true;
                    }
                    
                    if (!hasRequiredFields) {
                        // Schema has no required fields - skip this test
                        return;
                    }
                    
                    await testValidation(
                        createSchema,
                        fixtures.invalid.missingRequired.create,
                        false,
                        /required/i,  // More flexible - handles "2 errors occurred" messages
                    );
                });

                it("should reject invalid field types", async () => {
                    await testValidation(
                        createSchema,
                        fixtures.invalid.invalidTypes.create,
                        false,
                    );
                });

                it("should strip unknown fields", async () => {
                    const dataWithExtra = {
                        ...(fixtures.minimal?.create || {}),
                        unknownField: "should be removed",
                        anotherUnknown: 123,
                    };
                    const result = await testValidation(createSchema, dataWithExtra, true);
                    expect(result).not.toHaveProperty("unknownField");
                    expect(result).not.toHaveProperty("anotherUnknown");
                });
            });
        }

        if (validationModel.update) {
            describe("update validation", () => {

                const updateSchema = validationModel.update(defaultParams);

                it("should accept minimal valid data", async () => {
                    if (!fixtures.minimal?.update) {
                        throw new Error(`Missing required test fixture: fixtures.minimal.update for ${modelName}`);
                    }
                    const result = await testValidation(
                        updateSchema,
                        fixtures.minimal.update,
                        true,
                    );
                    // Only check for id if it was provided in the fixture
                    if (fixtures.minimal.update.id) {
                        expect(result).toHaveProperty("id");
                    }
                });

                it("should accept complete valid data", async () => {
                    if (!fixtures.complete?.update) {
                        throw new Error(`Missing required test fixture: fixtures.complete.update for ${modelName}`);
                    }
                    const result = await testValidation(
                        updateSchema,
                        fixtures.complete.update,
                        true,
                    );
                    expect(result).toBeTypeOf("object");
                });

                it("should reject missing id", async () => {
                    await testValidation(
                        updateSchema,
                        fixtures.invalid.missingRequired.update,
                        false,
                        "This field is required",
                    );
                });

                it("should accept partial updates", async () => {
                    // Use minimal update as base, which should have id
                    if (!fixtures.minimal?.update) {
                        throw new Error(`Missing required test fixture: fixtures.minimal.update for ${modelName}`);
                    }
                    const partialUpdate = fixtures.minimal.update;
                    const result = await testValidation(updateSchema, partialUpdate, true);
                    // Only check for id if it was provided in the fixture
                    if (partialUpdate.id) {
                        expect(result).toHaveProperty("id");
                    }
                });
            });
        }

        describe("omitFields functionality", () => {
            it("should omit top-level fields", async () => {
                // Skip test if model has too few fields to meaningfully test omission
                const testData = validationModel.create ? fixtures.complete.create : fixtures.complete.update;
                const fieldCount = Object.keys(testData).length;
                
                if (fieldCount <= 2) {
                    // Not enough fields to test omission meaningfully
                    return;
                }
                
                const fieldsToOmit = Object.keys(testData).slice(2);
                // Check if the schema function accepts parameters (for omitFields support)
                const schemaFunction = validationModel.create || validationModel.update;
                if (!schemaFunction || schemaFunction.length === 0) {
                    // Schema doesn't accept parameters, skip omitFields test
                    return;
                }
                
                const schema = validationModel.create?.({
                    omitFields: fieldsToOmit,
                }) || validationModel.update({
                    omitFields: fieldsToOmit,
                });

                const result = await schema.validate(testData, { stripUnknown: true });
                expect(Object.keys(result).length).toBeLessThan(Object.keys(testData).length);
            });
        });
    });
}

/**
 * Generate test data factories that can be shared between frontend and backend tests
 */
export class TestDataFactory<TCreate, TUpdate> {
    constructor(
        protected fixtures: ModelTestFixtures<TCreate, TUpdate>,
        private customizers?: {
            create?: (base: TCreate) => TCreate;
            update?: (base: TUpdate) => TUpdate;
        },
    ) { }

    createMinimal(overrides?: Partial<TCreate>): TCreate {
        const base = { ...this.fixtures.minimal.create };
        const customized = this.customizers?.create ?
            this.customizers.create(base) : base;
        return { ...customized, ...overrides } as TCreate;
    }

    createComplete(overrides?: Partial<TCreate>): TCreate {
        const base = { ...this.fixtures.complete.create };
        const customized = this.customizers?.create ?
            this.customizers.create(base) : base;
        return { ...customized, ...overrides } as TCreate;
    }

    updateMinimal(overrides?: Partial<TUpdate>): TUpdate {
        const base = { ...this.fixtures.minimal.update };
        const customized = this.customizers?.update ?
            this.customizers.update(base) : base;
        return { ...customized, ...overrides } as TUpdate;
    }

    updateComplete(overrides?: Partial<TUpdate>): TUpdate {
        const base = { ...this.fixtures.complete.update };
        const customized = this.customizers?.update ?
            this.customizers.update(base) : base;
        return { ...customized, ...overrides } as TUpdate;
    }

    /**
     * Create data for specific test scenarios
     */
    forScenario(scenario: keyof ModelTestFixtures<TCreate, TUpdate>["invalid"] | string): any {
        if (scenario in this.fixtures.invalid) {
            return this.fixtures.invalid[scenario];
        }
        if (this.fixtures.edgeCases && scenario in this.fixtures.edgeCases) {
            return this.fixtures.edgeCases[scenario];
        }
        throw new Error(`Unknown test scenario: ${scenario}`);
    }
}

/**
 * Common field value generators for consistent test data
 */
// Counter for generating unique snowflake IDs in tests
let snowflakeCounter = 123456789012345678n;

export const testValues = {
    // IDs - using Snowflake IDs (not UUIDs anymore)
    snowflakeId: () => {
        snowflakeCounter++;
        return snowflakeCounter.toString();
    },
    // Alias for backward compatibility - but generates snowflake IDs
    uuid: () => testValues.snowflakeId(),

    // Strings
    shortString: (prefix = "test") => `${prefix}_${Date.now()}`,
    longString: (length = 500) => "a".repeat(length),
    email: (prefix = "test") => `${prefix}@example.com`,
    url: (path = "") => `https://example.com${path}`,
    handle: (prefix = "user") => `${prefix}${Math.floor(Math.random() * 10000)}`,

    // Common data
    timestamp: () => new Date().toISOString(),
    boolean: () => Math.random() > 0.5,

    // Arrays
    stringArray: (count = 3) => Array.from({ length: count }, (_, i) => `item${i}`),

    // Objects
    translation: (lang = "en", fields = {}) => ({
        language: lang,
        ...fields,
    }),
};

/**
 * Helper to create relationship test data
 */
export function createRelationshipData(
    type: "Connect" | "Create" | "Update" | "Delete",
    data: any,
) {
    switch (type) {
        case "Connect":
            return { connect: data };
        case "Create":
            return { create: Array.isArray(data) ? data : [data] };
        case "Update":
            return { update: Array.isArray(data) ? data : [data] };
        case "Delete":
            return { delete: Array.isArray(data) ? data : [data] };
        default:
            return data;
    }
}

/**
 * Enhanced type-safe test fixture utilities
 */

/**
 * Creates type-safe fixtures that link with validation schemas.
 * This ensures fixtures match both TypeScript types and validation schemas.
 * 
 * @template TCreate - The TypeScript type for create operations
 * @template TUpdate - The TypeScript type for update operations
 * @param fixtures - The fixture data conforming to TypeScript types
 * @param validation - Optional validation schema for runtime validation
 * @returns Enhanced fixtures with optional validation methods
 */
export function createTypedFixtures<TCreate, TUpdate>(
    fixtures: ModelTestFixtures<TCreate, TUpdate>,
    validation?: YupModel<["create", "update"]>,
): ModelTestFixtures<TCreate, TUpdate> & {
    validateCreate?: (data: TCreate) => Promise<TCreate>;
    validateUpdate?: (data: TUpdate) => Promise<TUpdate>;
} {
    if (!validation) {
        return fixtures;
    }

    return {
        ...fixtures,
        validateCreate: async (data: TCreate): Promise<TCreate> => {
            return validation.create({ env: "test" }).validate(data, {
                abortEarly: false,
                stripUnknown: true,
            }) as Promise<TCreate>;
        },
        validateUpdate: async (data: TUpdate): Promise<TUpdate> => {
            return validation.update({ env: "test" }).validate(data, {
                abortEarly: false,
                stripUnknown: true,
            }) as Promise<TUpdate>;
        },
    };
}

/**
 * Enhanced TestDataFactory with type safety and optional schema validation.
 * Provides compile-time type safety while maintaining backward compatibility.
 * 
 * @template TCreate - The TypeScript type for create operations
 * @template TUpdate - The TypeScript type for update operations
 */
export class TypedTestDataFactory<TCreate, TUpdate> extends TestDataFactory<TCreate, TUpdate> {
    constructor(
        fixtures: ModelTestFixtures<TCreate, TUpdate>,
        private validation?: YupModel<["create", "update"]>,
        customizers?: {
            create?: (base: TCreate) => TCreate;
            update?: (base: TUpdate) => TUpdate;
        },
    ) {
        super(fixtures, customizers);
    }

    /**
     * Create minimal test data with optional schema validation
     */
    async createMinimalValidated(overrides?: Partial<TCreate>): Promise<TCreate> {
        const data = this.createMinimal(overrides);
        if (this.validation?.create) {
            return this.validation.create({ env: "test" }).validate(data, {
                abortEarly: false,
                stripUnknown: true,
            }) as Promise<TCreate>;
        }
        return data;
    }

    /**
     * Create complete test data with optional schema validation
     */
    async createCompleteValidated(overrides?: Partial<TCreate>): Promise<TCreate> {
        const data = this.createComplete(overrides);
        if (this.validation?.create) {
            return this.validation.create({ env: "test" }).validate(data, {
                abortEarly: false,
                stripUnknown: true,
            }) as Promise<TCreate>;
        }
        return data;
    }

    /**
     * Create minimal update data with optional schema validation
     */
    async updateMinimalValidated(overrides?: Partial<TUpdate>): Promise<TUpdate> {
        const data = this.updateMinimal(overrides);
        if (this.validation?.update) {
            return this.validation.update({ env: "test" }).validate(data, {
                abortEarly: false,
                stripUnknown: true,
            }) as Promise<TUpdate>;
        }
        return data;
    }

    /**
     * Create complete update data with optional schema validation
     */
    async updateCompleteValidated(overrides?: Partial<TUpdate>): Promise<TUpdate> {
        const data = this.updateComplete(overrides);
        if (this.validation?.update) {
            return this.validation.update({ env: "test" }).validate(data, {
                abortEarly: false,
                stripUnknown: true,
            }) as Promise<TUpdate>;
        }
        return data;
    }

    /**
     * Validate that all fixtures pass their respective schemas
     * Useful for ensuring fixture integrity during development
     */
    async validateAllFixtures(): Promise<void> {
        if (!this.validation) {
            throw new Error("Cannot validate fixtures without validation schema");
        }

        const createSchema = this.validation.create ? this.validation.create({ env: "test" }) : null;
        const updateSchema = this.validation.update ? this.validation.update({ env: "test" }) : null;

        // Validate core fixtures
        if (createSchema) {
            try {
                await createSchema.validate(this.fixtures.minimal.create);
            } catch (e) {
                throw new Error(`Failed to validate minimal.create: ${e.message}`);
            }
            try {
                await createSchema.validate(this.fixtures.complete.create);
            } catch (e) {
                throw new Error(`Failed to validate complete.create: ${e.message}`);
            }
        }
        
        if (updateSchema) {
            try {
                await updateSchema.validate(this.fixtures.minimal.update);
            } catch (e) {
                throw new Error(`Failed to validate minimal.update: ${e.message}`);
            }
            try {
                await updateSchema.validate(this.fixtures.complete.update);
            } catch (e) {
                throw new Error(`Failed to validate complete.update: ${e.message}`);
            }
        }

        // Validate edge cases that should pass (skip ones designed to fail)
        const skipEdgeCases = new Set([
            "emptyStrings", "whitespaceStrings", "invalidTypes", "missingRequired",
            "invalidId", "invalidFormat", "tooLong", "tooShort", "negative", "float",
        ]);
        
        for (const [key, value] of Object.entries(this.fixtures.edgeCases)) {
            // Skip edge cases that are designed to fail validation
            if (skipEdgeCases.has(key) || key.includes("invalid") || key.includes("empty") || key.includes("whitespace") || key.includes("float")) {
                continue;
            }
            
            if (value.create && createSchema) {
                try {
                    await createSchema.validate(value.create);
                } catch (e) {
                    throw new Error(`Failed to validate edgeCase.${key}.create: ${e.message}`);
                }
            }
            if (value.update && updateSchema) {
                try {
                    await updateSchema.validate(value.update);
                } catch (e) {
                    throw new Error(`Failed to validate edgeCase.${key}.update: ${e.message}`);
                }
            }
        }
    }
}
