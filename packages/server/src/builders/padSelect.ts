import { WithSelect } from "./types";

/**
 * Adds "select" to the correct parts of an object to make it a Prisma select
 */
export const padSelect = <T extends { [x: string]: any }>(fields: T): WithSelect<T> => {
    let converted: { [x: string]: any } = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length > 0) converted[key] = padSelect(fields[key]);
        else converted[key] = true;
    });
    return { select: converted } as any;
}