import { type ParseSearchParamsResult, parseSearchParams } from "@local/shared";
import { type SetLocation } from "./types.js";


/**
 * Adds values to the current search params, without removing any existing values. 
 * If a key already exists, it will be overwritten.
 * @param setLocation Function to set the location
 * @param params Object with key/value pairs, representing search params
 */
export function addSearchParams(setLocation: SetLocation, params: { [key: string]: any }) {
    const currentParams = parseSearchParams();
    const newParams = { ...currentParams, ...params };
    setLocation(window.location.pathname, { replace: true, searchParams: newParams });
}

/**
 * Sets the search params, removing any existing values
 * @param setLocation Function to set the location
 * @param params Object with key/value pairs, representing search params
 */
export function setSearchParams(setLocation: SetLocation, params: { [key: string]: any }) {
    setLocation(window.location.pathname, { replace: true, searchParams: params });
}

/**
 * Keeps only the search params specified in the keys array, removing any others
 * @param setLocation Function to set the location
 * @param keep Array of keys to keep
 */
export function keepSearchParams(setLocation: SetLocation, keep: string[]) {
    const keepResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams();
    keep.forEach(key => {
        if (searchParams[key] !== undefined) keepResult[key] = searchParams[key];
    });
    setLocation(window.location.pathname, { replace: true, searchParams: keepResult });
}

/**
 * Removes the search params specified in the keys array, keeping any others
 * @param setLocation Function to set the location
 * @param remove Array of keys to remove
 */
export function removeSearchParams(setLocation: SetLocation, remove: string[]) {
    const removeResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams();
    let hasRemoved = false;
    Object.keys(searchParams).forEach(key => {
        if (!remove.includes(key)) {
            removeResult[key] = searchParams[key];
        } else {
            hasRemoved = true;
        }
    });
    if (!hasRemoved) return;
    setLocation(window.location.pathname, { replace: true, searchParams: removeResult });
}
