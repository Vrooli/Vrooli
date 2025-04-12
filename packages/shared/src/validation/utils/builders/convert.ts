import * as yup from "yup";
import { handleRegex, urlRegex, walletAddressRegex } from "../regex.js";

/**
 * Strips non-numeric characters from a string, leaving a positive double
 */
export function toPosDouble(str: string | number) {
    if (typeof str === "number") {
        return Math.max(str, 0);
    }
    return parseFloat(str.replace(/[^0-9.]/g, ""));
}

/**
 * Strips non-numeric characters from a string, leaving a double
 */
export function toDouble(str: string | number) {
    if (typeof str === "number") {
        return str;
    }
    return parseFloat(str.replace(/[^0-9.-]/g, ""));
}

/**
 * Strips non-numeric characters from a string, leaving a positive integer
 */
export function toPosInt(str: string) {
    return parseInt(str.replace(/[^0-9]/g, ""), 10);
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
export function enumToYup(enumObj: { [x: string]: any }) {
    return yup.string().trim().removeEmptyString().oneOf(Object.values(enumObj));
}
