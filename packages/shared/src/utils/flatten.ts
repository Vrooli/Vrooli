/**
 * Flattens array one layer deep
 * @param array The array to flatten
 * @returns The flattened array.
 */
export function flatten<T>(array: T[]): T[] {
    return array.reduce((acc, item) => acc.concat(item), [] as T[]);
}
