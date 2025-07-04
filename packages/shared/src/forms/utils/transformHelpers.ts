/**
 * Transform Helper Functions
 * 
 * Utility functions to work with FormConfig transformations in a React-friendly way.
 */
import type { FormConfig } from "../configs/types.js";

/**
 * Creates a transform function compatible with useStandardUpsertForm from a FormConfig
 * 
 * @param formConfig - The form configuration containing transformation functions
 * @returns A transform function that can be used with useStandardUpsertForm
 */
export function createTransformFunction<
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
    TApiResult extends { __typename: string; id: string }
>(
    formConfig: FormConfig<TShape, TCreateInput, TUpdateInput, TApiResult>,
) {
    return (shape: TShape, existing: TShape, isCreate: boolean): TCreateInput | TUpdateInput | undefined => {
        if (isCreate) {
            if (!formConfig.transformations.shapeToInput.create) {
                console.warn(`Create transformation not available for ${formConfig.objectType}`);
                return undefined;
            }
            return formConfig.transformations.shapeToInput.create(shape);
        } else {
            if (!formConfig.transformations.shapeToInput.update) {
                console.warn(`Update transformation not available for ${formConfig.objectType}`);
                return undefined;
            }
            return formConfig.transformations.shapeToInput.update(existing, shape);
        }
    };
}
