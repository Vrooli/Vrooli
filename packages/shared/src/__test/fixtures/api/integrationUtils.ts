/* c8 ignore start */
/**
 * Integration utilities for connecting API fixtures with validation schemas and shape functions
 * 
 * These utilities provide seamless integration between the fixture system and the existing
 * Vrooli validation and transformation infrastructure.
 */

import type { ShapeModel } from "../../../consts/commonTypes.js";
import type { ValidationResult } from "./types.js";

/**
 * Integration with Yup validation schemas
 */
export interface YupValidationSchema<TInput> {
    validate: (input: TInput) => Promise<TInput>;
    isValid: (input: TInput) => Promise<boolean>;
    validateSync?: (input: TInput) => TInput;
    isValidSync?: (input: TInput) => boolean;
}

/**
 * Wrapper for Yup validation schemas to match our ValidationResult interface
 */
export class YupValidationAdapter<TInput> {
    constructor(private schema: YupValidationSchema<TInput>) {}
    
    async validate(input: TInput): Promise<ValidationResult> {
        try {
            const validatedData = await this.schema.validate(input);
            return {
                isValid: true,
                data: validatedData,
            };
        } catch (error: any) {
            const errors: string[] = [];
            
            if (error.inner && Array.isArray(error.inner)) {
                // Yup ValidationError with multiple errors
                errors.push(...error.inner.map((e: any) => e.message));
            } else if (error.message) {
                // Single error
                errors.push(error.message);
            } else {
                errors.push(String(error));
            }
            
            return {
                isValid: false,
                errors,
            };
        }
    }
}

/**
 * Creates validation adapters for create and update operations
 */
export function createValidationAdapters<TCreateInput, TUpdateInput>(validation: {
    create?: YupValidationSchema<TCreateInput>;
    update?: YupValidationSchema<TUpdateInput>;
}) {
    return {
        create: validation.create ? new YupValidationAdapter(validation.create) : undefined,
        update: validation.update ? new YupValidationAdapter(validation.update) : undefined,
    };
}

/**
 * Shape function integration wrapper
 * 
 * Provides type-safe access to Vrooli's shape transformation functions
 */
export class ShapeIntegration<TShape extends object, TCreateInput, TUpdateInput> {
    constructor(private shapeModel: ShapeModel<TShape, TCreateInput, TUpdateInput>) {}
    
    /**
     * Transform shape data to create input
     */
    transformToCreateInput(shapeData: TShape): TCreateInput {
        if ("create" in this.shapeModel && typeof this.shapeModel.create === "function") {
            return this.shapeModel.create(shapeData);
        }
        throw new Error("Shape model does not have a create method");
    }
    
    /**
     * Transform shape data to update input
     */
    transformToUpdateInput(original: TShape, updates: Partial<TShape>): TUpdateInput {
        if ("update" in this.shapeModel && typeof this.shapeModel.update === "function") {
            return this.shapeModel.update(original, updates);
        }
        throw new Error("Shape model does not have an update method");
    }
    
    /**
     * Create a shape-to-API transformer function
     */
    createAPITransformer() {
        return (shapeData: TShape) => this.transformToCreateInput(shapeData);
    }
    
    /**
     * Create an update transformer function
     */
    createUpdateTransformer() {
        return (original: TShape, updates: Partial<TShape>) => 
            this.transformToUpdateInput(original, updates);
    }
}

/**
 * Integration helper for both validation and shape transformation
 */
export class FullIntegration<TShape extends object, TCreateInput, TUpdateInput> {
    public readonly validation: ReturnType<typeof createValidationAdapters<TCreateInput, TUpdateInput>>;
    public readonly shape: ShapeIntegration<TShape, TCreateInput, TUpdateInput>;
    
    constructor(
        validationSchemas: {
            create?: YupValidationSchema<TCreateInput>;
            update?: YupValidationSchema<TUpdateInput>;
        },
        shapeModel: ShapeModel<TShape, TCreateInput, TUpdateInput>,
    ) {
        this.validation = createValidationAdapters(validationSchemas);
        this.shape = new ShapeIntegration(shapeModel);
    }
    
    /**
     * Complete transformation pipeline: Shape -> API Input -> Validation
     */
    async transformAndValidateCreate(shapeData: TShape): Promise<ValidationResult> {
        const apiInput = this.shape.transformToCreateInput(shapeData);
        
        if (this.validation.create) {
            return await this.validation.create.validate(apiInput);
        }
        
        return { isValid: true, data: apiInput };
    }
    
    /**
     * Complete update pipeline: Shape -> Update Input -> Validation
     */
    async transformAndValidateUpdate(original: TShape, updates: Partial<TShape>): Promise<ValidationResult> {
        const updateInput = this.shape.transformToUpdateInput(original, updates);
        
        if (this.validation.update) {
            return await this.validation.update.validate(updateInput);
        }
        
        return { isValid: true, data: updateInput };
    }
}

/**
 * Utility to create config for BaseAPIFixtureFactory with validation and shape integration
 */
export function createIntegratedFactoryConfig<TShape extends object, TCreateInput, TUpdateInput, TFindResult>(
    fixtures: {
        minimal: { create: TCreateInput; update: TUpdateInput; find: TFindResult };
        complete: { create: TCreateInput; update: TUpdateInput; find: TFindResult };
        invalid: {
            missingRequired: { create: Partial<TCreateInput>; update: Partial<TUpdateInput> };
            invalidTypes: { create: Record<string, unknown>; update: Record<string, unknown> };
            businessLogicErrors?: Record<string, Partial<TCreateInput | TUpdateInput>>;
            validationErrors?: Record<string, Partial<TCreateInput | TUpdateInput>>;
        };
        edgeCases: {
            minimalValid: { create: TCreateInput; update: TUpdateInput };
            maximalValid: { create: TCreateInput; update: TUpdateInput };
            boundaryValues?: Record<string, TCreateInput | TUpdateInput>;
            permissionScenarios?: Record<string, TCreateInput | TUpdateInput>;
        };
    },
    integration: FullIntegration<TShape, TCreateInput, TUpdateInput>,
) {
    return {
        ...fixtures,
        validationSchema: {
            create: integration.validation.create,
            update: integration.validation.update,
        },
        shapeTransforms: {
            toAPI: integration.shape.createAPITransformer(),
            fromDB: undefined, // Will be added when needed
        },
    };
}

/**
 * Utility to create a complete fixture factory with all integrations
 */
export function createIntegratedFixtureFactory<TShape, TCreateInput, TUpdateInput, TFindResult>(
    factoryClass: new (...args: any[]) => any,
    fixtures: Parameters<typeof createIntegratedFactoryConfig<TShape, TCreateInput, TUpdateInput, TFindResult>>[0],
    integration: FullIntegration<TShape, TCreateInput, TUpdateInput>,
    customizers?: any,
) {
    const config = createIntegratedFactoryConfig(fixtures, integration);
    return new factoryClass(config, customizers);
}

/**
 * Type helper to extract shape type from a shape model
 */
export type ExtractShapeType<T> = T extends ShapeModel<infer TShape, any, any> ? TShape : never;

/**
 * Type helper to extract create input type from a shape model
 */
export type ExtractShapeCreateInput<T> = T extends ShapeModel<any, infer TCreateInput, any> ? TCreateInput : never;

/**
 * Type helper to extract update input type from a shape model
 */
export type ExtractShapeUpdateInput<T> = T extends ShapeModel<any, any, infer TUpdateInput> ? TUpdateInput : never;
