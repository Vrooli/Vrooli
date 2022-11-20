import { CustomError } from "../events/error";

/**
 * Recursively sorts keys of an object alphabetically. 
 * This can be used to simplify object comparison
 * @parm obj The object to sort
 * @returns The sorted object
 */
export const sortObjectKeys = (obj: any): any => {
    if (typeof obj !== 'object' || obj === null || Object.prototype.toString.call(obj) !== '[object Date]') {
        return obj;
    }
    const sorted: { [x: string]: any } = {};
    Object.keys(obj).sort().forEach(key => {
        sorted[key] = sortObjectKeys(obj[key]);
    });
    return sorted;
}

/**
 * Parses JSON string, recursively sorts all keys alphabetically, and restringifies
 * @param stringified The stringified JSON object
 * @returns The sortified, stringified object
 */
export const sortify = (stringified: string): string => {
    try {
        const obj = JSON.parse(stringified);
        return JSON.stringify(sortObjectKeys(obj));
    } catch (error) {
        throw new CustomError('InvalidArgs', 'Invalid JSON stringified object', { trace: '0210' });
    }
}