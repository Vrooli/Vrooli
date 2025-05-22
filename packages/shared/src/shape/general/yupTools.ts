import { type ObjectSchema, type ValidationError } from "yup";

type YupTest = {
    OPTIONS: {
        name: string;
    }
}
type YupArrayField = {
    innerType: YupField;
    tests: YupTest[];
    type: "array";
}
type YupBooleanField = {
    tests: YupTest[];
    type: "boolean";
};

type YupDateField = {
    tests: YupTest[];
    type: "date";
};

type YupMixedField = {
    tests: YupTest[];
    type: "mixed";
};
type YupNumberField = {
    tests: YupTest[];
    type: "number";
};
type YupObjectField = {
    fields: { [key: string]: YupField; };
    tests: YupTest[];
    type: "object";
};
type YupStringField = {
    tests: YupTest[];
    type: "string";
};
type YupField = YupArrayField | YupBooleanField | YupDateField | YupMixedField | YupNumberField | YupObjectField | YupStringField;

/**
 * Finds all optional and required fields in a yup schema, in dot notation.
 * @param schema yup schema
 * @param recurseBase String to prepend to each field name 
 * @returns array of fields in dot notation, where every optional field ends with a "?"
 */
export function yupFields(schema: YupField, recurseBase = ""): string[] {
    // Determine if field is required
    const isRequired = recurseBase.length === 0 || Boolean(schema.tests.find(test => test.OPTIONS.name === "required"));
    // Prepend results with recurseBase + '.', or nothing if no recurseBase
    const prepend = recurseBase.length > 0 ? recurseBase + (isRequired ? "" : "?") + "." : "";
    // If current field is an object, recurse
    if (schema.type === "object") {
        const required: string[] = [];
        for (const [key, value] of Object.entries(schema.fields)) {
            console.log("object required recurse", key, prepend, value);
            required.push(...yupFields(value, prepend + key));
        }
        return required;
    }
    // Else if current field is an array, recurse
    else if (schema.type === "array") {
        console.log("array required recurse", prepend, schema);
        // Don't include '.'
        return yupFields(schema.innerType, recurseBase + (isRequired ? "" : "?"));
    }
    else {
        console.log("in else", recurseBase, isRequired);
        return isRequired ? [recurseBase] : [recurseBase + "?"];
    }
}

// TODO haven't checked if this works, so it probably does not
/**
 * Checks if an object contains all required fields. Supports nested objects through 
 * dot notation. Optional fields end with '?', and are included so we can check for 
 * required fields inside optional fields.
 * @param object Object to check
 * @param fields Array of fields in dot notation
 * @returns true if object contains all required fields, false otherwise
 */
export function yupObjectContainsRequiredFields(object: { [key: string]: any }, fields: string[]): boolean {
    // If object is not an object type, return false
    if (object === undefined || object === null || typeof object !== "object") return false;
    // Filter top-Level fields for required, and remove '?'
    const topLevelRequired = fields.map(field => field.split(".")[0]).filter(field => field?.endsWith("?")).map(field => field?.slice(0, -1));
    // If object does not contain all top-level required fields, return false
    if (!topLevelRequired.every(field => field && object[field] !== undefined)) return false;
    // Filter out top-level fields from fields
    const nextLevel = fields.map(field => field.split(".").slice(1).join(".")).filter(field => field.length > 0);
    // Recurse through object
    for (const [key, value] of Object.entries(object)) {
        // Find next level fields for this key
        const nextLevelFields = nextLevel.filter(field => field.startsWith(key + ".") || field.startsWith(key + "?."));
        // If there are next level fields, recurse
        if (nextLevelFields.length > 0) {
            if (!yupObjectContainsRequiredFields(value, nextLevelFields)) return false;
        }
    }
    return true;
}

/**
 * Finds all top-level fields in object that are part of the yup schema.
 * @param object Object to check
 * @param fields Array of fields in dot notation
 * @returns Object including all valid top-level fields
 */
export function grabValidTopLevelFields(object: { [key: string]: any }, fields: string[]): { [key: string]: any } {
    console.log("grabValidTopLevelFields", object, fields);
    // If object is not an object type, return empty object
    if (object === undefined || object === null || typeof object !== "object") return {};
    // Filter top-Level fields, and remove '?' if they exist
    const topLevel = fields.map(field => field.split(".")[0]).map(field => field?.endsWith("?") ? field.slice(0, -1) : field);
    // Initialize object to return
    const returnObject: { [key: string]: any } = {};
    // Loop through top-level required fields
    for (const field of topLevel) {
        if (typeof field !== "string") continue;
        // If object contains field, add to return object
        if (object[field] !== undefined) returnObject[field] = object[field];
    }
    return returnObject;
}

export function isYupValidationError(error: any): error is ValidationError {
    return error.name === "ValidationError";
}

/**
 * Validate a schema against a values object and return an object with the
 * error messages. If validation succeeds, return an empty object.
 * @param yupSchema A Yup schema
 * @param values An object of values to validate
 */
export async function validateAndGetYupErrors(
    yupSchema: ObjectSchema<any>,
    values: any,
): Promise<{} | Record<string, string>> {
    try {
        await yupSchema.validate(values);
        return {}; // Return an empty object if validation succeeds
    } catch (error) {
        if (isYupValidationError(error)) {
            // Check if the inner array is not empty
            if (error.inner.length > 0) {
                // Convert the Yup error object to a Formik-compatible error object
                return error.inner.reduce((errors, err) => {
                    if (err && err.path) {
                        errors[err.path] = err.message;
                    }
                    return errors;
                }, {});
            } else if (error.path) {
                // Handle the case when the inner array is empty
                return { [error.path]: error.message };
            }
        }
        // If it's not a Yup ValidationError, re-throw the error
        throw error;
    }
}
