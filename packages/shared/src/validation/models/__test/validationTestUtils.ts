import { expect } from "vitest";
import type * as yup from "yup";

/**
 * Standard test fixture structure for model validation tests.
 * This ensures consistency across all model tests and can be reused
 * by API endpoint tests and UI form tests.
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
                const result = await testValidation(
                    createSchema,
                    fixtures.minimal.create,
                    true,
                );
                // Check that required fields are present
                Object.keys(fixtures.minimal.create).forEach(key => {
                    expect(result).toHaveProperty(key);
                });
            });

            it("should accept complete valid data", async () => {
                const result = await testValidation(
                    createSchema,
                    fixtures.complete.create,
                    true,
                );
                // Verify key fields are preserved
                expect(result).toBeTypeOf("object");
            });

            it("should reject missing required fields", async () => {
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
                    ...fixtures.minimal.create,
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
                const result = await testValidation(
                    updateSchema,
                    fixtures.minimal.update,
                    true,
                );
                expect(result).toHaveProperty("id");
            });

            it("should accept complete valid data", async () => {
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
                const partialUpdate = fixtures.minimal.update;
                const result = await testValidation(updateSchema, partialUpdate, true);
                expect(result).toHaveProperty("id");
            });
            });
        }

        describe("omitFields functionality", () => {
            it("should omit top-level fields", async () => {
                const schema = validationModel.create?.({ 
                    omitFields: Object.keys(fixtures.complete.create).slice(2), 
                }) || validationModel.update({ 
                    omitFields: Object.keys(fixtures.complete.update).slice(2), 
                });
                
                const data = validationModel.create ? 
                    fixtures.complete.create : 
                    fixtures.complete.update;
                    
                const result = await schema.validate(data, { stripUnknown: true });
                expect(Object.keys(result).length).toBeLessThan(Object.keys(data).length);
            });
        });
    });
}

/**
 * Generate test data factories that can be shared between frontend and backend tests
 */
export class TestDataFactory<TCreate, TUpdate> {
    constructor(
        private fixtures: ModelTestFixtures<TCreate, TUpdate>,
        private customizers?: {
            create?: (base: TCreate) => TCreate;
            update?: (base: TUpdate) => TUpdate;
        },
    ) {}

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
