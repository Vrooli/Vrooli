import { DUMMY_ID, uuid } from "@local/shared";

/**
 * Helper function for setting a list of primitive fields of a create 
 * shape. Essentially, adds every field that's defined, and converts nulls to 
 * undefined
 * 
 * NOTE: Due to TypeScript limitations, return type assumes that every field 
 * will be defined, even if it's undefined
 */
export const createPrims = <T, K extends keyof T>(
    object: T,
    ...fields: (K | [K, (val: any) => any])[]
): { [F in K]: Exclude<T[F], null | undefined> } => {
    // Create prims
    const prims = fields.reduce((acc, field) => {
        const key = Array.isArray(field) ? field[0] : field;
        const value = Array.isArray(field) ? field[1](object[key]) : object[key];
        return { ...acc, [key]: value !== null ? value : undefined };
    }, {}) as any;

    // If "id" is defined in fields, make sure it's not DUMMY_ID
    if (!fields.some(field => Array.isArray(field) ? field[0] === "id" : field === "id")) return prims;

    return { ...prims, id: prims.id === DUMMY_ID ? uuid() : prims.id };
};
