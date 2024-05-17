/**
 * Recursively encodes values in preparation for JSON serialization and URL encoding.
 * This function specifically targets the handling of percent signs (%) in strings
 * to ensure they are preserved accurately in the URL parameters by replacing them with '%25'.
 * This avoids issues with URL encoding where percent-encoded values might be misinterpreted.
 * 
 * - For strings, all '%' characters are replaced with '%25'.
 * - For arrays, the function is applied recursively to each element.
 * - For objects, the function is applied recursively to each value.
 * 
 * @param {unknown} value - The value to be encoded. Can be of any type.
 * @returns {unknown} - The encoded value, with all '%' characters in strings replaced by '%25'.
 *                      Arrays and objects are handled recursively, and other types are returned as is.
 */
export const encodeValue = (value: unknown) => {
    if (typeof value === 'string') {
        // encodeURIComponent will skip what looks like percent-encoded values. 
        // For this reason, we must manually replace all '%' characters with '%25'
        return value.replace(/%/g, '%25');
    } else if (typeof value === 'object' && value !== null) {
        if (Array.isArray(value)) {
            return value.map(encodeValue);
        }
        return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, encodeValue(v)]));
    }
    return value;
};

/**
 * Recursively processes values after JSON parsing, intended to be the inverse of encodeValue.
 * 
 * @param {unknown} value - The value to be decoded, typically after JSON parsing.
 * @returns {unknown} - The value without any special encoding from encodeValue.
 */
export const decodeValue = (value: unknown) => {
    if (typeof value === 'string') {
        return value.replace(/%25/g, '%');
    } else if (typeof value === 'object' && value !== null) {
        if (Array.isArray(value)) {
            return value.map(decodeValue);
        }
        return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, decodeValue(v)]));
    }
    return value;
};