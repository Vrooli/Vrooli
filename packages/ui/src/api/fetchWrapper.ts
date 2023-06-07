import { CommonKey, ErrorKey, exists } from "@local/shared";
import { Method, ServerResponse } from "api";
import { PubSub } from "utils/pubsub";
import { errorToMessage } from "./errorParser";
import { fetchData } from "./fetchData";

// For some reason, these snack message types break when we omit "severity". So we must redefine them here
type TranslatedSnackMessage<KeyList = CommonKey | ErrorKey> = {
    messageKey: KeyList;
    messageVariables?: { [key: string]: string | number };
}
type UntranslatedSnackMessage = {
    message: string;
}
type SnackMessage<KeyList = CommonKey | ErrorKey> = TranslatedSnackMessage<KeyList> | UntranslatedSnackMessage;
type SnackPub<KeyList = CommonKey | ErrorKey> = SnackMessage<KeyList> & {
    autoHideDuration?: number | "persist";
    buttonClicked?: (event?: any) => any;
    buttonKey?: CommonKey;
    buttonVariables?: { [key: string]: string | number };
    data?: any;
    /**
     * If ID is set, a snack with the same ID will be replaced
     */
    id?: string;
};

interface FetchLazyWrapperProps<Input extends object | undefined, Output> {
    fetch: (input?: Input) => Promise<ServerResponse<Output>>;
    // Input to pass to endpoint
    inputs?: Input;
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (data: Output) => boolean;
    // Message displayed on success
    successMessage?: (data: Output) => SnackPub<CommonKey>;
    // Callback triggered on success
    onSuccess?: (data: Output) => any;
    // Message displayed on error
    errorMessage?: (response?: ServerResponse) => SnackPub<ErrorKey | CommonKey>;
    // If true, display default error snack. Will not display if error message is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response: ServerResponse) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null;
}

export const fetchLazyWrapper = async <Input extends object | undefined, Output>({
    fetch,
    inputs,
    successCondition = () => true,
    successMessage,
    onSuccess,
    errorMessage,
    showDefaultErrorSnack = true,
    onError,
    spinnerDelay = 1000,
}: FetchLazyWrapperProps<Input, Output>): Promise<ServerResponse<Output>> => {
    // Helper function to handle errors
    const handleError = (data?: ServerResponse | undefined) => {
        // Stop spinner
        if (spinnerDelay) PubSub.get().publishLoading(false);
        // Determine if error caused by bad response, or caught error
        const isError: boolean = exists(data) && exists(data.errors);
        // If specific error message is set, display it
        if (typeof errorMessage === "function") {
            PubSub.get().publishSnack({ ...errorMessage(data), severity: "Error", data });
        }
        // Otherwise, if show default error snack is set, display it
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({
                message: errorToMessage(data as ServerResponse, ["en"]),
                severity: "Error",
                data,
            });
        }
        // If error callback is set, call it
        if (typeof onError === "function") {
            onError(isError ? data as ServerResponse : { errors: [{ message: "Unknown error occurred" }] });
        }
    };
    // Start loading spinner
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    let result: ServerResponse<Output> = {};
    await fetch(inputs)
        .then((response: ServerResponse<Output>) => {
            result = response;
            // If response is null or undefined or not an object, then there must be an error
            if (!response || !response.data || typeof response.data !== "object") {
                handleError(response);
                return;
            }
            const data: Output = response.data;
            // If this is a Count object with count = 0, then there is an error
            if (Object.prototype.hasOwnProperty.call(data, "count") && (data as any).count === 0) {
                handleError(response);
                return;
            }
            // If data.success is false, then there is an error
            if (Object.prototype.hasOwnProperty.call(data, "success") && !(data as any).success) {
                handleError(response);
                return;
            }
            // If the success condition callback returns false, then there is an error
            if (typeof successCondition === "function" && !successCondition(data)) {
                handleError(response);
                return;
            }
            // If the success message callback is set, publish it
            if (typeof successMessage === "function") {
                PubSub.get().publishSnack({ ...successMessage(data), severity: "Success" });
            }
            // If the success callback is set, call it
            if (typeof onSuccess === "function") {
                onSuccess(data);
            }
        }).catch((error: ServerResponse<Output>) => {
            result = error;
            handleError(error);
        }).finally(() => {
            // Stop spinner
            if (spinnerDelay) PubSub.get().publishLoading(false);
        });
    return result;
};

interface FetchWrapperProps<Input extends object | undefined, Output> extends Omit<FetchLazyWrapperProps<Input, Output>, "fetch"> {
    // Endpoint to call
    endpoint: string;
    // Method for endpoint
    method: Method;
}

export const fetchWrapper = <Input extends object | undefined, Output>({
    endpoint,
    method,
    ...rest
}: FetchWrapperProps<Input, Output>) => {
    return fetchLazyWrapper<Input, Output>({
        fetch: (inputs) => fetchData<Input | undefined, Output>({ endpoint, method, inputs }),
        ...rest,
    });
};