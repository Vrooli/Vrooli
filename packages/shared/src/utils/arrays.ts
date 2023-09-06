/**
 * Finds items in first array that are not in second array.
 * @param array1
 * @param array2
 * @returns The difference of the two arrays.
 */
export function difference(array1: any[], array2: any[]): any[] {
    return array1.filter(item => array2.indexOf(item) === -1);
}

/**
 * Flattens array one layer deep
 * @param array The array to flatten
 * @returns The flattened array.
 */
export function flatten<T>(array: T[]): T[] {
    return array.reduce((acc, item) => acc.concat(item), [] as T[]);
}

/**
 * Finds unique items in array, using a comparer function.
 * @param array Array to find unique items in.
 * @param iteratee Iteratee to use to find unique items.
 * @returns Array of unique items.
 */
export function uniqBy(array: any[], iteratee: (item: any) => any): any[] {
    return array.filter((item, index, self) => {
        return self.findIndex(i => iteratee(i) === iteratee(item)) === index;
    });
}
