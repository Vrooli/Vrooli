import { CODE } from "@local/shared";

/**
 * Converts a GraphQL error into a snackbar message.
 * @param error The error to convert
 * @param defaultMessage The default message to display if no message is provided
 */
export const errorToSnack = (error: any, defaultMessage: string = 'Unknown error occurred.') => {
    let message = defaultMessage;
    if (Array.isArray(error)) {
        if (error.length > 0) {
            if (error[0].message) message = error[0].message;
        }
    }
    else if (typeof error === 'object') {
        if (error.message) message = error.message;
    }
    // Loop through possible error codes
    for (const code in CODE) {
        // If the error code matches, return the message
        if (message.endsWith(CODE[code].message)) 
            return CODE[code].snack ?? CODE[code].message ?? defaultMessage;
    }
    // If no message was found, return the default message
    return message;
};