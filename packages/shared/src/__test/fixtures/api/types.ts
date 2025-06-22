/* c8 ignore start */
/**
 * Core types for the API fixture factory system
 * 
 * This module defines the type-safe interfaces that all API fixtures must implement.
 * It ensures zero `any` types and provides integration with validation schemas and shape functions.
 */

/**
 * Result of validation performed on fixture data
 */
export interface ValidationResult {
    isValid: boolean;
    errors?: string[];
    data?: unknown;
}

/**
 * Configuration for relationship building in fixtures
 */
export interface RelationshipConfig {
    [relationName: string]: unknown[] | unknown | null;
}

/**
 * Configuration for permission scenarios in fixtures
 */
export interface PermissionConfig {
    role?: string;
    permissions?: string[];
    context?: Record<string, unknown>;
}

/**
 * Core API fixture factory interface
 * 
 * This interface ensures type safety and consistency across all API fixtures.
 * Every fixture must implement this interface with proper generic types.
 * 
 * @template TCreateInput - The API input type for create operations
 * @template TUpdateInput - The API input type for update operations  
 * @template TFindResult - The API response type for find operations
 * @template TFormData - The UI form data type (optional, for transformation)
 * @template TDatabaseType - The database record type (optional, for transformation)
 */
export interface APIFixtureFactory<
    TCreateInput = unknown,
    TUpdateInput = unknown,
    TFindResult = unknown,
    TFormData = unknown,
    TDatabaseType = unknown
> {
    // Core fixture sets - required for all factories
    minimal: {
        create: TCreateInput;
        update: TUpdateInput;
        find: TFindResult;
    };
    
    complete: {
        create: TCreateInput;
        update: TUpdateInput;
        find: TFindResult;
    };
    
    // Error scenarios - comprehensive error coverage
    invalid: {
        missingRequired: { 
            create: Partial<TCreateInput>; 
            update: Partial<TUpdateInput>; 
        };
        invalidTypes: { 
            create: Record<string, unknown>; 
            update: Record<string, unknown>; 
        };
        businessLogicErrors?: Record<string, Partial<TCreateInput | TUpdateInput>>;
        validationErrors?: Record<string, Partial<TCreateInput | TUpdateInput>>;
    };
    
    // Edge cases - boundary conditions and special scenarios
    edgeCases: {
        minimalValid: { 
            create: TCreateInput; 
            update: TUpdateInput; 
        };
        maximalValid: { 
            create: TCreateInput; 
            update: TUpdateInput; 
        };
        boundaryValues?: Record<string, TCreateInput | TUpdateInput>;
        permissionScenarios?: Record<string, TCreateInput | TUpdateInput>;
    };
    
    // Factory methods - dynamic data generation with type safety
    createFactory: (overrides?: Partial<TCreateInput>) => TCreateInput;
    updateFactory: (id: string, overrides?: Partial<TUpdateInput>) => TUpdateInput;
    findFactory: (overrides?: Partial<TFindResult>) => TFindResult;
    
    // Validation integration - connect with real validation schemas
    validateCreate: (input: TCreateInput) => Promise<ValidationResult>;
    validateUpdate: (input: TUpdateInput) => Promise<ValidationResult>;
    
    // Shape integration - transform between different representations
    transformToAPI?: (formData: TFormData) => TCreateInput;
    transformFromDB?: (dbRecord: TDatabaseType) => TFindResult;
    
    // Relationship helpers - build complex object graphs
    withRelationships: (base: TFindResult, relations: RelationshipConfig) => TFindResult;
    withPermissions: (base: TFindResult, permissions: PermissionConfig) => TFindResult;
}

/**
 * Base configuration for creating an API fixture factory
 */
export interface APIFixtureFactoryConfig<TCreateInput, TUpdateInput, TFindResult> {
    // Required: Core fixture data
    minimal: {
        create: TCreateInput;
        update: TUpdateInput;
        find: TFindResult;
    };
    complete: {
        create: TCreateInput;
        update: TUpdateInput;
        find: TFindResult;
    };
    invalid: {
        missingRequired: { 
            create: Partial<TCreateInput>; 
            update: Partial<TUpdateInput>; 
        };
        invalidTypes: { 
            create: Record<string, unknown>; 
            update: Record<string, unknown>; 
        };
        businessLogicErrors?: Record<string, Partial<TCreateInput | TUpdateInput>>;
        validationErrors?: Record<string, Partial<TCreateInput | TUpdateInput>>;
    };
    edgeCases: {
        minimalValid: { 
            create: TCreateInput; 
            update: TUpdateInput; 
        };
        maximalValid: { 
            create: TCreateInput; 
            update: TUpdateInput; 
        };
        boundaryValues?: Record<string, TCreateInput | TUpdateInput>;
        permissionScenarios?: Record<string, TCreateInput | TUpdateInput>;
    };
    
    // Optional: Integration functions
    validationSchema?: {
        create?: {
            validate: (input: TCreateInput) => Promise<ValidationResult>;
        };
        update?: {
            validate: (input: TUpdateInput) => Promise<ValidationResult>;
        };
    };
    
    shapeTransforms?: {
        toAPI?: (formData: unknown) => TCreateInput;
        fromDB?: (dbRecord: unknown) => TFindResult;
    };
}

/**
 * Factory customizer functions for dynamic data generation
 */
export interface FactoryCustomizers<TCreateInput, TUpdateInput> {
    create: (base: TCreateInput, overrides?: Partial<TCreateInput>) => TCreateInput;
    update: (base: TUpdateInput, overrides?: Partial<TUpdateInput>) => TUpdateInput;
}

/**
 * Enhanced factory interface that includes test utilities
 */
export interface EnhancedAPIFixtureFactory<TCreateInput, TUpdateInput, TFindResult> 
    extends APIFixtureFactory<TCreateInput, TUpdateInput, TFindResult> {
    
    // Test utilities
    generateUnique: {
        create: () => TCreateInput;
        update: (id: string) => TUpdateInput;
    };
    
    // Validation testing
    testValidation: {
        expectValid: (input: TCreateInput | TUpdateInput) => Promise<void>;
        expectInvalid: (input: Partial<TCreateInput | TUpdateInput>, expectedErrors?: string[]) => Promise<void>;
    };
    
    // Round-trip testing support
    roundTrip?: {
        executeCreateCycle: (createInput: TCreateInput) => Promise<TFindResult>;
        executeUpdateCycle: (updateInput: TUpdateInput) => Promise<TFindResult>;
        verifyDataIntegrity: (original: TCreateInput, result: TFindResult) => boolean;
    };
}

/**
 * Utility type to extract create input type from an API fixture factory
 */
export type ExtractCreateInput<T> = T extends APIFixtureFactory<infer TCreateInput, any, any> 
    ? TCreateInput 
    : never;

/**
 * Utility type to extract update input type from an API fixture factory
 */
export type ExtractUpdateInput<T> = T extends APIFixtureFactory<any, infer TUpdateInput, any> 
    ? TUpdateInput 
    : never;

/**
 * Utility type to extract find result type from an API fixture factory
 */
export type ExtractFindResult<T> = T extends APIFixtureFactory<any, any, infer TFindResult> 
    ? TFindResult 
    : never;
