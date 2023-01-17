import { DotNotation } from "@shared/consts";

/**
 * Sets data in an object using dot notation (ex: 'parent.child.property')
 */
export const setDotNotationValue = <T extends Record<string, any>>(
    object: T,
    notation: DotNotation<T>,
    value: any
) => {
    if (!object || !notation) return null;
    const keys = (notation as string).split('.');
    const lastKey = keys.pop() as string;
    const lastObj: Record<string, any> = keys.reduce((obj: Record<string,any>, key) => obj[key] = obj[key] || {}, object);
    lastObj[lastKey] = value;
    return object;
}