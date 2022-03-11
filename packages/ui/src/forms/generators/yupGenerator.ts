import { FormSchema, YupSchema } from "../types";
import { buildYup } from 'schema-to-yup';

/**
 * Generate a yup schema from a form schema
 */
export const generateYupSchema = (formSchema: FormSchema): any => {
    if (!formSchema) return null;
    // Create shape object to describe yup validation
    const shape: YupSchema = {
        title: 'validationSchema',
        type: 'object',
        required: [],
        properties: {}
    }
    // Create config object for yup builder. Currently only used for error messages
    const config = { errMessages: {} }
    // Loop through each field in the form schema
    formSchema.fields.forEach(field => {
        const name = field.fieldName;
        // Add field to properties, even if no validation is required
        shape.properties[name] = { type: field.type, nullable: true };
        if (field.yup) {
            // Set up required error message
            config.errMessages[name] = {}
            if (field.yup.required) {
                shape.properties[name].nullable = false;
                shape.required.push(name);
                config.errMessages[name].required = `${field.label} is required`
                config.errMessages[name].string = `${field.label} is required`
            }
            // Add additional validation checks
            if (Array.isArray(field.yup.checks)) {
                field.yup.checks.forEach(check => {
                    shape.properties[name][check.key] = check.value
                    if (check.error) { config.errMessages[name][check.key] = check.error }
                })
            }
        }
    })
    // Build yup using newly-created shape and config
    return buildYup(shape, config)
}