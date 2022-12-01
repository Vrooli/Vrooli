import { lowercaseFirstLetter } from "./lowercaseFirstLetter";
import { ExceptionsBuilderProps } from "./types";

/**
 * Assembles custom query exceptions (i.e. query has some condition OR <exceptions>). 
 * If an 'id' field is allowed (e.g. 'parent.id') and the current value is a string, then we treat as 
 * a 'connect' query (i.e. assume that the string is a primary key for the object)
 */
export function exceptionsBuilder({
    canQuery,
    defaultValue,
    exceptionField,
    input,
    mainField,
}: ExceptionsBuilderProps): { [x: string]: any } {
    // Initialize result
    const result: { [x: string]: any } = { [mainField]: input[mainField] ?? defaultValue };
    // Helper function for checking if a stringified object is a primitive or an array of primitives.
    // Returns boolean indicating whether it is a primitive, and the parsed object
    const getPrimitive = (x: string): [boolean, any] => {
        const primitiveCheck = (y: any): boolean => { return y === null || typeof y === 'string' || typeof y === 'number' || typeof y === 'boolean' };
        let value: any;
        try { value = JSON.parse(x); }
        catch (err) { return [false, undefined]; }
        if (Array.isArray(value)) {
            if (value.every(primitiveCheck)) return [true, value]
        }
        else if (primitiveCheck(value)) return [true, value];
        return [false, value];
    }
    /**
     * Helper function for converting a list of fields to a nested object
     * @param fields List of fields to convert
     * @param value Value to assign to the last field
     */
    const fieldsToObject = (fields: string[], value: any): { [x: string]: any } => {
        if (fields.length === 0) return value;
        const [field, ...rest] = fields;
        return { [field]: fieldsToObject(rest, value) };
    }
    /**
     * Helper function to add an object to the result's OR array
     * @param allowed Fields that are allowed to be queried
     * @param field Field's name
     * @param value Field's stringified value
     * @param recursedFields Nested fields in current recursion. These are used to generated nested queries
     */
    const addToOr = (allowed: string[], field: string, value: string, recursedFields: string[] = []): void => {
        const [isPrimitive, parsedValue] = getPrimitive(value);
        // Check if field is allowed
        if (isPrimitive && allowed.includes(field)) {
            // If not array, add to result
            if (!Array.isArray(parsedValue)) result.OR.push(fieldsToObject([...recursedFields, field], parsedValue));
            // Otherwise, wrap in { in: } and add to result
            else result.OR.push(fieldsToObject([...recursedFields, field], { in: parsedValue }));
        }
        // Check if field is allowed with 'id' appended
        else if (allowed.includes(`${field}.id`)) {
            // If not array, add to result
            if (!Array.isArray(parsedValue) && typeof parsedValue === 'string') result.OR.push(fieldsToObject([...recursedFields, field, 'id'], parsedValue));
            // Otherwise, wrap in { in: } and add to result
            else if (Array.isArray(parsedValue) && parsedValue.every(x => typeof x === 'string')) result.OR.push(fieldsToObject([...recursedFields, field, 'id'], { in: parsedValue }));
        }
        // Otherwise, check if we should recurse
        else if (typeof parsedValue === 'object' && field in parsedValue) {
            const matchingFields = allowed.filter(x => x.startsWith(`${field}.`));
            if (matchingFields.length > 0) {
                addToOr(
                    allowed.filter(x => x.startsWith(`${field}.`)),
                    field,
                    JSON.stringify(parsedValue[field]),
                    [...recursedFields, field],
                );
            }
        }
    }
    if (!(typeof input === 'object' && mainField in input)) return result;
    // Get mainField value
    // If exceptionField is present, wrap in OR
    if (exceptionField in input) {
        result.OR = [{ [mainField]: result[mainField] }];
        delete result[mainField];
        // If exceptionField is an array, add each item to OR
        if (Array.isArray(input[exceptionField])) {
            // Delete mainField from result, since it will be in OR
            for (const exception of input[exceptionField]) {
                addToOr(canQuery, lowercaseFirstLetter(exception.field), exception.value);
            }
        }
        // Otherwise, add exceptionField to OR
        else {
            addToOr(canQuery, lowercaseFirstLetter(input[exceptionField].field), input[exceptionField].value);
        }
    }
    return result;
}