// Wraps GraphQL mutations, to provide each mutation with the same functionality:
// - Success and error messages
// - Loading spinner
import isFunction from 'lodash/isFunction';
import { Pubs } from "utils";
import PubSub from 'pubsub-js';
import { ApolloCache, DefaultContext, FetchResult, MutationFunctionOptions, OperationVariables } from '@apollo/client';

interface Props {
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
    onSuccess?: (response: FetchResult<any, Record<string, any>, Record<string, any>>) => any;
    // Message displayed on error
    errorMessage?: (response?: any) => string;
    // Debug data to print on error
    errorData?: any;
    // If true, display default error snack. Will not display if error message or data is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response?: any) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null;
}

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
}: Props) => {
    if (spinnerDelay) PubSub.publish(Pubs.Loading, spinnerDelay);
    mutation(input ? { variables: { input } } : undefined).then((response) => {
        if (successCondition(response)) {
            if (successMessage || successData) PubSub.publish(Pubs.Snack, { message: successMessage && successMessage(response), ...successData });
            if (spinnerDelay) PubSub.publish(Pubs.Loading, false);
            if (isFunction(onSuccess)) onSuccess(response)
        } else {
            if (errorMessage || errorData) {
                PubSub.publish(Pubs.Snack, { message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
            }
            else if (showDefaultErrorSnack) PubSub.publish(Pubs.Snack, { message: 'Unknown error occurred.', severity: 'error', data: response });
            if (spinnerDelay) PubSub.publish(Pubs.Loading, false);
            if (isFunction(onError)) onError(response);
        }
    }).catch((response) => {
        if (spinnerDelay) PubSub.publish(Pubs.Loading, false);
        if (errorMessage || errorData) {
            PubSub.publish(Pubs.Snack, { message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
        }
        else if (showDefaultErrorSnack) {
            // Don't show internal errors, as they're often a block of code
            //TODO likely doesn't work
            const messageToShow = response.code === 'INTERNAL_SERVER_ERROR' ? 'Unknown error occurred.' : response.message ?? 'Unknown error occurred.';
            PubSub.publish(Pubs.Snack, { message: messageToShow, severity: 'error', data: response });
        }
        if (isFunction(onError)) onError(response);
    })
}