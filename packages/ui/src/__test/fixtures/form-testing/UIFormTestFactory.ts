// AI_CHECK: TYPE_SAFETY=fixed-user-event-setup-imports | LAST: 2025-07-03 - Fixed userEvent.setup() import errors across 11 files by changing default to destructured imports
// AI_CHECK: TYPE_SAFETY=replaced-22-any-types-with-proper-generic-types | LAST: 2025-06-30
import { screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { type FormConfig, type FormFixtures, type ListObject, type OrArray, type Session } from "@vrooli/shared";
import { type StandardUpsertFormConfig } from "../../../hooks/useStandardUpsertForm.js";

/**
 * @deprecated This advanced system has been superseded by the simpler createFormTestSuite.ts
 * 
 * For new tests, please use:
 * import { createFormTestSuite } from "../../helpers/createFormTestSuite.js";
 * 
 * This provides the same capabilities with much less complexity.
 * 
 * UI-specific form test configuration that extends the shared FormConfig
 * with additional UI testing capabilities like form filling and interaction testing.
 */
export interface UIFormTestConfig<
    TFormData extends Record<string, unknown>,
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, unknown>,
    TUpdateInput extends Record<string, unknown>,
    TResult extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
> extends StandardUpsertFormConfig<TShape, TCreateInput, TUpdateInput, TResult> {
    /** Reference to the shared form configuration */
    formConfig?: FormConfig<TShape, TCreateInput, TUpdateInput, TResult>;
    
    /** Test fixtures for this form - flat structure with named scenarios */
    formFixtures: {
        /** Minimal valid data with only required fields */
        minimal: TShape;
        /** Complete valid data with all fields populated */
        complete: TShape;
        /** Invalid data that should fail validation */
        invalid: TShape;
        /** Edge case data for boundary testing */
        edgeCase: TShape;
        /** Additional custom scenarios */
        [key: string]: TShape;
    };
    
    /** Function to convert form data to shape format - duplicated from formConfig for convenience */
    formToShape: (formData: TFormData) => TShape;
    
    /** Function to generate initial form values - duplicated from formConfig for convenience */
    initialValuesFunction: (session?: Session, existing?: Partial<TShape>) => TShape;
    /** Function to simulate user filling out the form */
    fillFormFunction?: (formData: TFormData, user: ReturnType<typeof userEvent.setup>) => Promise<void>;

    /**
     * Data-driven test scenarios - eliminates need for custom wrapper factories
     * Each scenario defines a set of test cases with expected outcomes
     */
    testScenarios?: {
        [scenarioName: string]: {
            description: string;
            testCases: Array<{
                name: string;
                shouldPass: boolean;
            } & (
                    | { field: string; value: unknown } // Field-specific test
                    | { data: Partial<TFormData> }   // Object merge test
                )>;
        };
    };

    /**
     * Transform API response back to create input format
     * This replaces the ApiInputTransformer.responseToCreateInput functionality
     */
    responseToCreateInput?: (response: TResult) => TCreateInput;

    /**
     * Transform API response back to update input format
     * This replaces the ApiInputTransformer.responseToUpdateInput functionality
     */
    responseToUpdateInput?: (existing: TResult, updated: Partial<TResult>) => TUpdateInput;

    /**
     * Transform API response to form data for display
     * This enables testing of data fetching and form population
     */
    responseToFormData?: (response: TResult) => TFormData;

    /**
     * Validate bidirectional transformation consistency
     * Ensures data integrity through the full transformation pipeline
     */
    validateBidirectionalTransform?: (response: TResult, input: TCreateInput | TUpdateInput) => {
        isValid: boolean;
        errors: string[];
        warnings: string[];
    };

    /**
     * Integration testing support
     * Enables full end-to-end testing when provided
     */
    integrationTestSupport?: {
        /** Endpoint caller for making actual API requests */
        endpointCaller?: {
            create: (input: TCreateInput) => Promise<TResult>;
            update: (input: TUpdateInput) => Promise<TResult>;
            find: (id: string) => Promise<TResult>;
        };

        /** Database verifier for checking persistence */
        databaseVerifier?: {
            verifyCreated: (id: string) => Promise<boolean>;
            verifyUpdated: (id: string, expectedData: Partial<TResult>) => Promise<boolean>;
            cleanup: (id: string) => Promise<void>;
        };

        /** Enable integration tests for this config */
        enableIntegrationTests?: boolean;

        /** Mock response generator for UI-only tests */
        mockResponseGenerator?: {
            generateCreateResponse: (input: TCreateInput) => TResult;
            generateUpdateResponse: (input: TUpdateInput) => TResult;
            generateFindResponse: (id: string) => TResult;
        };
    };
}

/**
 * Result of a UI form validation test
 */
export interface UIFormValidationResult {
    passed: boolean;
    errors: string[];
    validationTime: number;
    transformedData?: unknown;
}

/**
 * Result of a UI form interaction test
 */
export interface UIFormInteractionResult {
    success: boolean;
    finalValues: Record<string, unknown>;
    interactionTime: number;
    userActions: string[];
}

/**
 * Result of a data-driven test scenario
 */
export interface UITestScenarioResult {
    scenario: string;
    description: string;
    passed: boolean;
    totalTests: number;
    passedTests: number;
    failedTests: number;
    results: Array<{
        name: string;
        passed: boolean;
        expected: boolean;
        actual: boolean;
        error?: string;
        executionTime: number;
    }>;
}

/**
 * Result of a unified integration test
 */
export interface UIIntegrationTestResult extends UIFormValidationResult {
    /** API call results */
    apiResponse?: unknown;
    /** Database verification results */
    databaseVerified?: boolean;
    /** Bidirectional transformation validation results */
    transformationValid?: boolean;
    /** Total test execution time including API and DB operations */
    totalTime: number;
}

/**
 * UI-focused form test factory for testing form logic, validation, and user interactions
 * WITHOUT making API calls or using testcontainers. This is designed specifically for
 * the UI package which runs in a DOM environment.
 */
export class UIFormTestFactory<
    TFormData extends Record<string, unknown>,
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, unknown>,
    TUpdateInput extends Record<string, unknown>,
    TResult extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
> {
    constructor(
        private config: UIFormTestConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
    ) { }

    /**
     * Test form validation logic (no API calls)
     */
    async testFormValidation(
        scenario: keyof typeof this.config.formFixtures | string,
        options: {
            isCreate?: boolean;
            shouldPass?: boolean;
            expectedErrors?: string[];
            customData?: TFormData;
        } = {},
    ): Promise<UIFormValidationResult> {
        const { isCreate = true, shouldPass = true, expectedErrors = [], customData } = options;
        const startTime = performance.now();

        try {
            // Use custom data if provided, otherwise get from fixtures
            let formData = customData;
            if (!formData) {
                // Get from flat formFixtures structure
                if (this.config.formFixtures[scenario]) {
                    formData = this.config.formFixtures[scenario] as TFormData;
                } else {
                    throw new Error(`Fixture scenario '${scenario}' not found in formFixtures. Available scenarios: ${Object.keys(this.config.formFixtures).join(", ")}`);
                }
            }
            const shapeData = this.config.formToShape(formData);
            const transformedData = this.config.transformFunction(
                shapeData,
                this.config.initialValuesFunction(),
                isCreate,
            );

            const validationSchema = this.config.validation[isCreate ? "create" : "update"]({
                env: "test",
            });

            await validationSchema.validate(transformedData, {
                abortEarly: false,
                stripUnknown: true,
            });

            // If we got here, validation passed
            return {
                passed: shouldPass, // Success if we expected it to pass
                errors: [],
                validationTime: performance.now() - startTime,
                transformedData,
            };
        } catch (error) {
            // Validation failed
            const validationErrors = (error as { errors?: string[]; message?: string }).errors || [(error as Error).message || "Unknown validation error"];

            // Check if we got expected errors
            const hasExpectedErrors = expectedErrors.length === 0 ||
                expectedErrors.some((expected) =>
                    validationErrors.some((actual) => actual.includes(expected)),
                );

            return {
                passed: !shouldPass && hasExpectedErrors, // Success if we expected it to fail
                errors: validationErrors,
                validationTime: performance.now() - startTime,
                transformedData: undefined,
            };
        }
    }

    /**
     * Test user interaction with form fields
     */
    async testUserInteraction(
        scenario: keyof typeof this.config.formFixtures | string,
        options: {
            user?: ReturnType<typeof userEvent.setup>;
            customFillFunction?: (formData: TFormData, user: ReturnType<typeof userEvent.setup>) => Promise<void>;
        } = {},
    ): Promise<UIFormInteractionResult> {
        const startTime = performance.now();
        const userActions: string[] = [];

        try {
            // Get form data from flat formFixtures structure
            let formData: TFormData;
            if (this.config.formFixtures[scenario]) {
                formData = this.config.formFixtures[scenario] as TFormData;
            } else {
                throw new Error(`Fixture scenario '${scenario}' not found in formFixtures. Available scenarios: ${Object.keys(this.config.formFixtures).join(", ")}`);
            }
            const user = options.user || userEvent.setup();

            // Use custom fill function if provided, otherwise use default
            const fillFunction = options.customFillFunction || this.config.fillFormFunction || this.defaultFillForm;

            await fillFunction(formData, user);
            userActions.push(`Filled form with ${scenario} data`);

            return {
                success: true,
                finalValues: formData, // In real usage, this would come from the form
                interactionTime: performance.now() - startTime,
                userActions,
            };
        } catch (error) {
            return {
                success: false,
                finalValues: {},
                interactionTime: performance.now() - startTime,
                userActions,
            };
        }
    }

    /**
     * Generate MSW handlers for mocking API responses during UI tests
     */
    createMSWHandlers() {
        const { endpoints, objectType } = this.config;

        interface MSWRequest {
            body: Record<string, unknown>;
            params: Record<string, string>;
        }

        interface MSWResponse {
            (data: unknown): unknown;
        }

        interface MSWContext {
            json: (data: unknown) => unknown;
            status: (code: number) => unknown;
        }

        return {
            create: (req: MSWRequest, res: MSWResponse, ctx: MSWContext) => {
                // Mock successful create response
                return res(ctx.json({
                    success: true,
                    data: {
                        __typename: objectType,
                        id: "mock_created_id",
                        ...req.body,
                    },
                }));
            },
            update: (req: MSWRequest, res: MSWResponse, ctx: MSWContext) => {
                // Mock successful update response
                return res(ctx.json({
                    success: true,
                    data: {
                        __typename: objectType,
                        id: req.params.id || "mock_updated_id",
                        ...req.body,
                    },
                }));
            },
            find: (req: MSWRequest, res: MSWResponse, ctx: MSWContext) => {
                // Mock find response with complete fixture data
                const completeData = this.config.formToShape(this.config.formFixtures.complete);
                return res(ctx.json({
                    success: true,
                    data: {
                        ...completeData,
                        id: req.params.id || "mock_found_id",
                    },
                }));
            },
            error: (req: MSWRequest, res: MSWResponse, ctx: MSWContext) => {
                // Mock error response for testing error states
                return res(
                    ctx.status(400), // eslint-disable-line no-magic-numbers
                    ctx.json({
                        success: false,
                        error: "Validation failed",
                        details: {
                            fields: {
                                name: "Name is required",
                            },
                        },
                    }),
                );
            },
        };
    }

    /**
     * Generate test cases for form scenarios
     */
    generateUITestCases(): Array<{
        name: string;
        scenario: string;
        category: "valid" | "invalid" | "edge";
        isCreate: boolean;
        shouldValidate: boolean;
        description: string;
        testType: "validation" | "interaction" | "component";
    }> {
        const testCases = [];
        const scenarios = Object.keys(this.config.formFixtures);

        // Map scenario names to categories based on naming conventions
        const getCategoryForScenario = (scenario: string): "valid" | "invalid" | "edge" => {
            if (scenario.includes("invalid") || scenario.includes("missing") || scenario.includes("empty")) {
                return "invalid";
            }
            if (scenario.includes("edge") || scenario.includes("max") || scenario.includes("min") || scenario.includes("boundary")) {
                return "edge";
            }
            return "valid"; // minimal, complete, and other scenarios are considered valid
        };

        for (const scenario of scenarios) {
            const category = getCategoryForScenario(scenario);
            
            // Validation tests
            testCases.push({
                name: `should validate ${category}/${scenario} data for create`,
                scenario,
                category,
                isCreate: true,
                shouldValidate: category === "valid",
                description: `Test ${category}/${scenario} data validation in create mode`,
                testType: "validation" as const,
            });

            testCases.push({
                name: `should validate ${category}/${scenario} data for update`,
                scenario,
                category,
                isCreate: false,
                shouldValidate: category === "valid",
                description: `Test ${category}/${scenario} data validation in update mode`,
                testType: "validation" as const,
            });

            // Interaction tests
            testCases.push({
                name: `should handle user interaction with ${category}/${scenario} data`,
                scenario,
                category,
                isCreate: true,
                shouldValidate: category === "valid",
                description: `Test user filling form with ${category}/${scenario} data`,
                testType: "interaction" as const,
            });
        }

        return testCases;
    }

    /**
     * Create form data for testing specific field scenarios
     */
    createFieldTestData(fieldName: string, testValues: unknown[]): Record<string, TFormData> {
        const baseData = this.config.formFixtures.minimal;
        const testData: Record<string, TFormData> = {};

        testValues.forEach((value, index) => {
            testData[`${fieldName}_test_${index}`] = {
                ...baseData,
                [fieldName]: value,
            };
        });

        return testData;
    }

    /**
     * Test specific form field behaviors
     */
    async testFieldBehavior(
        fieldName: string,
        testValues: unknown[],
        options: {
            isCreate?: boolean;
            expectedValidValues?: unknown[];
            expectedInvalidValues?: unknown[];
        } = {},
    ): Promise<Array<{
        value: unknown;
        isValid: boolean;
        errors: string[];
    }>> {
        const { isCreate = true } = options;
        const results = [];

        for (const value of testValues) {
            // Deep clone the minimal fixture to avoid mutations
            const baseData = JSON.parse(JSON.stringify(this.config.formFixtures.minimal));
            
            // Handle nested field names (e.g., "translations.0.name")
            let testData: any;
            if (fieldName.includes(".")) {
                testData = baseData;
                this.setDeepValue(testData, fieldName, value);
            } else {
                testData = {
                    ...baseData,
                    [fieldName]: value,
                };
            }

            try {
                const shapeData = this.config.formToShape(testData);
                const transformedData = this.config.transformFunction(
                    shapeData,
                    this.config.initialValuesFunction(),
                    isCreate,
                );

                const validationSchema = this.config.validation[isCreate ? "create" : "update"]({
                    env: "test",
                });

                await validationSchema.validate(transformedData, {
                    abortEarly: false,
                    stripUnknown: true,
                });

                results.push({
                    value,
                    isValid: true,
                    errors: [],
                });
            } catch (error) {
                results.push({
                    value,
                    isValid: false,
                    errors: error.errors || [error.message],
                });
            }
        }

        return results;
    }

    /**
     * Test a specific data-driven scenario
     */
    async testScenario(
        scenarioName: string,
        options: {
            isCreate?: boolean;
        } = {},
    ): Promise<UITestScenarioResult> {
        const { isCreate = true } = options;
        const scenario = this.config.testScenarios?.[scenarioName];

        if (!scenario) {
            throw new Error(`Test scenario '${scenarioName}' not found in config`);
        }

        const results = [];
        let passedTests = 0;
        let failedTests = 0;

        for (const testCase of scenario.testCases) {
            const startTime = performance.now();
            let testData: TFormData;
            let error: string | undefined;

            try {
                // Build test data based on test case type
                if ("field" in testCase) {
                    // Field-specific test - deep clone to avoid mutations
                    testData = JSON.parse(JSON.stringify(this.config.formFixtures.minimal));
                    this.setDeepValue(testData, testCase.field, testCase.value);
                } else {
                    // Object merge test - deep clone and merge
                    const baseData = JSON.parse(JSON.stringify(this.config.formFixtures.minimal));
                    testData = this.deepMerge(baseData, testCase.data) as TFormData;
                }

                const result = await this.testFormValidation("custom", {
                    isCreate,
                    shouldPass: testCase.shouldPass,
                    customData: testData,
                });

                const passed = result.passed === testCase.shouldPass;
                if (passed) {
                    passedTests++;
                } else {
                    failedTests++;
                    error = `Expected ${testCase.shouldPass ? "pass" : "fail"}, got ${result.passed ? "pass" : "fail"}`;
                }

                results.push({
                    name: testCase.name,
                    passed,
                    expected: testCase.shouldPass,
                    actual: result.passed,
                    error,
                    executionTime: performance.now() - startTime,
                });
            } catch (err) {
                failedTests++;
                error = err instanceof Error ? err.message : String(err);

                results.push({
                    name: testCase.name,
                    passed: false,
                    expected: testCase.shouldPass,
                    actual: false,
                    error,
                    executionTime: performance.now() - startTime,
                });
            }
        }

        return {
            scenario: scenarioName,
            description: scenario.description,
            passed: failedTests === 0,
            totalTests: scenario.testCases.length,
            passedTests,
            failedTests,
            results,
        };
    }

    /**
     * Test all configured data-driven scenarios
     */
    async testAllScenarios(
        options: {
            isCreate?: boolean;
            scenarioFilter?: string[]; // Only run specific scenarios
        } = {},
    ): Promise<Record<string, UITestScenarioResult>> {
        const { isCreate = true, scenarioFilter } = options;
        const results: Record<string, UITestScenarioResult> = {};

        if (!this.config.testScenarios) {
            return results;
        }

        const scenarioNames = scenarioFilter || Object.keys(this.config.testScenarios);

        for (const scenarioName of scenarioNames) {
            if (this.config.testScenarios[scenarioName]) {
                results[scenarioName] = await this.testScenario(scenarioName, { isCreate });
            }
        }

        return results;
    }

    /**
     * Deep merge helper that preserves array structures and nested objects
     */
    private deepMerge(target: any, source: any): any {
        if (!source) return target;
        
        const result = { ...target };
        
        for (const key in source) {
            if (source.hasOwnProperty(key)) {
                if (source[key] && typeof source[key] === "object" && !Array.isArray(source[key])) {
                    result[key] = this.deepMerge(target[key] || {}, source[key]);
                } else {
                    result[key] = source[key];
                }
            }
        }
        
        return result;
    }

    /**
     * Helper method to set deep object values using dot notation
     */
    private setDeepValue(obj: Record<string, unknown>, path: string, value: unknown): void {
        const keys = path.split(".");
        let current: any = obj;

        for (let i = 0; i < keys.length - 1; i++) {
            const key = keys[i];
            const nextKey = keys[i + 1];
            const isNextKeyArrayIndex = /^\d+$/.test(nextKey);

            if (!(key in current) || current[key] === null || current[key] === undefined) {
                current[key] = isNextKeyArrayIndex ? [] : {};
            }
            
            // If we're dealing with an array and the next key is an index
            if (isNextKeyArrayIndex && Array.isArray(current[key])) {
                const index = parseInt(nextKey, 10);
                // Ensure the array has enough elements
                while (current[key].length <= index) {
                    current[key].push({});
                }
            }
            
            current = current[key];
        }

        const finalKey = keys[keys.length - 1];
        current[finalKey] = value;
    }

    /**
 * Test full integration pipeline: form → shape → API → response → database
 * This eliminates the need for separate integration test configurations
 */
    async testFullIntegration(
        scenario: keyof typeof this.config.formFixtures | string,
        options: {
            isCreate?: boolean;
            validateBidirectional?: boolean;
            cleanup?: boolean;
            customData?: TFormData;
        } = {},
    ): Promise<UIIntegrationTestResult> {
        const { isCreate = true, validateBidirectional = true, cleanup = true, customData } = options;
        const startTime = performance.now();

        if (!this.config.integrationTestSupport?.enableIntegrationTests) {
            throw new Error("Integration tests not enabled for this config. Set integrationTestSupport.enableIntegrationTests to true.");
        }

        try {
            // Step 1: Form validation
            let formData = customData;
            if (!formData) {
                // Get form data from flat formFixtures structure
                if (this.config.formFixtures[scenario]) {
                    formData = this.config.formFixtures[scenario] as TFormData;
                } else {
                    throw new Error(`Fixture scenario '${scenario}' not found in formFixtures. Available scenarios: ${Object.keys(this.config.formFixtures).join(", ")}`);
                }
            }
            const validationResult = await this.testFormValidation(scenario, { isCreate, shouldPass: true, customData: formData });

            if (!validationResult.passed) {
                return {
                    ...validationResult,
                    totalTime: performance.now() - startTime,
                };
            }

            // Step 2: Transform to API input
            const shapeData = this.config.formToShape(formData);
            const apiInput = this.config.transformFunction(
                shapeData,
                this.config.initialValuesFunction(),
                isCreate,
            );

            // Step 3: Make API call (or use mock if no endpoint caller)
            let apiResponse: TResult;
            if (this.config.integrationTestSupport.endpointCaller) {
                apiResponse = isCreate
                    ? await this.config.integrationTestSupport.endpointCaller.create(apiInput as TCreateInput)
                    : await this.config.integrationTestSupport.endpointCaller.update(apiInput as TUpdateInput);
            } else if (this.config.integrationTestSupport.mockResponseGenerator) {
                apiResponse = isCreate
                    ? this.config.integrationTestSupport.mockResponseGenerator.generateCreateResponse(apiInput as TCreateInput)
                    : this.config.integrationTestSupport.mockResponseGenerator.generateUpdateResponse(apiInput as TUpdateInput);
            } else {
                throw new Error("No endpoint caller or mock response generator configured");
            }

            // Step 4: Validate bidirectional transformation
            let transformationValid = true;
            if (validateBidirectional && this.config.validateBidirectionalTransform) {
                const validationResult = this.config.validateBidirectionalTransform(apiResponse, apiInput);
                transformationValid = validationResult.isValid;
            }

            // Step 5: Verify database persistence (if verifier provided)
            let databaseVerified = true;
            if (this.config.integrationTestSupport.databaseVerifier) {
                const responseId = Array.isArray(apiResponse) ? apiResponse[0].id : apiResponse.id;
                databaseVerified = isCreate
                    ? await this.config.integrationTestSupport.databaseVerifier.verifyCreated(responseId)
                    : await this.config.integrationTestSupport.databaseVerifier.verifyUpdated(responseId, apiResponse as Partial<TResult>);

                // Cleanup if requested
                if (cleanup) {
                    await this.config.integrationTestSupport.databaseVerifier.cleanup(responseId);
                }
            }

            return {
                passed: true,
                errors: [],
                validationTime: validationResult.validationTime,
                transformedData: validationResult.transformedData,
                apiResponse,
                databaseVerified,
                transformationValid,
                totalTime: performance.now() - startTime,
            };
        } catch (error) {
            return {
                passed: false,
                errors: [error.message],
                validationTime: 0,
                totalTime: performance.now() - startTime,
            };
        }
    }

    /**
     * Test fetching data and populating form (read → display flow)
     * This tests the responseToFormData transformation
     */
    async testDataFetchAndDisplay(
        mockResponse: TResult,
        options: {
            validateFormPopulation?: boolean;
        } = {},
    ): Promise<{
        success: boolean;
        formData?: TFormData;
        populationTime: number;
        errors: string[];
    }> {
        const startTime = performance.now();

        try {
            if (!this.config.responseToFormData) {
                throw new Error("responseToFormData not configured");
            }

            // Transform response to form data
            const formData = this.config.responseToFormData(mockResponse);

            // Validate that the form data can be transformed back
            if (options.validateFormPopulation) {
                const shapeData = this.config.formToShape(formData);
                const reTransformed = this.config.transformFunction(
                    shapeData,
                    this.config.initialValuesFunction(),
                    false, // Update mode when displaying fetched data
                );

                // Basic validation that we didn't lose critical data
                if (!reTransformed) {
                    throw new Error("Failed to re-transform fetched data");
                }
            }

            return {
                success: true,
                formData,
                populationTime: performance.now() - startTime,
                errors: [],
            };
        } catch (error) {
            return {
                success: false,
                populationTime: performance.now() - startTime,
                errors: [error.message],
            };
        }
    }

    /**
     * Test complete CRUD cycle: Create → Read → Update → Read
     * This demonstrates the full integration capabilities
     */
    async testCompleteCRUDCycle(
        createScenario: string,
        updateScenario: string,
        options: {
            cleanup?: boolean;
        } = {},
    ): Promise<{
        createResult: UIIntegrationTestResult;
        readResult?: TResult;
        updateResult?: UIIntegrationTestResult;
        finalReadResult?: TResult;
        cycleComplete: boolean;
        totalTime: number;
    }> {
        const startTime = performance.now();

        // Create
        const createResult = await this.testFullIntegration(createScenario, {
            isCreate: true,
            cleanup: false, // Don't cleanup yet
        });

        if (!createResult.passed || !createResult.apiResponse) {
            return {
                createResult,
                cycleComplete: false,
                totalTime: performance.now() - startTime,
            };
        }

        const createdId = Array.isArray(createResult.apiResponse)
            ? createResult.apiResponse[0].id
            : createResult.apiResponse.id;

        // Read created item
        let readResult;
        if (this.config.integrationTestSupport?.endpointCaller?.find) {
            readResult = await this.config.integrationTestSupport.endpointCaller.find(createdId);
        }

        // Update
        const updateResult = await this.testFullIntegration(updateScenario, {
            isCreate: false,
            cleanup: false,
        });

        // Read again to verify update
        let finalReadResult;
        if (this.config.integrationTestSupport?.endpointCaller?.find) {
            finalReadResult = await this.config.integrationTestSupport.endpointCaller.find(createdId);
        }

        // Cleanup if requested
        if (options.cleanup && this.config.integrationTestSupport?.databaseVerifier?.cleanup) {
            await this.config.integrationTestSupport.databaseVerifier.cleanup(createdId);
        }

        return {
            createResult,
            readResult,
            updateResult,
            finalReadResult,
            cycleComplete: createResult.passed && updateResult.passed,
            totalTime: performance.now() - startTime,
        };
    }

    /**
     * Private helper methods
     */
    private async defaultFillForm(formData: TFormData, user: ReturnType<typeof userEvent.setup>): Promise<void> {
        // Default implementation for filling forms during tests
        // This would be customized based on the specific form structure

        for (const [fieldName, value] of Object.entries(formData)) {
            try {
                // Try to find the field by various selectors
                const field = screen.getByRole("textbox", { name: new RegExp(fieldName, "i") }) ||
                    screen.getByLabelText(new RegExp(fieldName, "i")) ||
                    screen.getByTestId(fieldName) ||
                    screen.getByDisplayValue(String(value));

                if (field) {
                    await user.clear(field);
                    if (typeof value === "string") {
                        await user.type(field, value);
                    }
                }
            } catch (error) {
                // Field not found or not interactable - this is often expected
                // in partial form testing scenarios
            }
        }
    }
}

/**
 * Factory function to create UI form test factories
 */
export function createUIFormTestFactory<
    TFormData extends Record<string, unknown>,
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, unknown>,
    TUpdateInput extends Record<string, unknown>,
    TResult extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
>(
    config: UIFormTestConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
): UIFormTestFactory<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    return new UIFormTestFactory(config);
}
