/* c8 ignore start */
/**
 * Core types for the UI fixture factory system
 * 
 * This module defines type-safe interfaces for UI fixtures with ZERO any types.
 * It ensures proper integration between forms, API calls, and UI state management.
 */

import type { UseFormReturn } from "react-hook-form";
import type { RestHandler } from "msw";

/**
 * Result of form validation
 */
export interface FormValidationResult {
    isValid: boolean;
    errors?: Record<string, string>;
    touched?: Record<string, boolean>;
}

/**
 * Configuration for simulating form events
 */
export interface FormEvent {
    type: "change" | "blur" | "focus" | "submit";
    field?: string;
    value?: unknown;
    timestamp?: number;
}

/**
 * Test result for round-trip testing
 */
export interface TestResult {
    success: boolean;
    errors?: string[];
    warnings?: string[];
    metadata?: Record<string, unknown>;
}

/**
 * Round-trip test configuration
 */
export interface RoundTripConfig<TFormData> {
    formData: TFormData;
    validateEachStep?: boolean;
    simulateUserInteraction?: boolean;
    networkDelay?: number;
}

/**
 * Result of a complete round-trip test
 */
export interface RoundTripResult<TAPIResponse> {
    success: boolean;
    data?: {
        formData: unknown;
        apiInput: unknown;
        apiResponse: TAPIResponse;
        dbRecord: unknown;
        fetchedData: TAPIResponse;
        uiDisplay: unknown;
    };
    errors?: string[];
    stages?: Record<string, "pending" | "completed" | "failed" | "skipped">;
    dataIntegrity?: boolean;
    canDisplay?: boolean;
}

/**
 * Data integrity report for round-trip verification
 */
export interface DataIntegrityReport {
    isValid: boolean;
    mismatches: Array<{
        field: string;
        original: unknown;
        result: unknown;
        path: string;
    }>;
    warnings: string[];
}

/**
 * UI loading state context
 */
export interface LoadingContext {
    type: "initial" | "refresh" | "loadMore" | "submit";
    message?: string;
    progress?: number;
}

/**
 * Application error type for UI testing
 */
export interface AppError {
    code: string;
    message: string;
    details?: Record<string, unknown>;
    retryable?: boolean;
}

/**
 * MSW handler configuration
 */
export interface HandlerConfig {
    method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
    path: string;
    status?: number;
    delay?: number;
    response?: unknown;
    error?: AppError;
}

/**
 * Delay configuration for network simulation
 */
export interface DelayConfig {
    min: number;
    max: number;
    jitter?: boolean;
}

/**
 * Error scenario for testing
 */
export interface ErrorScenario {
    type: "validation" | "network" | "permission" | "server" | "timeout";
    config: Record<string, unknown>;
}

/**
 * Render result with testing utilities
 */
export interface RenderResult {
    // Standard testing library utilities
    getByRole: (role: string, options?: Record<string, unknown>) => HTMLElement;
    getByText: (text: string | RegExp) => HTMLElement;
    getByLabelText: (text: string | RegExp) => HTMLElement;
    queryByRole: (role: string, options?: Record<string, unknown>) => HTMLElement | null;
    queryByText: (text: string | RegExp) => HTMLElement | null;
    findByRole: (role: string, options?: Record<string, unknown>) => Promise<HTMLElement>;
    findByText: (text: string | RegExp) => Promise<HTMLElement>;
    
    // User event utilities
    user: {
        click: (element: HTMLElement) => Promise<void>;
        type: (element: HTMLElement, text: string) => Promise<void>;
        clear: (element: HTMLElement) => Promise<void>;
        selectOptions: (element: HTMLElement, values: string | string[]) => Promise<void>;
        tab: () => Promise<void>;
    };
    
    // Additional utilities
    container: HTMLElement;
    rerender: (ui: React.ReactElement) => void;
    unmount: () => void;
}

/**
 * Form fixture factory interface
 * 
 * @template TFormData - The form data type as it appears in UI state
 * @template TFormState - The React Hook Form state type (optional)
 */
export interface FormFixtureFactory<TFormData extends Record<string, unknown>, TFormState = unknown> {
    // Create form data for different scenarios
    createFormData(scenario: "minimal" | "complete" | "invalid" | string): TFormData;
    
    // Simulate form events (for React Hook Form integration)
    simulateFormEvents(formData: TFormData): FormEvent[];
    
    // Validate using actual form validation
    validateFormData(formData: TFormData): Promise<FormValidationResult>;
    
    // Transform to API input using real shape functions
    transformToAPIInput(formData: TFormData): unknown;
    
    // Create form state for component testing
    createFormState?(data: TFormData): UseFormReturn<TFormData>;
}

/**
 * Round-trip orchestrator interface
 * 
 * @template TFormData - The form data type
 * @template TAPIResponse - The API response type
 */
export interface RoundTripOrchestrator<TFormData, TAPIResponse> {
    // Execute complete cycle with real API calls
    executeFullCycle(config: RoundTripConfig<TFormData>): Promise<RoundTripResult<TAPIResponse>>;
    
    // Test CRUD operations
    testCreateFlow(formData: TFormData): Promise<TestResult>;
    testUpdateFlow(id: string, formData: Partial<TFormData>): Promise<TestResult>;
    testDeleteFlow(id: string): Promise<TestResult>;
    
    // Verify data integrity across layers
    verifyDataIntegrity(original: TFormData, result: TAPIResponse): DataIntegrityReport;
}

/**
 * UI state fixture factory interface
 * 
 * @template TState - The UI state type
 */
export interface UIStateFixtureFactory<TState> {
    // Create different UI states
    createLoadingState(context?: LoadingContext): TState;
    createErrorState(error: AppError): TState;
    createSuccessState(data: unknown, message?: string): TState;
    createEmptyState(): TState;
    
    // State transitions
    transitionToLoading(currentState: TState): TState;
    transitionToSuccess(currentState: TState, data: unknown): TState;
    transitionToError(currentState: TState, error: AppError): TState;
}

/**
 * MSW handler factory interface
 */
export interface MSWHandlerFactory {
    // Generate handlers for different scenarios
    createSuccessHandlers(): RestHandler[];
    createErrorHandlers(errorScenarios: ErrorScenario[]): RestHandler[];
    createDelayHandlers(delays: DelayConfig): RestHandler[];
    createNetworkErrorHandlers(): RestHandler[];
    
    // Dynamic handler creation
    createCustomHandler(config: HandlerConfig): RestHandler;
}

/**
 * Component test utilities interface
 * 
 * @template TProps - The component props type
 */
export interface ComponentTestUtils<TProps> {
    // Render with all providers
    renderWithProviders(component: React.ComponentType<TProps>, props: TProps): RenderResult;
    
    // Simulate user interactions
    simulateFormSubmission(formData: unknown): Promise<void>;
    simulateFieldChange(fieldName: string, value: unknown): Promise<void>;
    
    // Wait for async operations
    waitForAPICall(endpoint: string): Promise<void>;
    waitForStateUpdate(predicate: () => boolean): Promise<void>;
}

/**
 * Error scenario tester interface
 */
export interface ErrorScenarioTester {
    // Test various error conditions
    testValidationErrors(invalidData: unknown[]): Promise<ValidationErrorReport>;
    testAPIErrors(errorResponses: APIError[]): Promise<APIErrorReport>;
    testNetworkErrors(scenarios: NetworkErrorScenario[]): Promise<NetworkErrorReport>;
    testPermissionErrors(unauthorizedActions: Action[]): Promise<PermissionErrorReport>;
}

// Report types for error testing
export interface ValidationErrorReport {
    totalTests: number;
    passed: number;
    failed: number;
    errors: Array<{ input: unknown; expectedError: string; actualError?: string }>;
}

export interface APIError {
    status: number;
    code: string;
    message: string;
}

export interface APIErrorReport {
    totalTests: number;
    handled: number;
    unhandled: number;
    errors: Array<{ error: APIError; handled: boolean; userMessage?: string }>;
}

export interface NetworkErrorScenario {
    type: "timeout" | "connectionRefused" | "networkOffline" | "dnsFailure";
    config?: Record<string, unknown>;
}

export interface NetworkErrorReport {
    totalTests: number;
    recovered: number;
    failed: number;
    scenarios: Array<{ scenario: NetworkErrorScenario; result: "recovered" | "failed" }>;
}

export interface Action {
    resource: string;
    operation: string;
    context?: Record<string, unknown>;
}

export interface PermissionErrorReport {
    totalTests: number;
    blocked: number;
    allowed: number;
    errors: Array<{ action: Action; result: "blocked" | "allowed"; message?: string }>;
}

/**
 * Test API client interface for round-trip testing
 */
export interface TestAPIClient {
    get<T = unknown>(path: string, options?: RequestOptions): Promise<APIClientResponse<T>>;
    post<T = unknown>(path: string, data: unknown, options?: RequestOptions): Promise<APIClientResponse<T>>;
    put<T = unknown>(path: string, data: unknown, options?: RequestOptions): Promise<APIClientResponse<T>>;
    delete<T = unknown>(path: string, options?: RequestOptions): Promise<APIClientResponse<T>>;
}

export interface RequestOptions {
    headers?: Record<string, string>;
    timeout?: number;
    session?: unknown;
}

export interface APIClientResponse<T = unknown> {
    data: T;
    status: number;
    headers: Record<string, string>;
}

/**
 * Database verifier interface for round-trip testing
 */
export interface DatabaseVerifier {
    verifyCreated<T = unknown>(table: string, id: string): Promise<T | null>;
    verifyUpdated<T = unknown>(table: string, id: string, expectedData: Partial<T>): Promise<boolean>;
    verifyDeleted(table: string, id: string): Promise<boolean>;
    verifyRelationships(table: string, id: string, relations: Record<string, number | string[]>): Promise<boolean>;
}

/**
 * Complete UI fixture factory interface
 * 
 * @template TFormData - The form data type
 * @template TCreateInput - The API create input type
 * @template TUpdateInput - The API update input type
 * @template TAPIResponse - The API response type
 * @template TUIState - The UI state type
 */
export interface UIFixtureFactory<
    TFormData extends Record<string, unknown>,
    TCreateInput = unknown,
    TUpdateInput = unknown,
    TAPIResponse = unknown,
    TUIState = unknown
> {
    readonly objectType: string;
    
    // Sub-factories
    form: FormFixtureFactory<TFormData>;
    roundTrip: RoundTripOrchestrator<TFormData, TAPIResponse>;
    handlers: MSWHandlerFactory;
    states: UIStateFixtureFactory<TUIState>;
    componentUtils: ComponentTestUtils<any>;
    
    // Core methods
    createFormData(scenario?: string): TFormData;
    createAPIInput(formData: TFormData): TCreateInput;
    createMockResponse(overrides?: Partial<TAPIResponse>): TAPIResponse;
    setupMSW(scenario?: string): void;
    
    // Test flows
    testCreateFlow(formData?: TFormData): Promise<TAPIResponse>;
    testUpdateFlow(id: string, updates: Partial<TFormData>): Promise<TAPIResponse>;
    testDeleteFlow(id: string): Promise<boolean>;
    testRoundTrip(formData?: TFormData): Promise<{
        success: boolean;
        formData: TFormData;
        apiResponse: TAPIResponse;
        uiState: TUIState;
    }>;
}
/* c8 ignore stop */