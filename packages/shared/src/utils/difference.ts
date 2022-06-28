/**
 * Finds items in first array that are not in second array.
 * @param array1
 * @param array2
 * @returns The difference of the two arrays.
 */
export function difference(array1: any[], array2: any[]): any[] {
    return array1.filter(item => array2.indexOf(item) === -1);
}
