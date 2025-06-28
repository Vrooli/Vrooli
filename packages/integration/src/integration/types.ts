/**
 * Integration Testing Types and Interfaces
 * 
 * This module defines standardized interfaces for integration testing infrastructure
 * that bridges UI form fixtures with API endpoints and database validation.
 * 
 * Data Flow Architecture:
 * 1. UI Form Fixtures → formToShape → Shape Data
 * 2. Shape Data → transformToAPIInput → API Input
 * 3. API Input → Endpoint Logic → API Response
 * 4. API Response → Database Verification → Consistency Check
 * 
 * Note: ApiInputTransformer has been eliminated in favor of using transformation
 * methods directly in UIFormTestConfig for better cohesion and type safety.
 */
import type {
    ListObject,
    OrArray,
    Session,
    YupModel,
} from "@vrooli/shared";
import type { FormConfig, FormFixtures } from "@vrooli/shared";

/**
 * Core interface for standardized integration test configuration
 * 
 * This interface ensures consistent structure across all integration tests
 * and provides type safety for the complete data flow pipeline.
 */
export interface StandardIntegrationConfig<
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
    TResult extends OrArray<{ __typename: ListObject["__typename"]; id: string }>
> {
    /** Object type being tested */
    objectType: string;

    /** Form configuration from shared package */
    formConfig: FormConfig<TShape, TCreateInput, TUpdateInput, TResult>;

    /** Test fixtures for this form */
    fixtures: FormFixtures<TShape>;

    /** Endpoint caller for making actual API requests */
    endpointCaller: EndpointCaller<TCreateInput, TUpdateInput, TResult>;

    /** Database verifier for checking persistence */
    databaseVerifier: DatabaseVerifier<TResult>;

    /** Validation schema for API inputs */
    validation: YupModel<["create", "update"]>;
}

/**
 * @deprecated ApiInputTransformer interface has been eliminated.
 * 
 * Transformation methods are now included directly in UIFormTestConfig:
 * - responseToCreateInput?
 * - responseToUpdateInput?
 * - validateBidirectionalTransform?
 * 
 * This provides better type safety and keeps all form-related logic together.
 */

/**
 * Endpoint Caller interface
 * 
 * Handles actual API endpoint calls with proper authentication,
 * error handling, and response validation.
 */
export interface EndpointCaller<TCreateInput, TUpdateInput, TResult> {
    /**
     * Call create endpoint with input data
     */
    create(input: TCreateInput, session?: Session): Promise<{
        success: boolean;
        data?: TResult;
        error?: ApiError;
        timing: number;
    }>;

    /**
     * Call update endpoint with input data
     */
    update(id: string, input: TUpdateInput, session?: Session): Promise<{
        success: boolean;
        data?: TResult;
        error?: ApiError;
        timing: number;
    }>;

    /**
     * Call read endpoint to verify data
     */
    read(id: string, session?: Session): Promise<{
        success: boolean;
        data?: TResult;
        error?: ApiError;
        timing: number;
    }>;

    /**
     * Call delete endpoint for cleanup
     */
    delete(id: string, session?: Session): Promise<{
        success: boolean;
        error?: ApiError;
        timing: number;
    }>;
}

/**
 * Database Verifier interface
 * 
 * Provides direct database access for verifying that API operations
 * correctly persist data to the database.
 */
export interface DatabaseVerifier<TResult> {
    /**
     * Find record in database by ID
     */
    findById(id: string): Promise<TResult | null>;

    /**
     * Find record by unique constraints
     */
    findByConstraints(constraints: Record<string, any>): Promise<TResult | null>;

    /**
     * Verify that API result matches database state
     */
    verifyConsistency(apiResult: TResult, databaseResult: TResult): {
        consistent: boolean;
        differences: Array<{
            field: string;
            apiValue: any;
            dbValue: any;
        }>;
    };

    /**
     * Clean up test data
     */
    cleanup(id: string): Promise<void>;
}

/**
 * API Error interface for consistent error handling
 */
export interface ApiError {
    code: string;
    message: string;
    statusCode: number;
    details?: Record<string, any>;
    field?: string;
    path?: string;
}

/**
 * Integration test execution result
 * 
 * Comprehensive result object that captures all aspects of the integration test
 * including timing, data consistency, and error information.
 */
export interface IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    /** Overall test success */
    success: boolean;

    /** Test execution metadata */
    metadata: {
        objectType: string;
        scenario: string;
        operation: "create" | "update" | "read" | "delete";
        timestamp: string;
        testId: string;
    };

    /** Data flow stages */
    dataFlow: {
        formData: TFormData;
        shapeData: TShape;
        apiInput: TCreateInput | TUpdateInput;
        apiResponse: TResult | null;
        databaseData: TResult | null;
    };

    /** Performance metrics */
    timing: {
        transformation: number;
        apiCall: number;
        databaseWrite: number;
        databaseRead: number;
        total: number;
    };

    /** Data consistency validation */
    consistency: {
        formToApi: boolean;
        apiToDatabase: boolean;
        bidirectionalTransform: boolean;
        overallValid: boolean;
        details: string[];
    };

    /** Errors encountered during test execution */
    errors: Array<{
        stage: "transformation" | "api" | "database" | "validation";
        error: string;
        details?: any;
    }>;

    /** Warnings that don't cause test failure */
    warnings: string[];
}

/**
 * Integration test configuration options
 */
export interface IntegrationTestOptions {
    /** Whether this is a create or update operation */
    isCreate?: boolean;

    /** Existing object ID for update operations */
    existingId?: string;

    /** Session to use for API calls */
    session?: Session;

    /** Whether to validate data consistency */
    validateConsistency?: boolean;

    /** Whether to clean up test data after execution */
    cleanup?: boolean;

    /** Maximum execution time before timeout */
    timeout?: number;

    /** Whether to capture detailed timing metrics */
    captureTimings?: boolean;
}

/**
 * Batch integration test result
 * 
 * Results from running multiple integration tests in batch mode
 */
export interface BatchIntegrationResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    /** Overall batch success */
    success: boolean;

    /** Individual test results */
    results: Array<IntegrationTestResult<TFormData, TShape, TCreateInput, TUpdateInput, TResult>>;

    /** Batch performance metrics */
    performance: {
        totalTime: number;
        averageTime: number;
        successRate: number;
        throughput: number; // tests per second
    };

    /** Summary statistics */
    summary: {
        total: number;
        passed: number;
        failed: number;
        skipped: number;
        warnings: number;
    };

    /** Aggregated errors */
    errors: Array<{
        testId: string;
        scenario: string;
        error: string;
    }>;
}

/**
 * Integration scenario definition
 * 
 * Defines a specific test scenario with expected behavior
 */
export interface IntegrationScenario {
    /** Scenario name (matches fixture key) */
    name: string;

    /** Human-readable description */
    description: string;

    /** Whether this scenario should succeed */
    shouldSucceed: boolean;

    /** Expected operation type */
    operation: "create" | "update";

    /** Additional test options */
    options?: Partial<IntegrationTestOptions>;

    /** Custom validation rules for this scenario */
    customValidation?: (result: IntegrationTestResult<any, any, any, any, any>) => {
        valid: boolean;
        message?: string;
    };
}

/**
 * Factory configuration for creating integration test suites
 */
export interface IntegrationSuiteConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult> {
    /** Base integration configuration */
    baseConfig: StandardIntegrationConfig<TFormData, TShape, TCreateInput, TUpdateInput, TResult>;

    /** Test scenarios to execute */
    scenarios: IntegrationScenario[];

    /** Suite-level options */
    suiteOptions?: {
        /** Run tests in parallel */
        parallel?: boolean;

        /** Maximum concurrency for parallel execution */
        maxConcurrency?: number;

        /** Global timeout for all tests */
        globalTimeout?: number;

        /** Whether to stop on first failure */
        failFast?: boolean;

        /** Setup function to run before each test */
        beforeEach?: () => Promise<void>;

        /** Cleanup function to run after each test */
        afterEach?: () => Promise<void>;
    };
}

/**
 * Re-export commonly used types for convenience
 */
export type { ListObject, OrArray, Session, YupModel } from "@vrooli/shared";
export type { UIFormTestConfig } from "@vrooli/ui/src/__test/fixtures/form-testing/UIFormTestFactory.js";
