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