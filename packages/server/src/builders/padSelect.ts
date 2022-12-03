import { CustomError } from "../events";
import { isRelationshipObject } from "./isRelationshipObject";
import { WithSelect } from "./types";

/**
 * Adds "select" to the correct parts of an object to make it a Prisma select
 */
export const padSelect = <T extends { [x: string]: any }>(fields: T): WithSelect<T> => {
    // Ensure fields is an object
    if (!isRelationshipObject(fields)) throw new CustomError('0363', 'InternalError', ['en']);
    let converted: { [x: string]: any } = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length > 0) converted[key] = padSelect(fields[key]);
        else converted[key] = true;
    });
    return { select: converted } as any;
}