/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import * as yup from 'yup';
import { MessageParams } from 'yup/lib/types';

/**
 * Error message for max string length
 */
export const maxStrErr = (params: { max: number } & MessageParams) => {
    const amountOver = params.value.length - params.max;
    if (amountOver === 1) {
        return "1 character over the limit";
    } else {
        return `${amountOver} characters over the limit`;
    }
}

/**
 * Error message for min string length
 */
export const minStrErr = (params: { min: number } & MessageParams) => {
    const amountUnder = params.min - params.value.length;
    if (amountUnder === 1) {
        return "1 character under the limit";
    } else {
        return `${amountUnder} characters under the limit`;
    }
}

/**
 * Error message for max number
 */
export const maxNumErr = (params: { max: number } & MessageParams) => {
    return `Minimum value is ${params.max}`;
}

/**
 * Error message for min number
 */
export const minNumErr = (params: { min: number } & MessageParams) => {
    return `Minimum value is ${params.min}`;
}

/**
 * Error message for required field
 */
export const reqErr = () => {
    return `This field is required`;
}

/**
 * Calculates major, moderate, and minor versions from a version string
 * Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
 * Ex: 1 => major = 1, moderate = 0, minor = 0
 * Ex: 1.2 => major = 1, moderate = 2, minor = 0
 * Ex: asdfasdf (or any other invalid number) => major = 1, moderate = 0, minor = 0
 * @param version Version string
 * @returns Major, moderate, and minor versions
 */
export const calculateVersionsFromString = (version: string): { major: number, moderate: number, minor: number } => {
    const [major, moderate, minor] = version.split('.').map(v => parseInt(v));
    return {
        major: major || 0,
        moderate: moderate || 0,
        minor: minor || 0
    }
}

/**
 * Determines if a version is greater than or equal to the minimum version
 * @param version Version string
 * @param minimumVersion Minimum version string
 * @returns True if version is greater than or equal to minimum version
 */
export const meetsMinimumVersion = (version: string, minimumVersion: string): boolean => {
    // Parse versions
    const { major: major1, moderate: moderate1, minor: minor1 } = calculateVersionsFromString(version);
    const { major: major2, moderate: moderate2, minor: minor2 } = calculateVersionsFromString(minimumVersion);
    // Return false if major version is less than minimum
    if (major1 < major2) return false;
    // Return false if major version is equal to minimum and moderate version is less than minimum
    if (major1 === major2 && moderate1 < moderate2) return false;
    // Return false if major and moderate versions are equal to minimum and minor version is less than minimum
    if (major1 === major2 && moderate1 === moderate2 && minor1 < minor2) return false;
    // Otherwise, return true
    return true;
}

/**
 * Test for minimum version
 */
export const minVersionTest = (minVersion: string): [string, string, (value: string | undefined) => boolean] => {
    return [
        'version',
        `Minimum version is ${minVersion}`,
        (value: string | undefined) => {
            if (!value) return true;
            return meetsMinimumVersion(value, minVersion);
        }
    ]
}

/**
 * Transform for converting empty/whitespace strings to undefined
 */
export const blankToUndefined = (value: string | undefined) => {
    if (!value) return undefined;
    const trimmed = value.trim();
    if (trimmed === '') return undefined;
    return trimmed;
}

/**
 * Appends .notRequired().default(undefined) to a yup field
 */
export const opt = <T extends yup.AnySchema>(field: T) => {
    return field.notRequired().default(undefined);
}

/**
 * Appends .required(reqErr) to a yup field
 */
export const req = <T extends yup.AnySchema>(field: T) => {
    return field.required(reqErr);
}

/**
 * Creates an array of a required field
 */
export const reqArr = <T extends yup.AnySchema>(field: T) => {
    return yup.array().of(req(field));
}

/**
 * Creates an array of an optional field
 */
export const optArr = <T extends yup.AnySchema>(field: T) => {
    return yup.array().of(opt(field));
}

export const id = yup.string().transform(blankToUndefined).max(256, maxStrErr)
export const bio = yup.string().transform(blankToUndefined).max(2048, maxStrErr);
export const email = yup.string().transform(blankToUndefined).email().max(256, maxStrErr);
export const description = yup.string().transform(blankToUndefined).max(2048, maxStrErr)
export const helpText = yup.string().transform(blankToUndefined).max(2048, maxStrErr)
export const language = yup.string().transform(blankToUndefined).min(2, minStrErr).max(3, maxStrErr) // Language code
export const name = yup.string().transform(blankToUndefined).min(3, minStrErr).max(128, maxStrErr)
export const handle = yup.string().transform(blankToUndefined).min(3, minStrErr).max(16, maxStrErr).nullable() // ADA Handle
export const tag = yup.string().transform(blankToUndefined).min(2, minStrErr).max(64, maxStrErr)
export const version = (minVersion: string = '0.0.1') => yup.string().transform(blankToUndefined).max(16, maxStrErr).test(...minVersionTest(minVersion))
export const idArray = reqArr(id)
export const tagArray = reqArr(tag)









type RelationshipType = 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Update';

export type YupModel = {
    create: yup.ObjectSchema<any>;
    update: yup.ObjectSchema<any>;
}

// Array if isOneToOne is false, otherwise single
type MaybeArray<T extends 'object' | 'id', IsOneToOne extends 'one' | 'many'> =
    T extends 'object' ?
    IsOneToOne extends 'one' ? yup.ObjectSchema<any> : yup.ArraySchema<any> :
    IsOneToOne extends 'one' ? yup.StringSchema : yup.ArraySchema<any>;

// Array if isOneToOne is false, otherwise boolean
type MaybeArrayBoolean<IsOneToOne extends 'one' | 'many'> =
    IsOneToOne extends 'one' ? yup.BooleanSchema : yup.ArraySchema<any>;

type RelOutput<
    IsOneToOne extends 'one' | 'many',
    RelTypes extends string,
    FieldName extends string,
> = (
        ({ [x in `${FieldName}Connect`]: 'Connect' extends RelTypes ? MaybeArray<'id', IsOneToOne> : never }) &
        ({ [x in `${FieldName}Create`]: 'Create' extends RelTypes ? MaybeArray<'object', IsOneToOne> : never }) &
        ({ [x in `${FieldName}Delete`]: 'Delete' extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Disconnect`]: 'Disconnect' extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Update`]: 'Update' extends RelTypes ? MaybeArray<'object', IsOneToOne> : never })
    )

/**
 * Creates the validation fields for a relationship
 * @param relation The name of the relationship field
 * @param relTypes The allowed operations on the relations (e.g. create, connect)
 * @param yupModel The relationship's validation object
 * @param isRequired "opt" or "req" to mark the fields as optional or required. 
 * If required, only one "Connect" or "Create" is actually marked as required, and the other 
 * fields are marked as optional. If optional, all fields are marked as optional.
 * @param isOneToOne "one" if the relationship is one-to-one, and "many" otherwise. This makes the results a single object instead of an array
 * @returns An object with the validation fields for the relationship
 */
export const rel = <
    IsOneToOne extends 'one' | 'many',
    IsRequired extends 'opt' | 'req',
    RelTypes extends readonly RelationshipType[],
    FieldName extends string,
    Model extends YupModel,
>(
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: IsOneToOne,
    isRequired: IsRequired,
    model: Model,
): RelOutput<IsOneToOne, RelTypes[number], FieldName> => {
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through relation types
    for (const t of relTypes) {
        // Determine if field is required. If both 'Connect' and 'Create' are allowed, both 
        // are marked as optional here. This is because yup defines this one-of rule in the second
        // parameter of object.shape()
        // Count the number of times 'Connect' and 'Create' are in relTypes
        const connectCreateCount = relTypes.filter(x => x === 'Connect' || x === 'Create').length;
        const required = isRequired === 'req' && connectCreateCount === 1 && (t === 'Connect' || t === 'Create');
        // Add validation field to result
        if (t === 'Connect') {
            result[`${relation}${t}`] = isOneToOne === 'one' ?
                required ? req(id) : opt(id) :
                required ? reqArr(id) : optArr(id);
        }
        else if (t === 'Create') {
            result[`${relation}${t}`] = isOneToOne === 'one' ?
                required ? req(model.create) : opt(model.create) :
                required ? reqArr(model.create) : optArr(model.create);
        }
        else if (t === 'Delete') {
            result[`${relation}${t}`] = isOneToOne === 'one' ? opt(yup.bool()) : optArr(id);
        }
        else if (t === 'Disconnect') {
            result[`${relation}${t}`] = isOneToOne === 'one' ? opt(yup.bool()) : optArr(id);
        }
        else if (t === 'Update') {
            result[`${relation}${t}`] = isOneToOne === 'one' ? opt(model.update) : optArr(model.update);
        }
    }
    // Return result
    return result as any;
};