import { isRelationshipObject } from "./isRelationshipObject";
import { WithSelect } from "./types";

/**
 * Adds "select" to the correct parts of an object to make it a Prisma select
 */
export const selPad = <T extends object>(fields: T): WithSelect<T> => {
    // Only pad if fields is an object
    if (!isRelationshipObject(fields)) return fields;
    // If fields is an object with the "select" key already, return it
    if ("select" in fields) return fields as any;
    const converted: object = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length > 0) converted[key] = selPad(fields[key]);
        else converted[key] = true;
    });
    return { select: converted } as WithSelect<T>;
};
