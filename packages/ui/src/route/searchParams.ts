// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-01 - Fixed 4 'any' type assertions with 'unknown'
import { parseSearchParams, type ParseSearchParamsResult } from "@vrooli/shared";
import { type SetLocation } from "./types.js";


/**
 * Adds values to the current search params, without removing any existing values. 
 * If a key already exists, it will be overwritten.
 * @param currentSearch Current search string
 * @param params Object with key/value pairs, representing search params
 * @returns New search string
 */
export function addSearchParamsString(currentSearch: string, params: { [key: string]: unknown }): string {
    const currentParams = parseSearchParams(currentSearch);
    const newParams = { ...currentParams, ...params };
    return Object.entries(newParams)
        .map(([key, value]) => `${key}=${encodeURIComponent(JSON.stringify(value))}`)
        .join("&");
}

/**
 * Sets the search params, removing any existing values
 * @param params Object with key/value pairs, representing search params
 * @returns New search string
 */
export function setSearchParamsString(params: { [key: string]: unknown }): string {
    return Object.entries(params)
        .map(([key, value]) => `${key}=${encodeURIComponent(JSON.stringify(value))}`)
        .join("&");
}

/**
 * Keeps only the search params specified in the keys array, removing any others
 * @param currentSearch Current search string
 * @param keep Array of keys to keep
 * @returns New search string
 */
export function keepSearchParamsString(currentSearch: string, keep: string[]): string {
    const keepResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams(currentSearch);
    keep.forEach(key => {
        if (searchParams[key] !== undefined) keepResult[key] = searchParams[key];
    });
    return Object.entries(keepResult)
        .map(([key, value]) => `${key}=${encodeURIComponent(JSON.stringify(value))}`)
        .join("&");
}

/**
 * Removes the search params specified in the keys array, keeping any others
 * @param currentSearch Current search string
 * @param remove Array of keys to remove
 * @returns New search string
 */
export function removeSearchParamsString(currentSearch: string, remove: string[]): string {
    const removeResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams(currentSearch);
    Object.keys(searchParams).forEach(key => {
        if (!remove.includes(key)) {
            removeResult[key] = searchParams[key];
        }
    });
    return Object.entries(removeResult)
        .map(([key, value]) => `${key}=${encodeURIComponent(JSON.stringify(value))}`)
        .join("&");
}

// Navigation functions that work with setLocation
/**
 * Adds search params and updates the location using setLocation
 */
export function addSearchParams(setLocation: SetLocation, params: { [key: string]: unknown }): void {
    const currentParams = parseSearchParams(window.location.search);
    const newParams = { ...currentParams, ...params };
    setLocation(window.location.pathname, {
        replace: true,
        searchParams: newParams,
    });
}

/**
 * Sets search params and updates the location using setLocation
 */
export function setSearchParams(setLocation: SetLocation, params: { [key: string]: unknown }): void {
    setLocation(window.location.pathname, {
        replace: true,
        searchParams: params,
    });
}

/**
 * Keeps only specified search params and updates the location using setLocation
 */
export function keepSearchParams(setLocation: SetLocation, keep: string[]): void {
    const keepResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams(window.location.search);
    keep.forEach(key => {
        if (searchParams[key] !== undefined) keepResult[key] = searchParams[key];
    });
    setLocation(window.location.pathname, {
        replace: true,
        searchParams: keepResult,
    });
}

/**
 * Removes specified search params and updates the location using setLocation
 */
export function removeSearchParams(setLocation: SetLocation, remove: string[]): void {
    const removeResult: ParseSearchParamsResult = {};
    const searchParams = parseSearchParams(window.location.search);
    Object.keys(searchParams).forEach(key => {
        if (!remove.includes(key)) {
            removeResult[key] = searchParams[key];
        }
    });
    setLocation(window.location.pathname, {
        replace: true,
        searchParams: removeResult,
    });
}
