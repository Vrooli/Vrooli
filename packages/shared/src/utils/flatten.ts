/**
 * Flattens array one layer deep
 * @param array The array to flatten
 * @returns The flattened array.
 */
export function flatten(array: any[]): any[] {
    return array.reduce((acc, item) => acc.concat(item), []);
}