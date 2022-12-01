import { ApolloError } from "types";
import { TFuncKey } from 'i18next';

type ErrKey = TFuncKey<'error', undefined>

/**
 * Finds the error code in an ApolloError object
 * @param error The GraphQL error to convert
 * @returns The error code, which should be a translation key
 */
 export const errorToCode = (error: ApolloError): ErrKey => {
    // If there is an errors array
    if (error.graphQLErrors) {
        // Loop through array and return first error code
        for (const err of error.graphQLErrors) {
            if (err.extensions?.code) {
                return err.extensions.code as ErrKey;
            }
        }
    }
    // If code not found, return default
    return 'ErrorUnknown';
};

/**
 * Checks if an error code is in a GraphQL error
 * @param error The GraphQL error
 * @param code The error code to check for
 * @returns True if the error code is in the error, false otherwise
 */
export const hasErrorCode = (error: ApolloError, code: ErrKey): boolean => {
    return Array.isArray(error.graphQLErrors) && error.graphQLErrors.some(e => 
        e.extensions?.code === code
    );
}