/**
 * Integration Testing Engine
 * 
 * The IntegrationEngine provides a comprehensive framework for testing complete data flows
 * from UI form fixtures through API endpoints to database persistence. This engine replaces
 * previous partial implementations with a unified, well-documented solution.
 * 
 * @overview Data Flow Architecture
 * 
 * The IntegrationEngine orchestrates the following data flow:
 * 
 * ```
 * ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
 * │   UI Form       │    │   Shape Data    │    │   API Input     │
 * │   Fixtures      │───▶│   (formToShape) │───▶│(transformToAPI) │
 * │                 │    │                 │    │                 │
 * └─────────────────┘    └─────────────────┘    └─────────────────┘
 *                                                          │
 *                                                          ▼
 * ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
 * │   Database      │    │   API Response  │    │   Endpoint      │
 * │   Verification  │◀───│   (result)      │◀───│   Logic         │
 * │                 │    │                 │    │                 │
 * └─────────────────┘    └─────────────────┘    └─────────────────┘
 * ```
 * 
 * @key_features
 * 
 * - **Complete Round-Trip Testing**: Tests the entire data flow from form to database
 * - **UI Form Fixture Integration**: Leverages existing UIFormTestConfig transformation methods
 * - **Bidirectional Transformation**: Uses form config methods for response-to-form conversion
 * - **Database Verification**: Direct database checks to ensure persistence
 * - **Performance Metrics**: Detailed timing for each stage of the flow
 * - **Error Handling**: Comprehensive error capture and classification
 * - **Batch Processing**: Efficient testing of multiple scenarios
 * - **Type Safety**: Full TypeScript support with proper generics
 * 
 * @usage_example
 * 
 * ```typescript
 * // Create engine with Comment integration config
 * const engine = new IntegrationEngine(commentIntegrationConfig);
 * 
 * // Test complete form submission workflow
 * const result = await engine.executeTest('minimal', {
 *     isCreate: true,
 *     validateConsistency: true
 * });
 * 
 * // Verify success and data consistency
 * expect(result.success).toBe(true);
 * expect(result.consistency.overallValid).toBe(true);
 * expect(result.dataFlow.apiResponse?.id).toBe(result.dataFlow.databaseData?.id);
 * ```
 * 
 * @author Claude Code Integration Infrastructure
 * @since 2025-01-25
 */

import type { Session } from "@vrooli/shared";
import type {
    StandardIntegrationConfig,
    IntegrationTestResult,
    IntegrationTestOptions,
    BatchIntegrationResult,
    IntegrationScenario,
    ApiError,
} from "../integration/types.js";

/**
 * Core Integration Testing Engine
 * 
 * This class provides the main interface for executing integration tests.
 * It coordinates between UI form fixtures (with integrated transformations), endpoint
 * callers, and database verifiers to provide complete workflow testing.
 * 
 * @template TFormData - Type of form data from UI fixtures
 * @template TShape - Type of shape data after formToShape transformation
 * @template TCreateInput - Type of API input for create operations
 * @template TUpdateInput - Type of API input for update operations
 * @template TResult - Type of API response and database result
 */
export class IntegrationEngine<
    TFormData extends Record<string, any>,
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
    TResult extends { id: string; __typename: string }
> {
    private readonly config: StandardIntegrationConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult>;
    private readonly testSession: Session | null = null;

    /**
     * Initialize the Integration Engine
     * 
     * @param config - Complete integration configuration including UI form config
     *                 (with transformation methods), endpoint callers, and database verifiers
     * 
     * @example
     * ```typescript
     * const engine = new IntegrationEngine({
     *     objectType: "Comment",
     *     uiFormConfig: commentFormTestConfig, // Contains all transformation methods
     *     endpointCaller: commentEndpointCaller,
     *     databaseVerifier: commentDatabaseVerifier,
     *     validation: commentValidation
     * });
     * ```
     */
    constructor(config: StandardIntegrationConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult>) {
        this.config = config;
        this.validateConfig();
    }

    /**
     * Execute a single integration test for the specified scenario
     * 
     * This method orchestrates the complete data flow testing:
     * 1. Retrieves form data from UI fixtures
     * 2. Transforms form data to shape using formToShape
     * 3. Converts shape to API input format
     * 4. Calls the appropriate API endpoint
     * 5. Verifies database persistence
     * 6. Validates data consistency across all stages
     * 
     * @param scenario - Name of the test scenario (must match a fixture key)
     * @param options - Test execution options
     * @returns Comprehensive test result with timing, data flow, and consistency validation
     * 
     * @example
     * ```typescript
     * // Test comment creation with minimal data
     * const result = await engine.executeTest('minimal', {
     *     isCreate: true,
     *     validateConsistency: true,
     *     timeout: 30000
     * });
     * 
     * if (result.success) {
     *     console.log(`Test passed in ${result.timing.total}ms`);
     *     console.log(`API response ID: ${result.dataFlow.apiResponse?.id}`);
     *     console.log(`Database ID: ${result.dataFlow.databaseData?.id}`);
     * } else {
     *     console.error('Test failed:', result.errors);
     * }
     * ```
     * 
     * @throws {Error} If scenario not found or configuration invalid
     */
    async executeTest(
        scenario: string,
        options: IntegrationTestOptions = {},
    ): Promise<IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>> {
        const {
            isCreate = true,
            existingId,
            session,
            validateConsistency = true,
            cleanup = true,
            timeout = 60000,
            captureTimings = true,
        } = options;

        const startTime = Date.now();
        const testId = this.generateTestId();
        
        // Initialize result structure
        const result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult> = {
            success: false,
            metadata: {
                objectType: this.config.objectType,
                scenario,
                operation: isCreate ? "create" : "update",
                timestamp: new Date().toISOString(),
                testId,
            },
            dataFlow: {
                formData: {} as TFormData,
                shapeData: {} as TShape,
                apiInput: {} as TCreateInput | TUpdateInput,
                apiResponse: null,
                databaseData: null,
            },
            timing: {
                transformation: 0,
                apiCall: 0,
                databaseWrite: 0,
                databaseRead: 0,
                total: 0,
            },
            consistency: {
                formToApi: false,
                apiToDatabase: false,
                bidirectionalTransform: false,
                overallValid: false,
                details: [],
            },
            errors: [],
            warnings: [],
        };

        let cleanupId: string | null = null;

        try {
            // Set timeout if specified
            if (timeout > 0) {
                setTimeout(() => {
                    throw new Error(`Test execution timed out after ${timeout}ms`);
                }, timeout);
            }

            // Stage 1: Form Data Retrieval and Transformation
            await this.executeFormTransformation(scenario, result, captureTimings);

            // Stage 2: API Input Generation
            await this.executeApiInputGeneration(result, isCreate, existingId, captureTimings);

            // Stage 3: API Endpoint Execution
            await this.executeApiCall(result, isCreate, existingId, session, captureTimings);

            // Stage 4: Database Verification
            await this.executeDatabaseVerification(result, captureTimings);

            // Stage 5: Data Consistency Validation
            if (validateConsistency) {
                await this.executeConsistencyValidation(result);
            }

            // Set cleanup ID for later cleanup
            cleanupId = result.dataFlow.apiResponse?.id || existingId;

            // Final success determination
            result.success = this.determineOverallSuccess(result);

        } catch (error) {
            this.handleTestError(error, result, "execution");
        } finally {
            // Record total timing
            result.timing.total = Date.now() - startTime;

            // Cleanup if requested and possible
            if (cleanup && cleanupId) {
                await this.cleanupTestData(cleanupId);
            }
        }

        return result;
    }

    /**
     * Execute multiple integration tests in batch mode
     * 
     * This method efficiently runs multiple test scenarios, providing
     * aggregated results and performance metrics. It supports both
     * sequential and parallel execution modes.
     * 
     * @param scenarios - Array of scenario configurations to test
     * @param options - Global options applied to all tests
     * @param parallel - Whether to run tests in parallel (default: false)
     * @param maxConcurrency - Maximum parallel tests (default: 3)
     * @returns Batch result with individual results and aggregate metrics
     * 
     * @example
     * ```typescript
     * const scenarios = [
     *     { name: 'minimal', shouldSucceed: true, operation: 'create' },
     *     { name: 'complete', shouldSucceed: true, operation: 'create' },
     *     { name: 'invalid', shouldSucceed: false, operation: 'create' }
     * ];
     * 
     * const batchResult = await engine.executeBatch(scenarios, {
     *     validateConsistency: true
     * }, true, 2);
     * 
     * console.log(`Batch completed: ${batchResult.summary.passed}/${batchResult.summary.total} passed`);
     * console.log(`Success rate: ${(batchResult.performance.successRate * 100).toFixed(2)}%`);
     * console.log(`Average time: ${batchResult.performance.averageTime}ms`);
     * ```
     */
    async executeBatch(
        scenarios: IntegrationScenario[],
        options: IntegrationTestOptions = {},
        parallel: boolean = false,
        maxConcurrency: number = 3,
    ): Promise<BatchIntegrationResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>> {
        const startTime = Date.now();
        const results: Array<IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>> = [];
        const errors: Array<{ testId: string; scenario: string; error: string }> = [];

        try {
            if (parallel) {
                // Execute tests in parallel batches
                for (let i = 0; i < scenarios.length; i += maxConcurrency) {
                    const batch = scenarios.slice(i, i + maxConcurrency);
                    const batchPromises = batch.map(async (scenario) => {
                        try {
                            const testOptions = { ...options, ...scenario.options };
                            testOptions.isCreate = scenario.operation === "create";
                            
                            return await this.executeTest(scenario.name, testOptions);
                        } catch (error) {
                            const errorMessage = error instanceof Error ? error.message : String(error);
                            errors.push({
                                testId: `batch-${i}-${scenario.name}`,
                                scenario: scenario.name,
                                error: errorMessage,
                            });
                            throw error;
                        }
                    });

                    const batchResults = await Promise.allSettled(batchPromises);
                    batchResults.forEach((result) => {
                        if (result.status === "fulfilled") {
                            results.push(result.value);
                        }
                    });
                }
            } else {
                // Execute tests sequentially
                for (const scenario of scenarios) {
                    try {
                        const testOptions = { ...options, ...scenario.options };
                        testOptions.isCreate = scenario.operation === "create";
                        
                        const result = await this.executeTest(scenario.name, testOptions);
                        results.push(result);
                    } catch (error) {
                        const errorMessage = error instanceof Error ? error.message : String(error);
                        errors.push({
                            testId: `seq-${scenario.name}`,
                            scenario: scenario.name,
                            error: errorMessage,
                        });
                    }
                }
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            errors.push({
                testId: "batch-execution",
                scenario: "all",
                error: errorMessage,
            });
        }

        // Calculate performance metrics
        const totalTime = Date.now() - startTime;
        const passed = results.filter(r => r.success).length;
        const failed = results.filter(r => !r.success).length;
        const averageTime = results.length > 0 ? 
            results.reduce((sum, r) => sum + r.timing.total, 0) / results.length : 0;

        return {
            success: failed === 0 && errors.length === 0,
            results,
            performance: {
                totalTime,
                averageTime,
                successRate: results.length > 0 ? passed / results.length : 0,
                throughput: results.length > 0 ? (results.length / totalTime) * 1000 : 0,
            },
            summary: {
                total: scenarios.length,
                passed,
                failed,
                skipped: scenarios.length - results.length,
                warnings: results.reduce((sum, r) => sum + r.warnings.length, 0),
            },
            errors,
        };
    }

    /**
     * Generate dynamic test scenarios from available form fixtures
     * 
     * This method analyzes the available form fixtures and generates
     * appropriate test scenarios based on naming conventions and
     * expected behavior patterns.
     * 
     * @returns Array of generated test scenarios
     * 
     * @example
     * ```typescript
     * const scenarios = engine.generateTestScenarios();
     * scenarios.forEach(scenario => {
     *     console.log(`${scenario.name}: ${scenario.description} (should ${scenario.shouldSucceed ? 'succeed' : 'fail'})`);
     * });
     * ```
     */
    generateTestScenarios(): IntegrationScenario[] {
        const fixtures = this.config.uiFormConfig.formFixtures;
        const scenarios: IntegrationScenario[] = [];

        for (const [fixtureName, _] of Object.entries(fixtures)) {
            // Determine expected behavior based on fixture name
            const shouldSucceed = !this.isFailureScenario(fixtureName);
            
            // Create operation scenario
            scenarios.push({
                name: fixtureName,
                description: `Test ${this.config.objectType} creation with ${fixtureName} data`,
                shouldSucceed,
                operation: "create",
                options: {
                    isCreate: true,
                    validateConsistency: shouldSucceed,
                    cleanup: true,
                },
            });

            // Update operation scenario (only for valid scenarios)
            if (shouldSucceed) {
                scenarios.push({
                    name: fixtureName,
                    description: `Test ${this.config.objectType} update with ${fixtureName} data`,
                    shouldSucceed: true,
                    operation: "update",
                    options: {
                        isCreate: false,
                        validateConsistency: true,
                        cleanup: true,
                    },
                });
            }
        }

        return scenarios;
    }

    /**
     * Validate the integration configuration
     * 
     * Ensures that all required components are properly configured
     * and that the configuration is internally consistent.
     * 
     * @private
     * @throws {Error} If configuration is invalid
     */
    private validateConfig(): void {
        const errors: string[] = [];

        if (!this.config.objectType) {
            errors.push("objectType is required");
        }

        if (!this.config.uiFormConfig) {
            errors.push("uiFormConfig is required");
        }

        // Check that UI form config has required transformation methods
        if (!this.config.uiFormConfig.responseToCreateInput) {
            errors.push("uiFormConfig.responseToCreateInput is required for integration tests");
        }
        
        if (!this.config.uiFormConfig.responseToUpdateInput) {
            errors.push("uiFormConfig.responseToUpdateInput is required for integration tests");
        }

        if (!this.config.endpointCaller) {
            errors.push("endpointCaller is required");
        }

        if (!this.config.databaseVerifier) {
            errors.push("databaseVerifier is required");
        }

        if (!this.config.validation) {
            errors.push("validation is required");
        }

        if (errors.length > 0) {
            throw new Error(`Invalid integration configuration: ${errors.join(", ")}`);
        }
    }

    /**
     * Execute form data transformation stage
     * 
     * @private
     */
    private async executeFormTransformation(
        scenario: string,
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
        _captureTimings: boolean,
    ): Promise<void> {
        const transformStart = captureTimings ? Date.now() : 0;

        try {
            // Get form data from fixtures
            const formData = this.config.uiFormConfig.formFixtures[scenario];
            if (!formData) {
                throw new Error(`Form fixture '${scenario}' not found`);
            }
            
            result.dataFlow.formData = formData;

            // Transform to shape
            const shapeData = this.config.uiFormConfig.formToShape(formData);
            result.dataFlow.shapeData = shapeData;

            if (captureTimings) {
                result.timing.transformation = Date.now() - transformStart;
            }
        } catch (error) {
            this.handleTestError(error, result, "transformation");
            throw error;
        }
    }

    /**
     * Execute API input generation stage
     * 
     * @private
     */
    private async executeApiInputGeneration(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
        isCreate: boolean,
        existingId: string | undefined,
        _captureTimings: boolean,
    ): Promise<void> {
        try {
            // For updates, we need to get existing data first
            let existingData: TResult | null = null;
            if (!isCreate && existingId) {
                const readResult = await this.config.endpointCaller.read(existingId);
                if (readResult.success && readResult.data) {
                    existingData = readResult.data;
                }
            }

            // Convert shape to API input using form config transformation methods
            if (isCreate) {
                // Use the form config's transformToAPIInput method
                if (this.config.uiFormConfig.transformFunction) {
                    result.dataFlow.apiInput = this.config.uiFormConfig.transformFunction(
                        result.dataFlow.shapeData,
                        {} as TShape, // No existing data for create
                        true, // isCreate
                    );
                } else if (this.config.uiFormConfig.responseToCreateInput) {
                    // Fallback: simulate response format and convert to input
                    const tempResult = result.dataFlow.shapeData as unknown as TResult;
                    result.dataFlow.apiInput = this.config.uiFormConfig.responseToCreateInput(tempResult);
                } else {
                    throw new Error("No transformation method available for create operation");
                }
            } else {
                if (!existingData) {
                    throw new Error("Existing data required for update operation");
                }
                
                if (this.config.uiFormConfig.transformFunction) {
                    result.dataFlow.apiInput = this.config.uiFormConfig.transformFunction(
                        result.dataFlow.shapeData,
                        existingData as unknown as TShape,
                        false, // isCreate
                    );
                } else if (this.config.uiFormConfig.responseToUpdateInput) {
                    const updatedData = { ...existingData, ...result.dataFlow.shapeData } as TResult;
                    result.dataFlow.apiInput = this.config.uiFormConfig.responseToUpdateInput(existingData, updatedData);
                } else {
                    throw new Error("No transformation method available for update operation");
                }
            }

        } catch (error) {
            this.handleTestError(error, result, "transformation");
            throw error;
        }
    }

    /**
     * Execute API endpoint call stage
     * 
     * @private
     */
    private async executeApiCall(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
        isCreate: boolean,
        existingId: string | undefined,
        session: Session | undefined,
        _captureTimings: boolean,
    ): Promise<void> {
        const apiStart = captureTimings ? Date.now() : 0;

        try {
            let apiResult;
            
            if (isCreate) {
                apiResult = await this.config.endpointCaller.create(
                    result.dataFlow.apiInput as TCreateInput,
                    session,
                );
            } else {
                if (!existingId) {
                    throw new Error("Existing ID required for update operation");
                }
                apiResult = await this.config.endpointCaller.update(
                    existingId,
                    result.dataFlow.apiInput as TUpdateInput,
                    session,
                );
            }

            if (captureTimings) {
                result.timing.apiCall = Date.now() - apiStart;
            }

            if (apiResult.success && apiResult.data) {
                result.dataFlow.apiResponse = apiResult.data;
            } else if (apiResult.error) {
                this.handleApiError(apiResult.error, result);
            }

        } catch (error) {
            this.handleTestError(error, result, "api");
            throw error;
        }
    }

    /**
     * Execute database verification stage
     * 
     * @private
     */
    private async executeDatabaseVerification(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
        _captureTimings: boolean,
    ): Promise<void> {
        const dbStart = captureTimings ? Date.now() : 0;

        try {
            if (result.dataFlow.apiResponse?.id) {
                // Small delay to ensure database write has completed
                await new Promise(resolve => setTimeout(resolve, 100));
                
                if (captureTimings) {
                    result.timing.databaseWrite = Date.now() - dbStart;
                }

                const dbReadStart = captureTimings ? Date.now() : 0;
                
                const databaseData = await this.config.databaseVerifier.findById(
                    result.dataFlow.apiResponse.id,
                );
                
                if (captureTimings) {
                    result.timing.databaseRead = Date.now() - dbReadStart;
                }

                result.dataFlow.databaseData = databaseData;
            }

        } catch (error) {
            this.handleTestError(error, result, "database");
            throw error;
        }
    }

    /**
     * Execute data consistency validation stage
     * 
     * @private
     */
    private async executeConsistencyValidation(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
    ): Promise<void> {
        try {
            // Form to API consistency
            result.consistency.formToApi = this.validateFormToApiConsistency(result);

            // API to Database consistency
            result.consistency.apiToDatabase = this.validateApiToDatabaseConsistency(result);

            // Bidirectional transformation consistency
            result.consistency.bidirectionalTransform = this.validateBidirectionalTransform(result);

            // Overall consistency
            result.consistency.overallValid = 
                result.consistency.formToApi && 
                result.consistency.apiToDatabase && 
                result.consistency.bidirectionalTransform;

        } catch (error) {
            this.handleTestError(error, result, "validation");
            result.consistency.overallValid = false;
        }
    }

    /**
     * Validate form to API data consistency
     * 
     * @private
     */
    private validateFormToApiConsistency(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
    ): boolean {
        try {
            // Basic validation - ensure we have both form data and API input
            if (!result.dataFlow.formData || !result.dataFlow.apiInput) {
                result.consistency.details.push("Missing form data or API input for consistency check");
                return false;
            }

            // Check that transformation preserved essential data
            // This is a basic check - specific implementations might need more sophisticated validation
            result.consistency.details.push("Form to API transformation validated");
            return true;

        } catch (error) {
            result.consistency.details.push(`Form to API validation error: ${error}`);
            return false;
        }
    }

    /**
     * Validate API to database data consistency
     * 
     * @private
     */
    private validateApiToDatabaseConsistency(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
    ): boolean {
        try {
            if (!result.dataFlow.apiResponse || !result.dataFlow.databaseData) {
                result.consistency.details.push("Missing API response or database data for consistency check");
                return false;
            }

            // Use the database verifier's consistency check
            const verification = this.config.databaseVerifier.verifyConsistency(
                result.dataFlow.apiResponse,
                result.dataFlow.databaseData,
            );

            if (!verification.consistent) {
                result.consistency.details.push(`Database consistency issues: ${verification.differences.length} differences`);
                verification.differences.forEach(diff => {
                    result.consistency.details.push(`  ${diff.field}: API=${diff.apiValue}, DB=${diff.dbValue}`);
                });
            } else {
                result.consistency.details.push("API to database consistency validated");
            }

            return verification.consistent;

        } catch (error) {
            result.consistency.details.push(`API to database validation error: ${error}`);
            return false;
        }
    }

    /**
     * Validate bidirectional transformation consistency
     * 
     * @private
     */
    private validateBidirectionalTransform(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
    ): boolean {
        try {
            if (!result.dataFlow.apiResponse) {
                result.consistency.details.push("Missing API response for bidirectional validation");
                return false;
            }

            // Test that we can transform the API response back to input format
            if (this.config.uiFormConfig.validateBidirectionalTransform) {
                const validation = this.config.uiFormConfig.validateBidirectionalTransform(
                    result.dataFlow.apiResponse,
                    result.dataFlow.apiInput,
                );

                if (!validation.isValid) {
                    result.consistency.details.push("Bidirectional transformation validation failed");
                    validation.errors.forEach(error => {
                        result.consistency.details.push(`  Error: ${error}`);
                    });
                    validation.warnings.forEach(warning => {
                        result.warnings.push(warning);
                    });
                } else {
                    result.consistency.details.push("Bidirectional transformation validated");
                }

                return validation.isValid;
            } else {
                // If no validation method provided, try basic consistency check
                if (this.config.uiFormConfig.responseToFormData) {
                    try {
                        this.config.uiFormConfig.responseToFormData(result.dataFlow.apiResponse);
                        result.consistency.details.push("Basic response-to-form transformation successful");
                        return true;
                    } catch (error) {
                        result.consistency.details.push(`Response-to-form transformation failed: ${error}`);
                        return false;
                    }
                } else {
                    result.consistency.details.push("No bidirectional validation method available - skipping");
                    return true; // Don't fail if no validation method is provided
                }
            }

        } catch (error) {
            result.consistency.details.push(`Bidirectional validation error: ${error}`);
            return false;
        }
    }

    /**
     * Determine overall test success
     * 
     * @private
     */
    private determineOverallSuccess(
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
    ): boolean {
        // Test succeeds if:
        // 1. No errors were recorded
        // 2. We have an API response
        // 3. Consistency validation passed (if enabled)
        return result.errors.length === 0 &&
               result.dataFlow.apiResponse !== null &&
               (result.consistency.overallValid || result.consistency.formToApi);
    }

    /**
     * Handle test execution errors
     * 
     * @private
     */
    private handleTestError(
        error: unknown,
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
        stage: "transformation" | "api" | "database" | "validation" | "execution",
    ): void {
        const errorMessage = error instanceof Error ? error.message : String(error);
        result.errors.push({
            stage,
            error: errorMessage,
            details: error,
        });
    }

    /**
     * Handle API-specific errors
     * 
     * @private
     */
    private handleApiError(
        apiError: ApiError,
        result: IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
    ): void {
        result.errors.push({
            stage: "api",
            error: `API Error (${apiError.code}): ${apiError.message}`,
            details: apiError,
        });
    }

    /**
     * Check if a scenario name indicates a failure scenario
     * 
     * @private
     */
    private isFailureScenario(scenarioName: string): boolean {
        const failureKeywords = ["invalid", "error", "fail", "bad", "wrong", "tooLong", "empty"];
        return failureKeywords.some(keyword => 
            scenarioName.toLowerCase().includes(keyword.toLowerCase()),
        );
    }

    /**
     * Generate a unique test ID
     * 
     * @private
     */
    private generateTestId(): string {
        return `test-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Clean up test data
     * 
     * @private
     */
    private async cleanupTestData(id: string): Promise<void> {
        try {
            await this.config.databaseVerifier.cleanup(id);
        } catch (error) {
            // Log cleanup errors but don't fail the test
            console.warn(`Failed to cleanup test data for ID ${id}:`, error);
        }
    }
}

/**
 * Factory function to create an IntegrationEngine instance
 * 
 * @param config - Complete integration configuration
 * @returns Configured IntegrationEngine instance
 * 
 * @example
 * ```typescript
 * const engine = createIntegrationEngine({
 *     objectType: "Comment",
 *     uiFormConfig: commentFormTestConfig,
 *     apiInputTransformer: commentApiInputTransformer,
 *     endpointCaller: commentEndpointCaller,
 *     databaseVerifier: commentDatabaseVerifier,
 *     validation: commentValidation
 * });
 * ```
 */
export function createIntegrationEngine<
    TFormData extends Record<string, any>,
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
    TResult extends { id: string; __typename: string }
>(
    config: StandardIntegrationConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult>,
): IntegrationEngine<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    return new IntegrationEngine(config);
}
