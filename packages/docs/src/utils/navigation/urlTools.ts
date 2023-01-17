import { exists } from "@shared/utils/src";

type Primitive = string | number | boolean;
type ParseSearchParamsResult = { [x: string]: Primitive | Primitive[] | ParseSearchParamsResult };

/**
 * Converts url search params to object
 * See https://stackoverflow.com/a/8649003/10240279
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (): ParseSearchParamsResult => {
    const searchParams = window.location.search;
    if (searchParams.length <= 1 || !searchParams.startsWith('?')) return {};
    let search = searchParams.substring(1);
    try {
        // First, loop through and wrap all strings in quotes (if not already). 
        // This is required to parse as a valid JSON object.
        // Find every substring that's between a "=" and a "&" (or the end of the string).
        // If it's not already wrapped in quotes, and it doesn't contain any URL encoded characters, and it is not a boolean, wrap it in quotes.
        search = search.replace(/([^&=]+)=([^&]*)/g, (match, key, value) => {
            if (value.startsWith('"') || value.includes('%') || value === 'true' || value === 'false') return match;
            return `${key}="${value}"`;
        });
        // Decode and parse the search params
        const parsed = JSON.parse('{"'
            + decodeURI(search)
                .replace(/&/g, ',"')
                .replace(/=/g, '":')
                .replace(/%2F/g, '/')
                .replace(/%5B/g, '[')
                .replace(/%5D/g, ']')
                .replace(/%5C/g, '\\')
                .replace(/%2C/g, ',')
                .replace(/%3A/g, ':')
            + '}');
        // For any value that might be JSON, try to parse
        Object.keys(parsed).forEach((key) => {
            const value = parsed[key];
            if (typeof value === 'string' && value.startsWith('{') && value.endsWith('}')) {
                try {
                    parsed[key] = JSON.parse(value);
                } catch (e) {
                    // Do nothing
                }
            }
        });
        return parsed;
    } catch (error) {
        console.error('Could not parse search params', error);
        return {};
    }
}

/**
 * Converts object to url search params. 
 * Ignores keys with undefined or null values.
 * @param params Object with key/value pairs, representing search params
 * @returns string of search params, matching the format of window.location.search
 */
export const stringifySearchParams = (params: { [key: string]: any }): string => {
    const keys = Object.keys(params);
    if (keys.length === 0) return '';
    // Filter out any keys which are associated with undefined or null values
    const filteredKeys = keys.filter(key => exists(params[key]));
    const encodedParams = filteredKeys.map(key => encodeURIComponent(key) + '=' + encodeURIComponent(JSON.stringify(params[key]))).join('&');
    return '?' + encodedParams;
}