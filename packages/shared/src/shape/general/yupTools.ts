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

export function isYupValidationError(error: any): error is ValidationError {
    return error != null && typeof error === "object" && error.name === "ValidationError";
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
