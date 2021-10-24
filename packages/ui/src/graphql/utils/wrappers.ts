// Wraps GraphQL mutations, to provide each mutation with the same functionality:
// - Success and error messages
// - Loading spinner
import isFunction from 'lodash/isFunction';
import { PUBS } from "utils";
import PubSub from 'pubsub-js';
import { ApolloCache, DefaultContext, MutationFunctionOptions, OperationVariables } from '@apollo/client';

interface Props {
    // useMutation function
    mutation: (options?: MutationFunctionOptions<any, OperationVariables, DefaultContext, ApolloCache<any>> | undefined) => Promise<any>;
    // data to pass into useMutation function
    data: Object;
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (response?: any) => boolean;
    // Message displayed on success
    successMessage?: (respone?: any) => string;
    // Debug data to print on success
    successData?: any;
    // Callback triggered on success
    onSuccess?: (response?: any) => any;
    // Message displayed on error
    errorMessage?: (response?: any) => string;
    // Debug data to print on error
    errorData?: any;
    // If true, display default error snack. Will not display if error message or data is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response?: any) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null | undefined;
}

export const mutationWrapper = ({ 
    mutation,
    data,
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
    if (spinnerDelay) PubSub.publish(PUBS.Loading, spinnerDelay);
    mutation(data).then((response) => {
        if (successCondition(response)) {
            if (successMessage || successData) PubSub.publish(PUBS.Snack, { message: successMessage && successMessage(response), ...successData });
            if (spinnerDelay) PubSub.publish(PUBS.Loading, false);
            if (isFunction(onSuccess)) onSuccess(response)
        } else {
            if (errorMessage || errorData) {
                PubSub.publish(PUBS.Snack, { message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
            }
            else if (showDefaultErrorSnack) PubSub.publish(PUBS.Snack, { message: 'Unknown error occurred.', severity: 'error', data: response });
            if (spinnerDelay) PubSub.publish(PUBS.Loading, false);
            if (isFunction(onError)) onError(response);
        }
    }).catch((response) => {
        if (spinnerDelay) PubSub.publish(PUBS.Loading, false);
        if (errorMessage || errorData) {
            PubSub.publish(PUBS.Snack, { message: errorMessage && errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
        }
        else if (showDefaultErrorSnack) {
            // Don't show internal errors, as they're often a block of code
            //TODO likely doesn't work
            const messageToShow = response.code === 'INTERNAL_SERVER_ERROR' ? 'Unknown error occurred.' : response.message ?? 'Unknown error occurred.';
            PubSub.publish(PUBS.Snack, { message: messageToShow, severity: 'error', data: response });
        }
        if (isFunction(onError)) onError(response);
    })
}