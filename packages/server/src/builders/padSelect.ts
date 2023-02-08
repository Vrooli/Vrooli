import { CustomError } from "../events";
import { isRelationshipObject } from "./isRelationshipObject";
import { WithSelect } from "./types";

/**
 * Adds "select" to the correct parts of an object to make it a Prisma select
 */
export const padSelect = <T extends { [x: string]: any }>(fields: T): WithSelect<T> => {
    console.log('in padSelect', fields)
    // Only pad if fields is an object
    if (!isRelationshipObject(fields)) return fields;
    // If fields is an object with the "select" key already, return it
    if ('select' in fields) return fields as any;
    let converted: { [x: string]: any } = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length > 0) converted[key] = padSelect(fields[key]);
        else converted[key] = true;
    });
    return { select: converted } as any;
}