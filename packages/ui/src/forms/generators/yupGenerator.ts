import { FormSchema, YupSchema } from "../types";
import { buildYup } from 'schema-to-yup';
import { FormikConfig } from "formik";

/**
 * Generate a yup schema from a form schema. Each field in this schema 
 * contains its own yup schema, which we must combine into a single schema. 
 * Then we convert this schema into a yup object.
 * @param formSchema The schema of the entire form
 */
export const generateYupSchema = (formSchema: Pick<FormSchema, 'fields'>): FormikConfig<any>['validationSchema'] => {
    if (!formSchema) return null;
    // Create shape object to describe yup validation
    const shape: YupSchema = {
        title: 'validationSchema',
        type: 'object',
        required: [], // Name of every field that is required
        properties: {}
    }
    // Create config object for yup builder. Currently only used for error messages
    const config = { errMessages: {} }
    // Loop through each field in the form schema
    formSchema.fields.forEach(field => {
        console.log('yup loop field', field)
        const name = field.fieldName;
        if (field.yup) {
            // Add field to properties
            shape.properties[name] = { type: field.type };
            // Set up required and error messages
            config.errMessages[name] = {}
            if (field.yup.required) {
                shape.properties[name].required = true;
                shape.required.push(name);
                config.errMessages[name].required = `${field.label} is required`
                config.errMessages[name].string = `${field.label} is required`
            }
            else {
                shape.properties[name].required = false;
                shape.properties[name].nullable = true;
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
    console.log('GENERTATING YUPPPPPPP. Shape: ', shape);
    console.log('GENERTATING YUPPPPPPP. Config: ', config);
    // Build yup using newly-created shape and config
    return buildYup(shape, config)
}