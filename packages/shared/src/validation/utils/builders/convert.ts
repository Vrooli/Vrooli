import * as yup from "yup";
import { handleRegex, urlRegex, walletAddressRegex } from "../regex.js";

/**
 * Strips non-numeric characters from a string, leaving a positive double
 */
export function toPosDouble(str: string | number) {
    if (typeof str === "number") {
        return Math.max(str, 0);
    }
    // First handle scientific notation by parsing as-is
    const scientificParsed = parseFloat(str);
    if (!isNaN(scientificParsed) && str.toLowerCase().includes("e")) {
        return Math.max(scientificParsed, 0);
    }
    // Otherwise strip non-numeric characters (except . for decimals)
    const cleaned = str.replace(/[^0-9.]/g, "");
    const result = parseFloat(cleaned);
    return isNaN(result) ? NaN : Math.max(result, 0);
}

/**
 * Strips non-numeric characters from a string, leaving a double
 */
export function toDouble(str: string | number) {
    if (typeof str === "number") {
        return str;
    }
    // First handle scientific notation by parsing as-is
    const scientificParsed = parseFloat(str);
    if (!isNaN(scientificParsed) && str.toLowerCase().includes("e")) {
        return scientificParsed;
    }
    // Otherwise strip non-numeric characters (except . for decimals and - for negatives)
    // But preserve the position of the negative sign
    const isNegative = str.trim().startsWith("-");
    const cleaned = str.replace(/[^0-9.]/g, "");
    const result = parseFloat(cleaned);
    if (isNaN(result)) return NaN;
    return isNegative ? -result : result;
}

/**
 * Strips non-numeric characters from a string, leaving a positive integer
 */
export function toPosInt(str: string) {
    // First handle scientific notation by parsing and truncating
    const scientificParsed = parseFloat(str);
    if (!isNaN(scientificParsed) && str.toLowerCase().includes("e")) {
        return Math.max(Math.floor(Math.abs(scientificParsed)), 0);
    }
    // For decimal numbers, parse first then truncate
    const decimalParsed = parseFloat(str);
    if (!isNaN(decimalParsed) && str.includes(".")) {
        return Math.max(Math.floor(Math.abs(decimalParsed)), 0);
    }
    // Otherwise strip non-numeric characters
    const cleaned = str.replace(/[^0-9]/g, "");
    const result = parseInt(cleaned, 10);
    return isNaN(result) ? NaN : result;
}


/**
 * Adds https:// to the beginning of the URL if it doesn't start with http:// or https://, 
 * and it also doesn't match other regexes that are used for other types of links.
 * @param value the URL
 * @returns the URL with https:// prepended to it
 */
export function addHttps(value: string | undefined): string {
    if (
        typeof value === "string" &&
        !value.startsWith("http://") &&
        !value.startsWith("https://") &&
        !walletAddressRegex.test(value) &&
        !handleRegex.test(value) &&
        urlRegex.test(`https://${value}`)
    ) {
        return `https://${value}`;
    }
    return value || "";
}

/**
 * Converts a TypeScript enum to a yup oneOf array
 */
export function enumToYup(enumObj: { [x: string]: any } | (() => { [x: string]: any })) {
    // If enumObj is a function, call it to get the enum
    const actualEnum = typeof enumObj === "function" ? enumObj() : enumObj;
    
    // Handle undefined enum gracefully (can happen with circular dependencies)
    if (!actualEnum) {
        console.warn("enumToYup called with undefined enum. This may be due to circular dependencies.");
        return yup.string().trim().removeEmptyString();
    }
    return yup.string().trim().removeEmptyString().oneOf(Object.values(actualEnum) as string[]);
}
