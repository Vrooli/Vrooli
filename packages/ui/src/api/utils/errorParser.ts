import { ApolloError } from "@apollo/client";
import { ErrorKey } from "@local/shared";

/**
 * Finds the error code in an ApolloError object
 * @param error The GraphQL error to convert
 * @returns The error code, which should be a translation key
 */
export const errorToCode = (error: ApolloError): ErrorKey => {
    // If there is an errors array
    if (error.graphQLErrors) {
        // Loop through array and return first error code
        for (const err of error.graphQLErrors) {
            if (err.extensions?.code) {
                return err.extensions.code as ErrorKey;
            }
        }
    }
    // If code not found, return default
    return 'ErrorUnknown';
};

/**
 * Finds the error message in an ApolloError object
 * @param error The GraphQL error to convert
 * @returns The error message
 */
export const errorToMessage = (error: ApolloError): string => {
    // If there is an errors array
    if (error.graphQLErrors) {
        // Loop through array and return first error message
        for (const err of error.graphQLErrors) {
            if (err.message) {
                return err.message;
            }
        }
    }
    // If message not found, return code instead
    return errorToCode(error);
};

/**
 * Checks if an error code is in a GraphQL error
 * @param error The GraphQL error
 * @param code The error code to check for
 * @returns True if the error code is in the error, false otherwise
 */
export const hasErrorCode = (error: ApolloError, code: ErrorKey): boolean => {
    return Array.isArray(error.graphQLErrors) && error.graphQLErrors.some(e =>
        e.extensions?.code === code
    );
}