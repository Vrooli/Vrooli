/**
 * Omit an array of keys from an object
 * @param object The object to omit keys from
 * @param keys The keys to omit
 * @returns The object with the omitted keys
 */
export function omit(object: { [x: string]: any }, keys: string[]): any {
    return Object.keys(object).reduce((acc: any, key) => {
        if (keys.indexOf(key) === -1) {
            acc[key] = object[key];
        }
        return acc;
    }, {});
}