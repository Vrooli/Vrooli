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

/**
 * Compares two arrays for equality using a provided comparator function.
 * 
 * @template T - The type of elements in the arrays.
 * @param {T[]} a - The first array to compare.
 * @param {T[]} b - The second array to compare.
 * @param {(a: T, b: T) => boolean} comparator - A function that compares two elements of the arrays.
 * @returns {boolean} - Returns true if the arrays are equal based on the comparator, otherwise false.
 * 
 * @example
 * const arr1 = [1, 2, 3];
 * const arr2 = [1, 2, 3];
 * const numberComparator = (a: number, b: number) => a === b;
 * console.log(arraysEqual(arr1, arr2, numberComparator)); // Output: true
 * 
 * @example
 * const arr1 = ['a', 'b', 'c'];
 * const arr2 = ['a', 'b', 'c'];
 * const stringComparator = (a: string, b: string) => a === b;
 * console.log(arraysEqual(arr1, arr2, stringComparator)); // Output: true
 * 
 * @example
 * const arr1 = [{ id: 1 }, { id: 2 }];
 * const arr2 = [{ id: 1 }, { id: 2 }];
 * const objectComparator = (a: { id: number }, b: { id: number }) => a.id === b.id;
 * console.log(arraysEqual(arr1, arr2, objectComparator)); // Output: true
 */
export function arraysEqual<T>(a: T[], b: T[], comparator: (a: T, b: T) => boolean): boolean {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) {
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        if (!comparator(a[i]!, b[i]!)) return false;
    }
    return true;
}
