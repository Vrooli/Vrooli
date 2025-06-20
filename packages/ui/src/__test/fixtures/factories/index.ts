/* c8 ignore start */
/**
 * Central factory system for UI fixtures
 * 
 * This module provides the main entry point for creating type-safe fixtures
 * for all Vrooli object types with full integration to @vrooli/shared.
 */

import type {
    FormFixtureFactory,
    RoundTripOrchestrator,
    MSWHandlerFactory,
    UIStateFixtureFactory,
    ComponentTestUtils
} from "../types.js";

/**
 * Complete fixture factory for a Vrooli object type
 * 
 * Each object type should have one of these factories that provides
 * all necessary fixtures and utilities for comprehensive testing.
 * 
 * @template TFormData - The form data type
 * @template TCreateInput - The API create input type
 * @template TUpdateInput - The API update input type
 * @template TFindResult - The API response type
 * @template TUIState - The UI state type
 */
export interface UIFixtureFactory<
    TFormData extends Record<string, unknown>,
    TCreateInput = unknown,
    TUpdateInput = unknown,
    TFindResult = unknown,
    TUIState = unknown
> {
    // Object type identifier
    readonly objectType: string;
    
    // Form fixture factory
    form: FormFixtureFactory<TFormData>;
    
    // Round-trip testing orchestrator
    roundTrip: RoundTripOrchestrator<TFormData, TFindResult>;
    
    // MSW handler factory
    handlers: MSWHandlerFactory;
    
    // UI state factory
    states: UIStateFixtureFactory<TUIState>;
    
    // Component test utilities
    componentUtils: ComponentTestUtils<any>;
    
    // Quick access methods
    createFormData(scenario?: "minimal" | "complete" | string): TFormData;
    createAPIInput(formData: TFormData): TCreateInput;
    createMockResponse(overrides?: Partial<TFindResult>): TFindResult;
    setupMSW(scenario?: "success" | "error" | "loading"): void;
    
    // Test execution helpers
    testCreateFlow(formData?: TFormData): Promise<TFindResult>;
    testUpdateFlow(id: string, updates: Partial<TFormData>): Promise<TFindResult>;
    testDeleteFlow(id: string): Promise<boolean>;
    testRoundTrip(formData?: TFormData): Promise<{
        success: boolean;
        formData: TFormData;
        apiResponse: TFindResult;
        uiState: TUIState;
    }>;
}

/**
 * Factory registry for all object types
 */
export interface UIFixtureRegistry {
    // Core objects
    user: UIFixtureFactory<any, any, any, any, any>;
    team: UIFixtureFactory<any, any, any, any, any>;
    project: UIFixtureFactory<any, any, any, any, any>;
    routine: UIFixtureFactory<any, any, any, any, any>;
    
    // Communication objects
    chat: UIFixtureFactory<any, any, any, any, any>;
    comment: UIFixtureFactory<any, any, any, any, any>;
    issue: UIFixtureFactory<any, any, any, any, any>;
    meeting: UIFixtureFactory<any, any, any, any, any>;
    
    // Resource objects
    resource: UIFixtureFactory<any, any, any, any, any>;
    bookmark: UIFixtureFactory<any, any, any, any, any>;
    tag: UIFixtureFactory<any, any, any, any, any>;
    
    // Extended object list (41+ total)
    api: UIFixtureFactory<any, any, any, any, any>;
    bot: UIFixtureFactory<any, any, any, any, any>;
    bookmarkList: UIFixtureFactory<any, any, any, any, any>;
    chatInvite: UIFixtureFactory<any, any, any, any, any>;
    chatMessage: UIFixtureFactory<any, any, any, any, any>;
    chatParticipant: UIFixtureFactory<any, any, any, any, any>;
    member: UIFixtureFactory<any, any, any, any, any>;
    memberInvite: UIFixtureFactory<any, any, any, any, any>;
    notification: UIFixtureFactory<any, any, any, any, any>;
    pullRequest: UIFixtureFactory<any, any, any, any, any>;
    reminder: UIFixtureFactory<any, any, any, any, any>;
    reminderItem: UIFixtureFactory<any, any, any, any, any>;
    reminderList: UIFixtureFactory<any, any, any, any, any>;
    report: UIFixtureFactory<any, any, any, any, any>;
    reportResponse: UIFixtureFactory<any, any, any, any, any>;
    resourceVersion: UIFixtureFactory<any, any, any, any, any>;
    run: UIFixtureFactory<any, any, any, any, any>;
    runStep: UIFixtureFactory<any, any, any, any, any>;
    schedule: UIFixtureFactory<any, any, any, any, any>;
    scheduleException: UIFixtureFactory<any, any, any, any, any>;
    scheduleRecurrence: UIFixtureFactory<any, any, any, any, any>;
    transfer: UIFixtureFactory<any, any, any, any, any>;
    wallet: UIFixtureFactory<any, any, any, any, any>;
    
    // Add remaining objects as they are implemented...
}

/**
 * Configuration for creating a UI fixture factory
 */
export interface UIFixtureFactoryConfig<
    TFormData extends Record<string, unknown>,
    TCreateInput,
    TUpdateInput,
    TFindResult,
    TUIState
> {
    objectType: string;
    form: FormFixtureFactory<TFormData>;
    endpoints: {
        create: string;
        update: string;
        delete: string;
        find: string;
        list?: string;
    };
    shapeTransforms: {
        toAPI: (formData: TFormData) => TCreateInput;
        fromAPI: (apiResponse: TFindResult) => TUIState;
    };
    mockResponses: {
        minimal: TFindResult;
        complete: TFindResult;
        list?: TFindResult[];
    };
}

/**
 * Create a UI fixture factory for an object type
 */
export function createUIFixtureFactory<
    TFormData extends Record<string, unknown>,
    TCreateInput,
    TUpdateInput,
    TFindResult extends { id: string },
    TUIState
>(
    config: UIFixtureFactoryConfig<TFormData, TCreateInput, TUpdateInput, TFindResult, TUIState>
): UIFixtureFactory<TFormData, TCreateInput, TUpdateInput, TFindResult, TUIState> {
    // This is a placeholder - actual implementation would create all sub-factories
    // and wire them together. For now, we'll define the interface structure.
    
    throw new Error("createUIFixtureFactory implementation pending - use specific factories");
}

/**
 * Global fixture registry instance
 * 
 * This will be populated by individual fixture implementations
 * and provides centralized access to all fixtures.
 */
export const uiFixtures: Partial<UIFixtureRegistry> = {};

/**
 * Register a fixture factory in the global registry
 */
export function registerFixture<K extends keyof UIFixtureRegistry>(
    objectType: K,
    factory: UIFixtureRegistry[K]
): void {
    uiFixtures[objectType] = factory;
}

/**
 * Get a fixture factory from the registry
 */
export function getFixture<K extends keyof UIFixtureRegistry>(
    objectType: K
): UIFixtureRegistry[K] {
    const fixture = uiFixtures[objectType];
    if (!fixture) {
        throw new Error(`No fixture registered for object type: ${objectType}`);
    }
    return fixture;
}

/**
 * Batch fixture operations for performance
 */
export async function batchFixtureOperations<T>(
    operations: Array<() => Promise<T>>
): Promise<T[]> {
    return Promise.all(operations.map(op => op()));
}

/**
 * Test scenario builder for complex multi-object scenarios
 */
export class TestScenarioBuilder {
    private fixtures: Array<{
        type: keyof UIFixtureRegistry;
        scenario: string;
        data?: Record<string, unknown>;
    }> = [];
    
    add<K extends keyof UIFixtureRegistry>(
        type: K,
        scenario: string = "minimal",
        overrides?: Record<string, unknown>
    ): this {
        this.fixtures.push({ type, scenario, data: overrides });
        return this;
    }
    
    async build(): Promise<Record<string, unknown>> {
        const results: Record<string, unknown> = {};
        
        for (const fixture of this.fixtures) {
            const factory = getFixture(fixture.type);
            const data = factory.createFormData(fixture.scenario);
            
            if (fixture.data) {
                Object.assign(data, fixture.data);
            }
            
            results[fixture.type] = await factory.testCreateFlow(data);
        }
        
        return results;
    }
}

/**
 * Utility to create test data with relationships
 */
export async function createWithRelationships(
    primaryType: keyof UIFixtureRegistry,
    relationships: Array<{
        type: keyof UIFixtureRegistry;
        count?: number;
        scenario?: string;
    }>
): Promise<{
    primary: unknown;
    related: Record<string, unknown[]>;
}> {
    // Create primary object
    const primaryFactory = getFixture(primaryType);
    const primaryData = primaryFactory.createFormData("complete");
    const primary = await primaryFactory.testCreateFlow(primaryData);
    
    // Create related objects
    const related: Record<string, unknown[]> = {};
    
    for (const rel of relationships) {
        const factory = getFixture(rel.type);
        const items: unknown[] = [];
        
        const count = rel.count || 1;
        for (let i = 0; i < count; i++) {
            const data = factory.createFormData(rel.scenario || "minimal");
            const item = await factory.testCreateFlow(data);
            items.push(item);
        }
        
        related[rel.type] = items;
    }
    
    return { primary, related };
}
/* c8 ignore stop */