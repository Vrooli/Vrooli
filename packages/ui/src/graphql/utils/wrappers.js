import _ from 'lodash';
import { PUBS, PubSub } from "utils";

export function mutationWrapper({ 
    mutation, // Mutation function
    data, // data passed into mutation
    successCondition = () => true, // function that determines if response was a success
    successMessage, // callback that returns a success message, using response data
    successData, // data passed to snack if success
    onSuccess, // callback if success
    errorMessage, // callback that returns an error message, using response data
    errorData, // data passed to snack if error
    showDefaultErrorSnack = true, // if true, display default error snack. Will not display if error message or data is set
    onError, // callback if error
    spinnerDelay = 1000, // how long to wait before showing a spinner, in milliseconds. If undefined or null, spinner disabled
}) {
    if (spinnerDelay) PubSub.publish(PUBS.Loading, spinnerDelay);
    mutation(data).then((response) => {
        if (successCondition(response)) {
            if (successMessage || successData) PubSub.publish(PUBS.Snack, { message: successMessage(response), ...successData });
            if (spinnerDelay) PubSub.publish(PUBS.Loading, false);
            if (_.isFunction(onSuccess)) onSuccess(response)
        } else {
            if (errorMessage || errorData) {
                PubSub.publish(PUBS.Snack, { message: errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
            }
            else if (showDefaultErrorSnack) PubSub.publish(PUBS.Snack, { message: 'Unknown error occurred.', severity: 'error', data: response });
            if (spinnerDelay) PubSub.publish(PUBS.Loading, false);
            if (_.isFunction(onError)) onError(response);
        }
    }).catch((response) => {
        if (spinnerDelay) PubSub.publish(PUBS.Loading, false);
        if (errorMessage || errorData) {
            PubSub.publish(PUBS.Snack, { message: errorMessage(response), ...errorData, severity: errorData?.severity ?? 'error', data: errorData?.data ?? response });
        }
        else if (showDefaultErrorSnack) {
            // Don't show internal errors, as they're often a block of code
            //TODO likely doesn't work
            const messageToShow = response.code === 'INTERNAL_SERVER_ERROR' ? 'Unknown error occurred.' : response.message ?? 'Unknown error occurred.';
            PubSub.publish(PUBS.Snack, { message: messageToShow, severity: 'error', data: response });
        }
        if (_.isFunction(onError)) onError(response);
    })
}