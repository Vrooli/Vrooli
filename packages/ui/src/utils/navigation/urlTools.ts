import { SnackSeverity } from "components";
import { SetLocation } from "types";
import { PubSub } from "utils/pubsub";

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
    const filteredKeys = keys.filter(key => params[key] !== undefined && params[key] !== null);
    const encodedParams = filteredKeys.map(key => encodeURIComponent(key) + '=' + encodeURIComponent(JSON.stringify(params[key]))).join('&');
    return '?' + encodedParams;
}

/**
 * Adds values to the current search params, without removing any existing values. 
 * If a key already exists, it will be overwritten.
 * @param setLocation Function to set the location
 * @param params Object with key/value pairs, representing search params
 */
export const addSearchParams = (setLocation: SetLocation, params: { [key: string]: any }) => {
    const currentParams = parseSearchParams();
    const newParams = { ...currentParams, ...params };
    setLocation(`${window.location.pathname}${stringifySearchParams(newParams)}`, { replace: true });
};

/**
 * Sets the search params, removing any existing values
 * @param setLocation Function to set the location
 * @param params Object with key/value pairs, representing search params
 */
export const setSearchParams = (setLocation: SetLocation, params: { [key: string]: any }) => {
    setLocation(`${window.location.pathname}${stringifySearchParams(params)}`, { replace: true });
};

/**
 * Keeps only the search params specified in the keys array, removing any others
 * @param setLocation Function to set the location
 * @param keep Array of keys to keep
 */
export const keepSearchParams = (setLocation: SetLocation, keep: string[]) => {
    const keepResult: { [key: string]: any } = {};
    const searchParams = parseSearchParams();
    keep.forEach(key => {
        if (searchParams[key] !== undefined) keepResult[key] = searchParams[key];
    });
    setLocation(`${window.location.pathname}${stringifySearchParams(keepResult)}`, { replace: true });
};

/**
 * Removes the search params specified in the keys array, keeping any others
 * @param setLocation Function to set the location
 * @param remove Array of keys to remove
 */
export const removeSearchParams = (setLocation: SetLocation, remove: string[]) => {
    const removeResult: { [key: string]: any } = {};
    const searchParams = parseSearchParams();
    Object.keys(searchParams).forEach(key => {
        if (!remove.includes(key)) removeResult[key] = searchParams[key];
    });
    setLocation(`${window.location.pathname}${stringifySearchParams(removeResult)}`, { replace: true });
};

/**
 * @returns last part of the url path
 * @param offset Number of parts to offset from the end of the path (default: 0)
 * @returns part of the url path that is <offset> parts from the end, or empty string if no path
 */
export const getLastUrlPart = (offset: number = 0): string => {
    let parts = window.location.pathname.split('/');
    // Remove any empty strings
    parts = parts.filter(part => part !== '');
    // Check to make sure there is a part at the offset
    if (parts.length < offset + 1) return '';
    return parts[parts.length - offset - 1];
}

/**
 * Converts a string to a BigInt
 * @param value String to convert
 * @param radix Radix (base) to use
 * @returns 
 */
function toBigInt(value: string, radix: number) {
    return [...value.toString()]
        .reduce((r, v) => r * BigInt(radix) + BigInt(parseInt(v, radix)), 0n);
}

/**
 * Converts a UUID into a shorter, base 36 string without dashes. 
 * Useful for displaying UUIDs in a more compact format, such as in a URL.
 * @param uuid v4 UUID to convert
 * @returns base 36 string without dashes
 */
export const uuidToBase36 = (uuid: string): string => {
    try {
        const base36 = toBigInt(uuid.replace(/-/g, ''), 16).toString(36);
        return base36 === '0' ? '' : base36;
    } catch (error) {
        PubSub.get().publishSnack({ message: 'Could not convert ID', severity: SnackSeverity.Error, data: { uuid } });
        return '';
    }
}

/**
 * Converts a base 36 string without dashes into a UUID.
 * @param base36 base 36 string without dashes
 * @param showError Whether to show an error snack if the conversion fails
 * @returns v4 UUID
 */
export const base36ToUuid = (base36: string, showError = true): string => {
    try {
        // Convert to base 16. If the ID is less than 32 characters, pad start with 0s. 
        // Then, insert dashes
        const uuid = toBigInt(base36, 36).toString(16).padStart(32, '0').replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, '$1-$2-$3-$4-$5');
        return uuid === '0' ? '' : uuid;
    } catch (error) {
        if (showError) PubSub.get().publishSnack({ message: 'Could not parse ID in URL', severity: SnackSeverity.Error, data: { base36 } });
        return '';
    }
}