import { InputType, Tag } from '@shared/consts';
import { FormSchema } from 'forms/types';

/**
 * Converts a radio button value (which can only be a string) to its underlying search URL value
 * @param value The value of the radio button
 * @returns The value as-is, a boolean, or undefined
 */
const radioValueToSearch = (value: string): string | boolean | undefined => {
    // Check if value should be a boolean
    if (value === 'true' || value === 'false') return value === 'true';
    // Check if value should be undefined
    if (value === 'undefined') return undefined;
    // Otherwise, return as-is
    return value;
};

/**
 * Converts a radio button search value to its radio button value (i.e. stringifies)
 * @param value The value of the radio button
 * @returns The value stringified
 */
const searchToRadioValue = (value: string | boolean | undefined): string => value + '';

/**
 * Converts an array of items to a search URL value
 * @param arr The array of items
 * @param convert The function to convert each item to a search URL value
 * @returns The items of the array converted, or undefined if the array is empty or undefined
 */
function arrayConvert<T, U>(arr: T[] | undefined, convert?: (T) => U): U[] | undefined {
    if (Array.isArray(arr)) {
        if (arr.length === 0) return undefined;
        return convert ? arr.map(convert) : arr as unknown as U[];
    }
    return undefined;
};

/**
 * Map for converting input type values to search URL values
 */
const inputTypeToSearch: { [key in InputType]: (value: any) => any } = {
    [InputType.Checkbox]: (value) => value, //TODO
    [InputType.Dropzone]: (value) => value, //TODO
    [InputType.IntegerInput]: (value) => value, //TODO
    [InputType.JSON]: (value) => value, //TODO
    [InputType.LanguageInput]: (value: string[]) => arrayConvert(value),
    [InputType.Markdown]: (value: string) => typeof value === 'string' && value.trim().length > 0 ? value.trim() : undefined,
    [InputType.Radio]: (value: string) => radioValueToSearch(value),
    [InputType.Selector]: (value) => value, //TODO
    [InputType.Slider]: (value) => value, //TODO
    [InputType.Switch]: (value) => value, //TODO 
    [InputType.TagSelector]: (value: Tag[]) => arrayConvert(value, ({ tag }) => tag),
    [InputType.TextField]: (value: string) => typeof value === 'string' && value.trim().length > 0 ? value.trim() : undefined,
}

/**
 * Map for converting search URL values to input type values
 */
const searchToInputType: { [key in InputType]: (value: any) => any } = {
    [InputType.Checkbox]: (value) => value, //TODO
    [InputType.Dropzone]: (value) => value, //TODO
    [InputType.IntegerInput]: (value) => value, //TODO
    [InputType.JSON]: (value) => value, //TODO
    [InputType.LanguageInput]: (value: string[]) => arrayConvert(value),
    [InputType.Markdown]: (value: string) => value,
    [InputType.Radio]: (value: string) => searchToRadioValue(value),
    [InputType.Selector]: (value) => value, //TODO
    [InputType.Slider]: (value) => value, //TODO
    [InputType.Switch]: (value) => value, //TODO
    [InputType.TagSelector]: (value: string[]) => arrayConvert(value, (tag) => ({ tag })),
    [InputType.TextField]: (value: string) => value,
}

/**
 * Converts formik values to search URL values 
 * @param values The formik values
 * @param schema The form schema to use
 * @returns Values converted for stringifySearchParams
 */
export const convertFormikForSearch = (values: { [x: string]: any }, schema: FormSchema): { [x: string]: any } => {
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
 * Converts search URL values to formik values
 * @param values The search URL values
 * @param schema The form schema to use
 * @returns Values converted for formik
 */
export const convertSearchForFormik = (values: { [x: string]: any }, schema: FormSchema): { [x: string]: any } => {
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through all fields in the schema
    for (const field of schema.fields) {
        // If field in values, convert and add to result
        if (values[field.fieldName]) {
            const value = searchToInputType[field.type](values[field.fieldName]);
            result[field.fieldName] = value;
        }
        // Otherwise, set as undefined
        else result[field.fieldName] = undefined;
    }
    // Return result
    return result;
}