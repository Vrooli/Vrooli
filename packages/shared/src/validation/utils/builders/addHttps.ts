import { handleRegex, urlRegex, walletAddressRegex } from "../regex";

/**
 * Adds https:// to the beginning of the URL if it doesn't start with http:// or https://, 
 * and it also doesn't match other regexes that are used for other types of links.
 * @param value the URL
 * @returns the URL with https:// prepended to it
 */
export const addHttps = (value: string | undefined) => {
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
    return value;
};
