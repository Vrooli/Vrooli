/**
 * Generic Shape Converters
 * 
 * Utility functions for converting between different data formats in the application:
 * - Form Data → Shape: User input to internal shape format
 * - Shape → API Input: Internal shape to API request format
 * - API Output → Shape: API response to internal shape format
 * 
 * These converters ensure type safety and consistency across the data transformation pipeline.
 */

import { type OrArray } from "../types.d.js";
import { type ListObject } from "../utils/url.js";

/**
 * Shape converter interface that all shape converters must implement
 */
export interface ShapeConverter<
    TShape extends { __typename: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>
> {
    /** Convert shape to create input for API */
    create: (shape: TShape) => TCreateInput;
    
    /** Convert shape to update input for API */
    update: (existing: TShape, updated: TShape) => TUpdateInput;
    
    /** Optional: Convert API output back to shape */
    fromApiOutput?: (output: any) => TShape;
}

/**
 * Generic function to convert form data to shape
 * @param formData - Raw form data from UI
 * @param converter - Function to transform form data to shape
 * @returns Shape object
 */
export function formToShape<TForm, TShape>(
    formData: TForm,
    converter: (data: TForm) => TShape,
): TShape {
    try {
        return converter(formData);
    } catch (error) {
        throw new Error(`Failed to convert form data to shape: ${error instanceof Error ? error.message : String(error)}`);
    }
}

/**
 * Generic function to convert shape to API input
 * @param shape - Internal shape object
 * @param shapeModel - Shape model with create/update methods
 * @param isCreate - Whether this is for create or update operation
 * @param existing - Existing shape for update operations
 * @returns API input object
 */
export function shapeToApiInput<
    TShape extends { __typename: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>
>(
    shape: TShape,
    shapeModel: ShapeConverter<TShape, TCreateInput, TUpdateInput>,
    isCreate: boolean,
    existing?: TShape | null,
): TCreateInput | TUpdateInput {
    try {
        if (isCreate) {
            return shapeModel.create(shape);
        } else {
            if (!existing) {
                throw new Error("Existing shape required for update operation");
            }
            return shapeModel.update(existing, shape);
        }
    } catch (error) {
        throw new Error(`Failed to convert shape to API input: ${error instanceof Error ? error.message : String(error)}`);
    }
}

/**
 * Generic function to convert API output to shape
 * @param apiOutput - Response from API
 * @param shapeModel - Shape model with fromApiOutput method
 * @returns Shape object
 */
export function apiOutputToShape<
    TShape extends { __typename: string },
    TApiOutput
>(
    apiOutput: TApiOutput,
    shapeModel: Pick<ShapeConverter<TShape, any, any>, "fromApiOutput">,
): TShape {
    if (!shapeModel.fromApiOutput) {
        throw new Error("Shape model does not support API output conversion");
    }
    
    try {
        return shapeModel.fromApiOutput(apiOutput);
    } catch (error) {
        throw new Error(`Failed to convert API output to shape: ${error instanceof Error ? error.message : String(error)}`);
    }
}

/**
 * Batch converter for arrays of shapes
 */
export function batchShapeToApiInput<
    TShape extends { __typename: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>
>(
    shapes: TShape[],
    shapeModel: ShapeConverter<TShape, TCreateInput, TUpdateInput>,
    isCreate: boolean,
    existingShapes?: TShape[],
): Array<TCreateInput | TUpdateInput> {
    return shapes.map((shape, index) => {
        const existing = existingShapes?.[index] || null;
        return shapeToApiInput(shape, shapeModel, isCreate, existing);
    });
}

/**
 * Type-safe converter factory
 */
export function createConverter<
    TForm extends Record<string, any>,
    TShape extends { __typename: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>
>(config: {
    formToShape: (form: TForm) => TShape;
    shapeModel: ShapeConverter<TShape, TCreateInput, TUpdateInput>;
}) {
    return {
        /**
         * Convert form data directly to API input
         */
        formToApiInput(
            formData: TForm,
            isCreate: boolean,
            existing?: TShape,
        ): TCreateInput | TUpdateInput {
            const shape = formToShape(formData, config.formToShape);
            return shapeToApiInput(shape, config.shapeModel, isCreate, existing);
        },
        
        /**
         * Get shape from form data
         */
        getShape(formData: TForm): TShape {
            return formToShape(formData, config.formToShape);
        },
        
        /**
         * Get API input from shape
         */
        getApiInput(
            shape: TShape,
            isCreate: boolean,
            existing?: TShape,
        ): TCreateInput | TUpdateInput {
            return shapeToApiInput(shape, config.shapeModel, isCreate, existing);
        },
        
        /**
         * Validate transformation pipeline
         */
        async validatePipeline(
            formData: TForm,
            validator?: (input: TCreateInput | TUpdateInput) => Promise<boolean>,
        ): Promise<{
            valid: boolean;
            shape: TShape;
            apiInput: TCreateInput | TUpdateInput;
            errors?: string[];
        }> {
            try {
                const shape = config.formToShape(formData);
                const apiInput = config.shapeModel.create(shape);
                
                let valid = true;
                let errors: string[] = [];
                
                if (validator) {
                    try {
                        valid = await validator(apiInput);
                    } catch (error) {
                        valid = false;
                        errors = [error instanceof Error ? error.message : String(error)];
                    }
                }
                
                return { valid, shape, apiInput, errors };
            } catch (error) {
                return {
                    valid: false,
                    shape: null as any,
                    apiInput: null as any,
                    errors: [error instanceof Error ? error.message : String(error)],
                };
            }
        },
    };
}

/**
 * Helper to check if a shape converter supports bidirectional conversion
 */
export function isBidirectionalShapeConverter<T extends ShapeConverter<any, any, any>>(
    converter: T,
): converter is T & Required<Pick<T, "fromApiOutput">> {
    return typeof converter.fromApiOutput === "function";
}

/**
 * Helper to create a shape converter from individual converter functions
 */
export function createShapeConverter<
    TShape extends { __typename: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>
>(
    converters: ShapeConverter<TShape, TCreateInput, TUpdateInput>,
): ShapeConverter<TShape, TCreateInput, TUpdateInput> {
    return converters;
}

/**
 * Example usage:
 * 
 * import { shapeComment } from "./models/models.js";
 * import { commentFormToShape } from "../ui/forms/comment.js";
 * 
 * const commentConverter = createConverter({
 *     formToShape: commentFormToShape,
 *     shapeModel: shapeComment,
 * });
 * 
 * // Use in form submission
 * const apiInput = commentConverter.formToApiInput(formData, true);
 * 
 * // Validate before submission
 * const validation = await commentConverter.validatePipeline(formData, async (input) => {
 *     const schema = commentValidation.create({ env: "test" });
 *     await schema.validate(input);
 *     return true;
 * });
 */
