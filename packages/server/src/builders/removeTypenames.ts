/**
 * Removes the "__typename" field recursively from a JSON-serializable object
 * @param obj - JSON-serializable object with possible type fields
 * @return obj without type fields
 */
export const removeTypenames = (obj: { [x: string]: any }): { [x: string]: any } => {
    return JSON.parse(JSON.stringify(obj, (k, v) => (k === '__typename') ? undefined : v))
}