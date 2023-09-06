import { ErrorKey, exists } from "@local/shared";
import { ServerResponse } from "api";
import i18next from "i18next";

/**
 * Finds the error code in an ServerResponse object
 * @param response A response from the server
 * @returns The first error code, which should be a translation key
 */
const errorToCode = (response: ServerResponse): ErrorKey => {
    // If there is an errors array
    if (response.errors) {
        // Loop through array and return first error code
        for (const error of response.errors) {
            if (error?.code) {
                return error.code;
            }
        }
    }
    // If code not found, return default
    return "ErrorUnknown";
};

/**
 * Finds the error message in an ServerResponse object
 * @param response A response from the server
 * @param langauges User's languages
 * @returns The first error message, or code if message not found
 */
export const errorToMessage = (response: ServerResponse, languages: string[]): string => {
    // If there is an errors array
    if (response.errors) {
        // Loop through array and return first error message
        for (const error of response.errors) {
            if (error?.message) {
                return error.message;
            }
        }
    }
    // If message not found, return code instead
    const code = errorToCode(response);
    const lng = languages.length > 0 ? languages[0] : "en";
    return i18next.t(code, { lng, defaultValue: "Unknown error occurred." }) as unknown as string;
};

/**
 * Checks if a specific error code or codes exists in a response
 * @param response The response to check
 * @param codes The error code or codes to check for
 * @returns True if one of the error codes is in the error, false otherwise
 */
export const hasErrorCode = (error: ServerResponse, ...codes: ErrorKey[]): boolean => {
    return exists(error) && exists(error.errors) && error.errors.some((e) => exists(e.code) && codes.includes(e.code));
};
