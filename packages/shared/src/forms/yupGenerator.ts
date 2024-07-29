import { InputType } from "@local/shared";
import { buildYup } from "schema-to-yup";
import { FormInputBase, FormSchema, YupSchema, YupType } from "./types";

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
) {
    if (!formSchema) return null;
    // Create shape object to describe yup validation
    const shape: YupSchema = {
        title: "validationSchema",
        type: "object",
        required: [], // Name of every field that is required
        properties: {},
    };
    // Create config object for yup builder. Currently only used for error messages
    const config = { errMessages: {} };
    // Loop through each field in the form schema
    formSchema.elements.forEach(field => {
        // Skip non-input fields
        if (!Object.prototype.hasOwnProperty.call(field, "fieldName")) return;
        const formInput = field as FormInputBase;
        const name = formInput.fieldName;
        // Field will only be validated if it has a yup schema, and it is a valid input type
        if (formInput.yup && InputToYupType[field.type]) {
            // Add field to properties
            shape.properties[name] = { type: InputToYupType[field.type] };
            // Set up required and error messages
            config.errMessages[name] = {};
            if (formInput.yup.required) {
                shape.properties[name].required = true;
                shape.required.push(name);
                config.errMessages[name].required = `${field.label} is required`;
                config.errMessages[name].string = `${field.label} is required`;
            }
            else {
                shape.properties[name].required = false;
                shape.properties[name].nullable = true;
            }
            // Add additional validation checks
            if (Array.isArray(formInput.yup.checks)) {
                formInput.yup.checks.forEach(check => {
                    shape.properties[name][check.key] = check.value;
                    if (check.error) { config.errMessages[name][check.key] = check.error; }
                });
            }
        }
    });
    // Build yup using newly-created shape and config
    const yup = buildYup(shape, config);
    return yup;
    // return buildYup({
    //     $schema: "http://json-schema.org/draft-07/schema#",
    //     $id: "http://example.com/person.schema.json",
    //     title: "Person",
    //     description: "A person",
    //     type: "object",
    //     properties: {
    //         name: {
    //             description: "Name of the person",
    //             type: "string",
    //         },
    //         email: {
    //             type: "string",
    //             format: "email",
    //         },
    //         foobar: {
    //             type: "string",
    //             matches: "(foo|bar)",
    //         },
    //         age: {
    //             description: "Age of person",
    //             type: "number",
    //             exclusiveMinimum: 0,
    //             required: true,
    //         },
    //         characterType: {
    //             enum: ["good", "bad"],
    //             enum_titles: ["Good", "Bad"],
    //             type: "string",
    //             title: "Type of people",
    //             propertyOrder: 3,
    //         },
    //     },
    //     required: ["name", "email"],
    // }, {
    //     errMessages: {
    //         age: {
    //             required: "A person must have an age",
    //         },
    //         email: {
    //             required: "You must enter an email address",
    //             format: "Not a valid email address",
    //         },
    //     },
    // })
}
