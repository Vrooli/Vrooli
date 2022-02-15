import { CODE } from "@local/shared";

/**
 * Converts a GraphQL error into a snackbar message.
 * @param error The error to convert
 * @param defaultMessage The default message to display if no message is provided
 */
export const errorToSnack = (error: any, defaultMessage: string = 'Unknown error occurred.') => {
    if (!error.message) return defaultMessage;
    // Loop through possible error codes
    for (const code in CODE) {
        // If the error code matches, return the message
        if ((error.message as string).endsWith(CODE[code].message)) 
            return CODE[code].snack ?? CODE[code].message ?? defaultMessage;
    }
    // If no message was found, return the default message
    return defaultMessage;
};