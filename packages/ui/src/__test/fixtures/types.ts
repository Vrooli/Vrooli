/**
 * Core type definitions for the production-grade UI fixtures architecture.
 * 
 * This file provides type-safe interfaces that eliminate the use of `any` types
 * and ensure proper integration with @vrooli/shared functions.
 */

// AI_CHECK: TYPE_SAFETY=replaced-17-any-types-with-specific-types | LAST: 2025-06-28

import type { 
    Bookmark, 
    BookmarkCreateInput, 
    BookmarkUpdateInput, 
    BookmarkFor,
    BookmarkList,
    BookmarkListCreateInput,
    BookmarkListUpdateInput,
    User, 
} from "@vrooli/shared";
import type { FormikProps } from "formik";
import type { RestHandler } from "msw";

// ============================================
// FORM DATA TYPES
// ============================================

/**
 * UI-specific form data for bookmark creation.
 * Includes UI-only fields that don't exist in the API.
 */
export interface BookmarkFormData {
    /** Type of object being bookmarked */
    bookmarkFor: BookmarkFor;
    /** ID of the object being bookmarked */
    forConnect: string;
    /** ID of existing list to add bookmark to */
    listId?: string;
    /** Whether to create a new list (UI-only field) */
    createNewList?: boolean;
    /** Label for new list (UI-only field) */
    newListLabel?: string;
    /** Whether the bookmark should be private (UI-only field) */
    isPrivate?: boolean;
}

/**
 * UI state for bookmark components
 */
export interface BookmarkUIState {
    isLoading: boolean;
    bookmark: Bookmark | null;
    error: string | null;
    isBookmarked: boolean;
    availableLists: Array<{ id: string; label: string }>;
    showListSelection: boolean;
}

// ============================================
// FIXTURE FACTORY INTERFACES
// ============================================

/**
 * Standard interface for all fixture factories.
 * Ensures consistent API across all object types.
 */
export interface FixtureFactory<TFormData, TCreateInput, TUpdateInput, TObject> {
    /** Object type identifier */
    readonly objectType: string;
    
    /** Create form data for different test scenarios */
    createFormData(scenario: string): TFormData;
    
    /** Transform form data to API create input using real shape functions */
    transformToAPIInput(formData: TFormData): TCreateInput;
    
    /** Create API update input */
    createUpdateInput(id: string, updates: Partial<TFormData>): TUpdateInput;
    
    /** Create mock API response */
    createMockResponse(overrides?: Partial<TObject>): TObject;
    
    /** Validate form data using real validation functions */
    validateFormData(formData: TFormData): Promise<ValidationResult>;
    
    /** Create MSW handlers for testing */
    createMSWHandlers(): MSWHandlers;
}

/**
 * Validation result with detailed error information
 */
export interface ValidationResult {
    isValid: boolean;
    errors?: string[];
    fieldErrors?: Record<string, string[]>;
}

/**
 * MSW handlers for different scenarios
 */
export interface MSWHandlers {
    success: RestHandler[];
    error: RestHandler[];
    loading: RestHandler[];
    networkError: RestHandler[];
}

// ============================================
// INTEGRATION TEST INTERFACES
// ============================================

/**
 * Configuration for integration tests
 */
export interface IntegrationTestConfig<TFormData> {
    formData: TFormData;
    shouldSucceed: boolean;
    expectedErrors?: string[];
    authContext?: AuthContext;
    databasePreConditions?: DatabasePreCondition[];
}

/**
 * Result of integration test execution
 */
export interface IntegrationTestResult<TObject, TFormData = Record<string, unknown>, TApiInput = Record<string, unknown>> {
    success: boolean;
    data?: {
        formData: TFormData;
        apiInput: TApiInput;
        apiResponse: TObject;
        databaseRecord: Record<string, unknown>;
        fetchedData: TObject;
    };
    errors?: string[];
    duration: number;
}

/**
 * Authentication context for tests
 */
export interface AuthContext {
    userId: string;
    handle: string;
    permissions: string[];
    teamMemberships?: string[];
}

/**
 * Database pre-conditions for tests
 */
export interface DatabasePreCondition {
    table: string;
    data: Record<string, unknown>;
    cleanup?: boolean;
}

// ============================================
// SCENARIO ORCHESTRATOR INTERFACES
// ============================================

/**
 * Configuration for multi-step scenario tests
 */
export interface ScenarioConfig {
    name: string;
    description: string;
    steps: ScenarioStep[];
    cleanup?: boolean;
}

/**
 * Individual step in a scenario
 */
export interface ScenarioStep {
    name: string;
    action: "create" | "update" | "delete" | "verify" | "wait";
    objectType: string;
    data?: Record<string, unknown>;
    assertions?: Assertion[];
    dependencies?: string[];
}

/**
 * Assertion for scenario steps
 */
export interface Assertion {
    type: "exists" | "equals" | "contains" | "count" | "custom";
    target: string;
    expected: unknown;
    message?: string;
}

/**
 * Result of scenario execution
 */
export interface ScenarioResult {
    success: boolean;
    stepResults: StepResult[];
    duration: number;
    error?: string;
    data?: Record<string, unknown>;
}

/**
 * Result of individual scenario step
 */
export interface StepResult {
    stepName: string;
    success: boolean;
    duration: number;
    data?: unknown;
    error?: string;
    assertions?: AssertionResult[];
}

/**
 * Result of assertion execution
 */
export interface AssertionResult {
    assertion: string;
    passed: boolean;
    actual?: unknown;
    expected?: unknown;
    message?: string;
}

// ============================================
// UI STATE INTERFACES
// ============================================

/**
 * Generic UI state fixture factory
 */
export interface UIStateFactory<TState, TData = unknown> {
    createLoadingState(context?: LoadingContext): TState;
    createErrorState(error: AppError): TState;
    createSuccessState(data: TData, message?: string): TState;
    createEmptyState(): TState;
    transitionToLoading(currentState: TState): TState;
    transitionToSuccess(currentState: TState, data: TData): TState;
    transitionToError(currentState: TState, error: AppError): TState;
}

/**
 * Loading context for UI states
 */
export interface LoadingContext {
    operation: string;
    progress?: number;
    message?: string;
}

/**
 * Application error interface
 */
export interface AppError {
    code: string;
    message: string;
    details?: Record<string, unknown>;
    recoverable?: boolean;
}

// ============================================
// COMPONENT TEST INTERFACES
// ============================================

/**
 * Component test utilities
 */
export interface ComponentTestUtils<TProps> {
    /** Render component with all required providers */
    renderWithProviders(props: TProps): ComponentRenderResult;
    
    /** Simulate form submission */
    simulateFormSubmission(formData: Record<string, unknown>): Promise<void>;
    
    /** Simulate field changes */
    simulateFieldChange(fieldName: string, value: unknown): Promise<void>;
    
    /** Wait for API calls to complete */
    waitForAPICall(endpoint: string): Promise<void>;
    
    /** Wait for specific state updates */
    waitForStateUpdate(predicate: () => boolean, timeout?: number): Promise<void>;
}

/**
 * Component render result with enhanced utilities
 */
export interface ComponentRenderResult {
    container: HTMLElement;
    rerender: (props: unknown) => void;
    unmount: () => void;
    
    // Form-specific utilities
    getFormValues: () => Record<string, unknown>;
    setFormValues: (values: Record<string, unknown>) => Promise<void>;
    submitForm: () => Promise<void>;
    
    // State utilities
    getComponentState: () => unknown;
    waitForState: (predicate: (state: unknown) => boolean) => Promise<void>;
    
    // Query utilities
    queryByTestId: (testId: string) => HTMLElement | null;
    getByTestId: (testId: string) => HTMLElement;
    queryByRole: (role: string) => HTMLElement | null;
    getByRole: (role: string) => HTMLElement;
}

// ============================================
// FORM INTEGRATION INTERFACES
// ============================================

/**
 * Formik integration utilities
 */
export interface FormIntegration<TFormData> {
    /** Create form instance with fixture data */
    createFormInstance(data: TFormData): FormikProps<TFormData>;
    
    /** Simulate form validation */
    validateForm(formInstance: FormikProps<TFormData>): Promise<ValidationResult>;
    
    /** Simulate form submission */
    submitForm(formInstance: FormikProps<TFormData>): Promise<unknown>;
    
    /** Set form errors */
    setFormErrors(formInstance: FormikProps<TFormData>, errors: Record<string, string>): void;
    
    /** Clear form */
    clearForm(formInstance: FormikProps<TFormData>): void;
}

// ============================================
// DATABASE INTEGRATION INTERFACES
// ============================================

/**
 * Database verification utilities
 */
export interface DatabaseVerifier {
    /** Verify record exists in database */
    verifyRecordExists(table: string, id: string): Promise<boolean>;
    
    /** Verify record data matches expected */
    verifyRecordData(table: string, id: string, expected: Record<string, unknown>): Promise<boolean>;
    
    /** Get record from database */
    getRecord(table: string, id: string): Promise<Record<string, unknown>>;
    
    /** Count records matching criteria */
    countRecords(table: string, criteria: Record<string, unknown>): Promise<number>;
    
    /** Clean up test data */
    cleanup(tables: string[]): Promise<void>;
}

/**
 * Test transaction manager
 */
export interface TestTransactionManager {
    /** Start new transaction */
    begin(): Promise<void>;
    
    /** Commit transaction */
    commit(): Promise<void>;
    
    /** Rollback transaction */
    rollback(): Promise<void>;
    
    /** Execute in transaction */
    executeInTransaction<T>(fn: () => Promise<T>): Promise<T>;
}

// ============================================
// API CLIENT INTERFACES
// ============================================

/**
 * Test API client for integration tests
 */
export interface TestAPIClient {
    /** Make GET request */
    get<T>(endpoint: string, options?: RequestOptions): Promise<APIResponse<T>>;
    
    /** Make POST request */
    post<T>(endpoint: string, data: unknown, options?: RequestOptions): Promise<APIResponse<T>>;
    
    /** Make PUT request */
    put<T>(endpoint: string, data: unknown, options?: RequestOptions): Promise<APIResponse<T>>;
    
    /** Make DELETE request */
    delete<T>(endpoint: string, options?: RequestOptions): Promise<APIResponse<T>>;
    
    /** Set authentication */
    setAuth(token: string): void;
    
    /** Clear authentication */
    clearAuth(): void;
}

/**
 * API request options
 */
export interface RequestOptions {
    headers?: Record<string, string>;
    timeout?: number;
    retries?: number;
}

/**
 * API response wrapper
 */
export interface APIResponse<T> {
    data: T;
    status: number;
    headers: Record<string, string>;
    duration: number;
}

// ============================================
// EXPORT HELPER TYPES
// ============================================

/**
 * All scenario types for type safety
 */
export type BookmarkScenario = 
    | "minimal" 
    | "complete" 
    | "invalid" 
    | "withNewList" 
    | "withExistingList" 
    | "forProject" 
    | "forRoutine";

/**
 * UI state types
 */
export type UIStateType = 
    | "loading" 
    | "error" 
    | "success" 
    | "empty" 
    | "withData";

/**
 * Test data generation utilities
 */
export interface TestDataGenerator {
    /** Generate unique ID */
    generateId(): string;
    
    /** Generate random string */
    generateString(length?: number): string;
    
    /** Generate random email */
    generateEmail(): string;
    
    /** Generate random date */
    generateDate(options?: { past?: boolean; future?: boolean; daysOffset?: number }): Date;
    
    /** Generate random number */
    generateNumber(min?: number, max?: number): number;
}
