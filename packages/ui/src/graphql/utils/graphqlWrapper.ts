import { Pubs, PubSub } from "utils";
import { ApolloCache, DefaultContext, MutationFunctionOptions, QueryFunctionOptions } from '@apollo/client';
import { errorToMessage } from './errorParser';
import { ApolloError } from 'types';

interface MutationWrapperProps<Output, Input extends { input: { [x: string]: any } }> {
    // useMutation function
    mutation: (options?: MutationFunctionOptions<Output, Input, DefaultContext, ApolloCache<any>> | undefined) => Promise<any>;
    // data to pass into useMutation function
    input?: Input['input'];
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (response: { data: Output }) => boolean;
    // Message displayed on success
    successMessage?: (respone: { data: Output }) => string;
    // Debug data to print on success
    successData?: any;
    // Callback triggered on success
    onSuccess?: (response: { data: Output }) => any;
    // Message displayed on error
    errorMessage?: (response?: ApolloError | { data: Output }) => string;
    // Debug data to print on error
    errorData?: any;
    // If true, display default error snack. Will not display if error message or data is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response: ApolloError) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null;
}

/**
 * Wraps GraphQL mutations, to provide each mutation with the following functionality:
 * - Success and error messages
 * - Loading spinner
 */
export const mutationWrapper = <Output, Input extends { input: any }>({ 
    mutation,
    input,
    successCondition = () => true,
    successMessage,
    successData,
    onSuccess,
    errorMessage,
    errorData,
    showDefaultErrorSnack = true,
    onError,
    spinnerDelay = 1000,
}: MutationWrapperProps<Output, Input>) => {
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    mutation(input ? { variables: { input } } as Input['input'] : undefined).then((response: { data: Output }) => {
        if (successCondition(response)) {
            if (successMessage || successData) PubSub.get().publishSnack({ message: successMessage && successMessage(response), ...successData });
            if (spinnerDelay) PubSub.get().publishLoading(false);
            if (onSuccess && typeof onSuccess === 'function') onSuccess(response);
        } else {
            if (errorMessage || errorData) {
                PubSub.get().publishSnack({ message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
            }
            else if (showDefaultErrorSnack) {
                PubSub.get().publishSnack({ message: 'Unknown error occurred.', severity: 'error', data: response });
            }
            if (spinnerDelay) PubSub.get().publish(Pubs.Loading, false);
            if (onError && typeof onError === 'function') onError({ message: 'Unknown error occurred.' });
        }
    }).catch((response: ApolloError) => {
        if (spinnerDelay) PubSub.get().publishLoading(false);
        if (errorMessage || errorData) {
            PubSub.get().publishSnack({ message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
        }
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({ message: errorToMessage(response), severity: 'error', data: response });
        }
        if (onError && typeof onError === 'function') onError(response);
    })
}

interface QueryWrapperProps<Output, Input extends { input: { [x: string]: any } }> {
    // useQuery function
    query: (options?: QueryFunctionOptions<Output, Input> | undefined) => Promise<any>;
    // data to pass into useQuery function
    input?: Input['input'];
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (response: { data: Output }) => boolean;
    // Message displayed on success
    successMessage?: (respone: { data: Output }) => string;
    // Debug data to print on success
    successData?: any;
    // Callback triggered on success
    onSuccess?: (response: { data: Output }) => any;
    // Message displayed on error
    errorMessage?: (response?: ApolloError | { data: Output }) => string;
    // Debug data to print on error
    errorData?: any;
    // If true, display default error snack. Will not display if error message or data is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response: ApolloError) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null;
}

/**
 * Wraps GraphQL queries, to provide each mutation with the following functionality:
 * - Success and error messages
 * - Loading spinner
 */
export const queryWrapper = <Output, Input extends { input: any }>({ 
    query,
    input,
    successCondition = () => true,
    successMessage,
    successData,
    onSuccess,
    errorMessage,
    errorData,
    showDefaultErrorSnack = true,
    onError,
    spinnerDelay = 1000,
}: QueryWrapperProps<Output, Input>) => {
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    query(input ? { variables: { input } } as Input['input'] : undefined).then((response: { data: Output }) => {
        if (successCondition(response)) {
            if (successMessage || successData) PubSub.get().publishSnack({ message: successMessage && successMessage(response), ...successData });
            if (spinnerDelay) PubSub.get().publishLoading(false);
            if (onSuccess && typeof onSuccess === 'function') onSuccess(response);
        } else {
            if (errorMessage || errorData) {
                PubSub.get().publishSnack({ message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
            }
            else if (showDefaultErrorSnack) {
                PubSub.get().publishSnack({ message: 'Unknown error occurred.', severity: 'error', data: response });
            }
            if (spinnerDelay) PubSub.get().publish(Pubs.Loading, false);
            if (onError && typeof onError === 'function') onError({ message: 'Unknown error occurred.' });
        }
    }).catch((response: ApolloError) => {
        if (spinnerDelay) PubSub.get().publishLoading(false);
        if (errorMessage || errorData) {
            PubSub.get().publishSnack({ message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
        }
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({ message: errorToMessage(response), severity: 'error', data: response });
        }
        if (onError && typeof onError === 'function') onError(response);
    })
}