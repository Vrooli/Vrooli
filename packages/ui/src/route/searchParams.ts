import { SetLocation } from "./types";

type Primitive = string | number | boolean | object;
export type ParseSearchParamsResult = { [x: string]: Primitive | Primitive[] | ParseSearchParamsResult | null | undefined };

const encodeValue = (value: unknown) => {
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

const decodeValue = (value: unknown) => {
    if (typeof value === 'string') {
        return value;
    } else if (typeof value === 'object' && value !== null) {
        if (Array.isArray(value)) {
            return value.map(decodeValue);
        }
        return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, decodeValue(v)]));
    }
    return value;
};

/**
 * Converts object to url search params. 
 * Ignores top-level keys with null values.
 * 
 * NOTE: Values which are NOT supported are:
 * - undefined
 * - functions
 * - symbols
 * - BigInts, Infinity, -Infinity, NaN
 * - Dates
 * - Maps, Sets
 * @param params Object with key/value pairs, representing search params
 * @returns string of search params, matching the format of window.location.search
 */
export const stringifySearchParams = (params: ParseSearchParamsResult) => {
    const keys = Object.keys(params).filter(key => params[key] != null && params[key] !== undefined);
    const encodedParams = keys.map(key => {
        try {
            return `${encodeURIComponent(key)}=${encodeURIComponent(JSON.stringify(encodeValue(params[key])))}`;
        } catch (e: any) {
            console.error(`Error encoding value for key "${key}": ${e.message}`);
            return null;
        }
    }).filter(param => param !== null).join("&");
    return encodedParams ? `?${encodedParams}` : "";
};

/**
 * Converts url search params to object
 * @returns Object with key/value pairs, or empty object if no params
 */
export const parseSearchParams = (): ParseSearchParamsResult => {
    const params = new URLSearchParams(window.location.search);
    const obj = {};
    for (let [key, value] of params) {
        try {
            obj[decodeURIComponent(key)] = decodeValue(JSON.parse(decodeURIComponent(value)));
        } catch (e: any) {
            console.error(`Error decoding parameter "${key}": ${e.message}`);
        }
    }
    return obj;
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
