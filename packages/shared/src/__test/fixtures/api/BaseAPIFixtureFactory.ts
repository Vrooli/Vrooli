/* c8 ignore start */
/**
 * Base implementation for API fixture factories
 * 
 * This class provides a concrete implementation of the APIFixtureFactory interface
 * with sensible defaults and integration points for validation and shape functions.
 */

import { generatePK } from "../../../id/snowflake.js";
import type { 
    APIFixtureFactory, 
    APIFixtureFactoryConfig, 
    FactoryCustomizers, 
    PermissionConfig, 
    RelationshipConfig, 
    ValidationResult 
} from "./types.js";

/**
 * Base API fixture factory implementation
 * 
 * Provides a type-safe foundation for all API fixtures with integration points
 * for validation schemas, shape functions, and relationship building.
 */
export class BaseAPIFixtureFactory<
    TCreateInput = unknown,
    TUpdateInput = unknown,
    TFindResult = unknown,
    TFormData = unknown,
    TDatabaseType = unknown
> implements APIFixtureFactory<TCreateInput, TUpdateInput, TFindResult, TFormData, TDatabaseType> {
    
    protected config: APIFixtureFactoryConfig<TCreateInput, TUpdateInput, TFindResult>;
    protected customizers?: FactoryCustomizers<TCreateInput, TUpdateInput>;
    
    constructor(
        config: APIFixtureFactoryConfig<TCreateInput, TUpdateInput, TFindResult>,
        customizers?: FactoryCustomizers<TCreateInput, TUpdateInput>
    ) {
        this.config = config;
        this.customizers = customizers;
    }
    
    // Core fixture sets - delegate to config
    get minimal() {
        return this.config.minimal;
    }
    
    get complete() {
        return this.config.complete;
    }
    
    get invalid() {
        return this.config.invalid;
    }
    
    get edgeCases() {
        return this.config.edgeCases;
    }
    
    // Factory methods with type safety
    createFactory = (overrides?: Partial<TCreateInput>): TCreateInput => {
        const base = { ...this.config.complete.create };
        const withOverrides = { ...base, ...overrides };
        
        if (this.customizers) {
            return this.customizers.create(withOverrides, overrides);
        }
        
        return withOverrides;
    };
    
    updateFactory = (id: string, overrides?: Partial<TUpdateInput>): TUpdateInput => {
        const base = { ...this.config.complete.update };
        const withId = { ...base, id } as TUpdateInput;
        const withOverrides = { ...withId, ...overrides };
        
        if (this.customizers) {
            return this.customizers.update(withOverrides, overrides);
        }
        
        return withOverrides;
    };
    
    findFactory = (overrides?: Partial<TFindResult>): TFindResult => {
        const base = { ...this.config.complete.find };
        return { ...base, ...overrides };
    };
    
    // Validation integration
    validateCreate = async (input: TCreateInput): Promise<ValidationResult> => {
        if (!this.config.validationSchema?.create?.validate) {
            // Return success if no validation schema provided
            return { isValid: true, data: input };
        }
        
        try {
            return await this.config.validationSchema.create.validate(input);
        } catch (error) {
            return {
                isValid: false,
                errors: [error instanceof Error ? error.message : String(error)]
            };
        }
    };
    
    validateUpdate = async (input: TUpdateInput): Promise<ValidationResult> => {
        if (!this.config.validationSchema?.update?.validate) {
            // Return success if no validation schema provided
            return { isValid: true, data: input };
        }
        
        try {
            return await this.config.validationSchema.update.validate(input);
        } catch (error) {
            return {
                isValid: false,
                errors: [error instanceof Error ? error.message : String(error)]
            };
        }
    };
    
    // Shape integration (optional)
    transformToAPI = (formData: TFormData): TCreateInput => {
        if (!this.config.shapeTransforms?.toAPI) {
            throw new Error("transformToAPI not implemented for this fixture");
        }
        
        return this.config.shapeTransforms.toAPI(formData);
    };
    
    transformFromDB = (dbRecord: TDatabaseType): TFindResult => {
        if (!this.config.shapeTransforms?.fromDB) {
            throw new Error("transformFromDB not implemented for this fixture");
        }
        
        return this.config.shapeTransforms.fromDB(dbRecord);
    };
    
    // Relationship helpers
    withRelationships = (base: TFindResult, relations: RelationshipConfig): TFindResult => {
        // Default implementation: merge relations into the base object
        // Subclasses can override for more sophisticated relationship handling
        return { 
            ...base, 
            ...relations 
        } as TFindResult;
    };
    
    withPermissions = (base: TFindResult, permissions: PermissionConfig): TFindResult => {
        // Default implementation: add permissions as metadata
        // Subclasses can override for object-specific permission handling
        return {
            ...base,
            __permissions: permissions
        } as TFindResult;
    };
    
    // Utility methods for subclasses
    protected generateId(): string {
        return generatePK().toString();
    }
    
    protected mergeWithBase<T>(base: T, overrides?: Partial<T>): T {
        return { ...base, ...overrides };
    }
    
    // Test helper methods
    generateUniqueCreate = (): TCreateInput => {
        return this.createFactory({ id: this.generateId() } as any);
    };
    
    generateUniqueUpdate = (id?: string): TUpdateInput => {
        return this.updateFactory(id || this.generateId());
    };
    
    // Validation test helpers
    expectValid = async (input: TCreateInput | TUpdateInput): Promise<void> => {
        const isCreate = this.isCreateInput(input);
        const result = isCreate 
            ? await this.validateCreate(input as TCreateInput)
            : await this.validateUpdate(input as TUpdateInput);
            
        if (!result.isValid) {
            throw new Error(`Expected input to be valid, but got errors: ${result.errors?.join(", ")}`);
        }
    };
    
    expectInvalid = async (input: Partial<TCreateInput | TUpdateInput>, expectedErrors?: string[]): Promise<void> => {
        const isCreate = this.isCreateInput(input);
        const result = isCreate 
            ? await this.validateCreate(input as TCreateInput)
            : await this.validateUpdate(input as TUpdateInput);
            
        if (result.isValid) {
            throw new Error("Expected input to be invalid, but validation passed");
        }
        
        if (expectedErrors && result.errors) {
            for (const expectedError of expectedErrors) {
                if (!result.errors.some(error => error.includes(expectedError))) {
                    throw new Error(`Expected error containing "${expectedError}", but got: ${result.errors.join(", ")}`);
                }
            }
        }
    };
    
    // Type guard helper
    private isCreateInput(input: Partial<TCreateInput | TUpdateInput>): boolean {
        // This is a simple heuristic - subclasses can override for better detection
        // Generally, create inputs don't have an id field (it's generated)
        // while update inputs require an id field
        return !("id" in input && input.id);
    }
}

/**
 * Enhanced base factory with additional test utilities
 */
export class EnhancedAPIFixtureFactory<
    TCreateInput = unknown,
    TUpdateInput = unknown,
    TFindResult = unknown,
    TFormData = unknown,
    TDatabaseType = unknown
> extends BaseAPIFixtureFactory<TCreateInput, TUpdateInput, TFindResult, TFormData, TDatabaseType> {
    
    // Test utilities
    generateUnique = {
        create: () => this.generateUniqueCreate(),
        update: (id: string) => this.generateUniqueUpdate(id)
    };
    
    testValidation = {
        expectValid: this.expectValid,
        expectInvalid: this.expectInvalid
    };
    
    // Round-trip testing support (optional - requires implementation by subclasses)
    roundTrip?: {
        executeCreateCycle: (createInput: TCreateInput) => Promise<TFindResult>;
        executeUpdateCycle: (updateInput: TUpdateInput) => Promise<TFindResult>;
        verifyDataIntegrity: (original: TCreateInput, result: TFindResult) => boolean;
    };
}