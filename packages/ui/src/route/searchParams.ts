import { SetLocation } from "./useLocation";

type Primitive = string | number | boolean;
export type ParseSearchParamsResult = { [x: string]: Primitive | Primitive[] | ParseSearchParamsResult };

/**
 * Converts url search params to object
 * See https://stackoverflow.com/a/8649003/10240279
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (): ParseSearchParamsResult => {
    const searchParams = window.location.search;
    if (searchParams.length <= 1 || !searchParams.startsWith("?")) return {};
    let search = searchParams.substring(1);
    try {
        // First, loop through and wrap all strings in quotes (if not already). 
        // This is required to parse as a valid JSON object.
        // Find every substring that's between a "=" and a "&" (or the end of the string).
        // If it's not already wrapped in quotes, and it doesn't contain any URL encoded characters, and it is not a boolean, wrap it in quotes.
        search = search.replace(/([^&=]+)=([^&]*)/g, (match, key, value) => {
            if (value.startsWith("\"") || value.includes("%") || value === "true" || value === "false") return match;
            return `${key}="${value}"`;
        });
        // Decode and parse the search params
        const parsed = JSON.parse("{\""
            + decodeURI(search)
                .replace(/&/g, ",\"")
                .replace(/=/g, "\":")
                .replace(/%2F/g, "/")
                .replace(/%5B/g, "[")
                .replace(/%5D/g, "]")
                .replace(/%5C/g, "\\")
                .replace(/%2C/g, ",")
                .replace(/%3A/g, ":")
            + "}");
        // For any value that might be JSON, try to parse
        Object.keys(parsed).forEach((key) => {
            const value = parsed[key];
            if (typeof value === "string" && value.startsWith("{") && value.endsWith("}")) {
                try {
                    parsed[key] = JSON.parse(value);
                } catch (e) {
                    // Do nothing
                }
            }
        });
        return parsed;
    } catch (error) {
        console.error("Could not parse search params", error);
        return {};
    }
};

/**
 * Converts object to url search params. 
 * Ignores keys with undefined or null values.
 * @param params Object with key/value pairs, representing search params
 * @returns string of search params, matching the format of window.location.search
 */
export const stringifySearchParams = (params: object): string => {
    const keys = Object.keys(params);
    if (keys.length === 0) return "";
    // Filter out any keys which are associated with undefined or null values
    const filteredKeys = keys.filter(key => params[key] !== null && params[key] !== undefined);
    const encodedParams = filteredKeys.map(key => encodeURIComponent(key) + "=" + encodeURIComponent(JSON.stringify(params[key]))).join("&");
    return "?" + encodedParams;
};

/**
 * Adds values to the current search params, without removing any existing values. 
 * If a key already exists, it will be overwritten.
 * @param setLocation Function to set the location
 * @param params Object with key/value pairs, representing search params
 */
export const addSearchParams = (setLocation: SetLocation, params: { [key: string]: any }) => {
    const currentParams = parseSearchParams();
    const newParams = { ...currentParams, ...params };
    setLocation(window.location.pathname, { replace: true, searchParams: newParams });
};

/**
 * Sets the search params, removing any existing values
 * @param setLocation Function to set the location
 * @param params Object with key/value pairs, representing search params
 */
export const setSearchParams = (setLocation: SetLocation, params: { [key: string]: any }) => {
    setLocation(window.location.pathname, { replace: true, searchParams: params });
};

/**
 * Keeps only the search params specified in the keys array, removing any others
 * @param setLocation Function to set the location
 * @param keep Array of keys to keep
 */
export const keepSearchParams = (setLocation: SetLocation, keep: string[]) => {
    const keepResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams();
    keep.forEach(key => {
        if (searchParams[key] !== undefined) keepResult[key] = searchParams[key];
    });
    setLocation(window.location.pathname, { replace: true, searchParams: keepResult });
};

/**
 * Removes the search params specified in the keys array, keeping any others
 * @param setLocation Function to set the location
 * @param remove Array of keys to remove
 */
export const removeSearchParams = (setLocation: SetLocation, remove: string[]) => {
    const removeResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams();
    Object.keys(searchParams).forEach(key => {
        if (!remove.includes(key)) removeResult[key] = searchParams[key];
    });
    setLocation(window.location.pathname, { replace: true, searchParams: removeResult });
};
