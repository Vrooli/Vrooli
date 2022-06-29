import { Pubs } from "utils";
import PubSub from 'pubsub-js';
import { ApolloCache, DefaultContext, MutationFunctionOptions, OperationVariables } from '@apollo/client';
import { errorToMessage } from './errorParser';
import { ApolloError, ApolloResponse } from 'types';

interface MutationWrapperProps {
    // useMutation function
    mutation: (options?: MutationFunctionOptions<any, OperationVariables, DefaultContext, ApolloCache<any>> | undefined) => Promise<any>;
    // data to pass into useMutation function
    input?: Object;
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (response?: any) => boolean;
    // Message displayed on success
    successMessage?: (respone?: any) => string;
    // Debug data to print on success
    successData?: any;
    // Callback triggered on success
    onSuccess?: (response: ApolloResponse) => any;
    // Message displayed on error
    errorMessage?: (response?: any) => string;
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
export const mutationWrapper = ({ 
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
}: MutationWrapperProps) => {
    if (spinnerDelay) PubSub.publish(Pubs.Loading, spinnerDelay);
    mutation(input ? { variables: { input } } : undefined).then((response: ApolloResponse) => {
        if (successCondition(response)) {
            if (successMessage || successData) PubSub.publish(Pubs.Snack, { message: successMessage && successMessage(response), ...successData });
            if (spinnerDelay) PubSub.publish(Pubs.Loading, false);
            if (onSuccess && typeof onSuccess === 'function') onSuccess(response);
        } else {
            if (errorMessage || errorData) {
                PubSub.publish(Pubs.Snack, { message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
            }
            else if (showDefaultErrorSnack) {
                PubSub.publish(Pubs.Snack, { message: 'Unknown error occurred.', severity: 'error', data: response });
            }
            if (spinnerDelay) PubSub.publish(Pubs.Loading, false);
            if (onError && typeof onError === 'function') onError({ message: 'Unknown error occurred.' });
        }
    }).catch((response: ApolloError) => {
        if (spinnerDelay) PubSub.publish(Pubs.Loading, false);
        if (errorMessage || errorData) {
            PubSub.publish(Pubs.Snack, { message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
        }
        else if (showDefaultErrorSnack) {
            PubSub.publish(Pubs.Snack, { message: errorToMessage(response), severity: 'error', data: response });
        }
        if (onError && typeof onError === 'function') onError(response);
    })
}