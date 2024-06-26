import { InputType, OrArray, ParseSearchParamsResult, Tag, UrlPrimitive } from "@local/shared";
import { FormSchema } from "forms/types";

/**
 * Prepares an array of form items to be encoded into a URL search parameter.
 * @param array The array of items
 * @param convert The function to convert each item to a search URL value
 * @returns The items of the array converted, or undefined if the array is empty or undefined
 */
export function arrayToSearch(
    array: unknown,
    convert: (item: unknown) => UrlPrimitive | undefined,
): UrlPrimitive[] | undefined {
    if (!Array.isArray(array)) return undefined;
    const shaped = array.map(convert);
    // Filter out any values that weren't converted
    const filtered = shaped.filter((value) => value !== undefined) as UrlPrimitive[];
    // Return undefined if the array is empty
    return filtered.length > 0 ? filtered : undefined;
}

/**
 * Converts a valued parsed from the URL into an array of form items.
 * @param value The value parsed from the URL
 * @param convert The function to convert each search URL value to a form item
 * @returns The items of the array converted, or undefined if the value is undefined (we don't check for empty array)
 */
export function searchToArray(
    value: OrArray<UrlPrimitive>,
    convert: (value: UrlPrimitive) => unknown,
): unknown {
    if (!Array.isArray(value)) return undefined;
    const shaped = value.map(convert);
    // Filter out any values that weren't converted
    const filtered = shaped.filter((value) => value !== undefined);
    return filtered;
}

/**
 * Ensures that the value is a non-zero number
 * @param value The value to check
 * @returns The value if it is a non-zero number, or undefined otherwise
 */
export function nonZeroNumber(value: unknown): number | undefined {
    const num = typeof value === "number" ? value : parseFloat(value as string);
    return num !== undefined && Number.isFinite(num) && num !== 0 ? num : undefined;
}

/**
 * Ensures that the value is a non-empty string
 * @param value The value to check
 * @returns The value if it is a non-empty string, or undefined otherwise
 */
export function nonEmptyString(value: unknown): string | undefined {
    return typeof value === "string" && value.trim().length > 0 ? value.trim() : undefined;
}

/**
 * Ensures that the value is a valid boolean. 
 * Also accepts stringified boolean values (i.e. "true" or "false").
 * @param value The value to check
 * @returns The value if it is a boolean, or undefined otherwise
 */
export function validBoolean(value: unknown): boolean | undefined {
    if (typeof value === "boolean") return value;
    if (typeof value === "string") {
        if (value === "true") return true;
        if (value === "false") return false;
    }
    return undefined;
}

/**
 * Ensures that the value is any URL-encodable primitive type
 * @param value The value to check
 * @returns The value if it is a URL-encodable primitive type, or undefined otherwise
 */
export function validPrimitive(value: unknown): UrlPrimitive | undefined {
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
        // Reuse the existing validation functions for basic types
        return validBoolean(value) ?? nonZeroNumber(value) ?? nonEmptyString(value);
    } else if (typeof value === "object" && value !== null && !Array.isArray(value) && !(value instanceof Date)) {
        // Allow object if it's not an array or a Date
        return value;
    }
    return undefined;
}

/**
 * Converts Tag objects to their tag property
 * @param tag The tag object
 * @returns The tag property of the object
 */
export function tagObjectToString(tag: unknown): string | undefined {
    const tagString = tag !== null && typeof tag === "object" && Object.prototype.hasOwnProperty.call(tag, "tag") ? (tag as Tag).tag.trim() : "";
    return tagString.length > 0 ? tagString : undefined;
}

/**
 * Converts strings to Tag objects
 * @param tag The tag string
 * @returns The tag object with the tag property
 */
export function stringToTagObject(tag: unknown): Tag | undefined {
    if (typeof tag !== "string" || tag.trim().length === 0) return undefined;
    return { __typename: "Tag" as const, tag: tag.trim() } as Tag;
}

/**
 * Map for converting input type values to search URL values
 */
export const inputTypeToSearch: { [key in InputType]: (value: unknown) => OrArray<UrlPrimitive> | undefined } = {
    [InputType.Checkbox]: (value) => arrayToSearch(value, validBoolean),
    [InputType.Dropzone]: () => undefined, // Not supported
    [InputType.IntegerInput]: (value) => nonZeroNumber(value),
    [InputType.JSON]: (value) => nonEmptyString(value),
    [InputType.LanguageInput]: (value) => arrayToSearch(value, nonEmptyString),
    [InputType.LinkItem]: (value) => nonEmptyString(value),
    [InputType.LinkUrl]: (value) => nonEmptyString(value),
    [InputType.Radio]: (value) => nonEmptyString(value),
    [InputType.Selector]: (value) => validPrimitive(value),
    [InputType.Slider]: (value) => nonZeroNumber(value),
    [InputType.Switch]: (value) => validBoolean(value),
    [InputType.TagSelector]: (value) => arrayToSearch(value, tagObjectToString),
    [InputType.Text]: (value) => nonEmptyString(value),
};

/**
 * Map for converting search URL values to input type values.
 * 
 * NOTE: Many of these may be the same, especially for basic types. However, this
 * mapping is necessary to ensure that the correct conversion function is used for
 * things like tags, where we go from an object to a string, and then back to an object.
 */
export const searchToInputType: { [key in InputType]: (value: OrArray<UrlPrimitive>) => unknown } = {
    [InputType.Checkbox]: (value) => searchToArray(value, validBoolean),
    [InputType.Dropzone]: () => undefined, // Not supported
    [InputType.IntegerInput]: (value) => nonZeroNumber(value),
    [InputType.JSON]: (value) => nonEmptyString(value),
    [InputType.LanguageInput]: (value) => searchToArray(value, nonEmptyString),
    [InputType.LinkItem]: (value) => nonEmptyString(value),
    [InputType.LinkUrl]: (value) => nonEmptyString(value),
    [InputType.Radio]: (value) => nonEmptyString(value),
    [InputType.Selector]: (value) => validPrimitive(value),
    [InputType.Slider]: (value) => nonZeroNumber(value),
    [InputType.Switch]: (value) => validBoolean(value),
    [InputType.TagSelector]: (value) => searchToArray(value, stringToTagObject),
    [InputType.Text]: (value) => nonEmptyString(value),
};

/**
 * Converts Formik values to a format suitable for URL search parameters. This function
 * uses a form schema to determine how to process each field's data type and converts
 * them into a format that can be serialized into a URL query string. This allows the
 * application to maintain the state of complex search forms across page reloads or
 * when sharing links.
 * 
 * NOTE: See `stringifySearchParams` to better understand what types of values are supported.
 * 
 * @param values The form values obtained from Formik.
 * @param schema The form schema that describes the types and properties of each field.
 * @returns An object where each key corresponds to a form field and each value is in a 
 *          format ready to be encoded into a URL query string.
 */
export function convertFormikForSearch(values: { [x: string]: any }, schema: FormSchema): ParseSearchParamsResult {
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through all fields in the schema
    for (const field of schema.fields) {
        // If field in values, convert and add to result
        if (values[field.fieldName]) {
            const value = inputTypeToSearch[field.type](values[field.fieldName]);
            if (value !== undefined) result[field.fieldName] = value;
        }
    }
    // Return result
    return result;
}

/**
 * Converts URL search parameters back to Formik values according to a form schema.
 * This function facilitates loading of search forms with pre-filled parameters from
 * a URL, maintaining the state of form fields after a page reload or when accessed
 * via a shared link. It ensures that all fields are returned in a format that Formik
 * can directly utilize to populate the form.
 * 
 * @param values The search URL values parsed from a query string.
 * @param schema The form schema to use, which describes how to reverse the conversion
 *               for each field type.
 * @returns An object formatted for Formik where each key corresponds to a form field 
 *          and each value is derived from the corresponding URL parameter.
 */
export function convertSearchForFormik(values: ParseSearchParamsResult, schema: FormSchema): { [x: string]: any } {
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through all fields in the schema
    for (const field of schema.fields) {
        // If field in values, convert and add to result
        const searchValue = values[field.fieldName];
        if (searchValue !== null && searchValue !== undefined) {
            const formValue = searchToInputType[field.type](searchValue);
            result[field.fieldName] = formValue;
        }
        // Otherwise, set as undefined
        else result[field.fieldName] = undefined;
    }
    // Return result
    return result;
}
