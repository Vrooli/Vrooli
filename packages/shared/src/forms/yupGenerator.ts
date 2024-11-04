import * as Yup from "yup";
import { InputType } from "../consts/model";
import { FormInputBase, FormSchema, YupType } from "./types";

/**
 * Maps input types to their yup types. 
 * If an input type is not in this object, then 
 * it cannot be validated with yup.
 */
export const InputToYupType: { [key in InputType]?: YupType } = {
    // [InputType.Checkbox]: 'array', //TODO
    [InputType.JSON]: "string",
    [InputType.IntegerInput]: "number",
    [InputType.Radio]: "string",
    [InputType.Selector]: "string",
    [InputType.Slider]: "number",
    [InputType.Switch]: "boolean",
    [InputType.Text]: "string",
};

/**
 * Generate a yup schema from a form schema. Each field in this schema 
 * contains its own yup schema, which we must combine into a single schema. 
 * Then we convert this schema into a yup object.
 * @param formSchema The schema of the entire form
 */
export function generateYupSchema(
    formSchema: Pick<FormSchema, "elements">,
    prefix?: string,
) {
    if (!formSchema) return null;

    // Initialize an empty object to hold the field schemas
    const shape: { [fieldName: string]: Yup.AnySchema } = {};

    // Loop through each field in the form schema
    formSchema.elements.forEach(field => {
        // Skip non-input fields
        if (!("fieldName" in field)) return;
        const formInput = field as FormInputBase;
        const name = prefix ? `${prefix}-${formInput.fieldName}` : formInput.fieldName;

        // Field will only be validated if it has a yup schema, and it is a valid input type
        if (formInput.yup && InputToYupType[field.type]) {
            // Start building the Yup schema for this field
            let validator: Yup.AnySchema;

            // Determine the base Yup type based on InputToYupType
            const baseType = InputToYupType[field.type];

            if (!baseType) return; // Skip if type is not supported

            switch (baseType) {
                case "string":
                    validator = Yup.string();
                    break;
                case "number":
                    validator = Yup.number();
                    break;
                case "boolean":
                    validator = Yup.boolean();
                    break;
                case "date":
                    validator = Yup.date();
                    break;
                case "object":
                    validator = Yup.object();
                    break;
                default:
                    validator = Yup.mixed();
                    break;
            }

            // Apply required
            if (formInput.yup.required) {
                validator = validator.required(`${field.label} is required`);
            } else {
                validator = validator.nullable();
            }

            // Apply additional checks
            formInput.yup.checks.forEach(check => {
                const { key, value, error } = check;
                const method = (validator as any)[key];
                if (typeof method === "function") {
                    if (value !== undefined && error !== undefined) {
                        validator = method.call(validator, value, error);
                    } else if (value !== undefined) {
                        validator = method.call(validator, value);
                    } else if (error !== undefined) {
                        validator = method.call(validator, error);
                    } else {
                        validator = method.call(validator);
                    }
                } else {
                    throw new Error(`Validation method ${key} does not exist on Yup.${baseType}()`);
                }
            });

            // Add the validator to the shape
            shape[name] = validator;
        }
    });

    // Build the Yup object schema
    const yupSchema = Yup.object().shape(shape);

    return yupSchema;
}
