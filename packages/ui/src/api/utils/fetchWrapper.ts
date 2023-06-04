import { ApolloError } from "@apollo/client";
import { CommonKey, ErrorKey, exists } from "@local/shared";
import { PubSub, SnackPub } from "utils/pubsub";
import { errorToCode } from "./errorParser";
import { fetchData } from "./fetchTools";

interface FestWrapperProps<Input extends object | undefined, Output extends object> {
    // Endpoint to call
    endpoint: string;
    // Input to pass to endpoint
    input?: Input;
    // Method for endpoint (GET, POST, PUT, etc.)
    method: string;
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (data: Output) => boolean;
    // Message displayed on success
    successMessage?: (data: Output) => SnackPub<CommonKey>
    // Callback triggered on success
    onSuccess?: (data: Output) => any;
    // Message displayed on error
    errorMessage?: (response?: ApolloError | Output) => SnackPub<ErrorKey | CommonKey>;
    // If true, display default error snack. Will not display if error message is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response: ApolloError) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null;
}

export const fetchWrapper = <Input extends object | undefined, Output extends object>({
    endpoint,
    input,
    method,
    successCondition = () => true,
    successMessage,
    onSuccess,
    errorMessage,
    showDefaultErrorSnack = true,
    onError,
    spinnerDelay = 1000,
}: FestWrapperProps<Input, Output>) => {
    // Helper function to handle errors
    const handleError = (data?: Output | ApolloError | undefined) => {
        // Stop spinner
        if (spinnerDelay) PubSub.get().publishLoading(false);
        // Determine if error caused by bad response, or caught error
        const isApolloError: boolean = exists(data) && Object.prototype.hasOwnProperty.call(data, "graphQLErrors");
        // If specific error message is set, display it
        if (typeof errorMessage === "function") {
            PubSub.get().publishSnack({ ...(errorMessage(data) as any), severity: "Error", data });
        }
        // Otherwise, if show default error snack is set, display it
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({ messageKey: isApolloError ? errorToCode(data as ApolloError) : "ErrorUnknown", severity: "Error", data });
        }
        // If error callback is set, call it
        if (typeof onError === "function") {
            onError((isApolloError ? data : { message: "Unknown error occurred" }) as ApolloError);
        }
    };
    // Start loading spinner
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    fetchData({ endpoint, input, method })
        .then((response: { data?: Output | null | undefined }) => {
            // If response is null or undefined or not an object, then there must be an error
            if (!response || !response.data || typeof response.data !== "object") {
                handleError();
                return;
            }
            // We likely need to go one layer deeper to get the actual data, since the 
            // result from Apollo is wrapped in an object with the endpoint name as the key. 
            // If this is not the case, then we are (hopefully) using a custom hook which already 
            // removes the extra layer.
            const data: Output = (Object.keys(response.data).length === 1 && !["success", "count"].includes(Object.keys(response.data)[0])) ? Object.values(response.data)[0] : response.data;
            // If this is a Count object with count = 0, then there is an error
            if (Object.prototype.hasOwnProperty.call(data, "count") && (data as any).count === 0) {
                handleError(data as any);
                return;
            }
            // If data.success is false, then there is an error
            if (Object.prototype.hasOwnProperty.call(data, "success") && !(data as any).success) {
                handleError(data as any);
                return;
            }
            // If the success condition callback returns false, then there is an error
            if (typeof successCondition === "function" && !successCondition(data)) {
                handleError(data);
                return;
            }
            // If the success message callback is set, publish it
            if (typeof successMessage === "function") {
                PubSub.get().publishSnack(successMessage(data));
            }
            // Stop spinner
            if (spinnerDelay) PubSub.get().publishLoading(false);
            // If the success callback is set, call it
            if (typeof onSuccess === "function") {
                onSuccess(data);
            }
        }).catch((error: ApolloError) => {
            handleError(error);
        });
};

interface FestLazyWrapperProps<Input extends object | undefined, Output extends object> extends Omit<FestWrapperProps<Input, Output>, "endpoint" | "method"> {
    fetch: (input?: Input) => Promise<Output>;
}

export const fetchLazyWrapper = <Input extends object | undefined, Output extends object>({
    endpoint,
    input,
    method,
    successCondition = () => true,
    successMessage,
    onSuccess,
    errorMessage,
    showDefaultErrorSnack = true,
    onError,
    spinnerDelay = 1000,
}: FestLazyWrapperProps<Input, Output>) => {
    // Helper function to handle errors
    const handleError = (data?: Output | ApolloError | undefined) => {
        // Stop spinner
        if (spinnerDelay) PubSub.get().publishLoading(false);
        // Determine if error caused by bad response, or caught error
        const isApolloError: boolean = exists(data) && Object.prototype.hasOwnProperty.call(data, "graphQLErrors");
        // If specific error message is set, display it
        if (typeof errorMessage === "function") {
            PubSub.get().publishSnack({ ...(errorMessage(data) as any), severity: "Error", data });
        }
        // Otherwise, if show default error snack is set, display it
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({ messageKey: isApolloError ? errorToCode(data as ApolloError) : "ErrorUnknown", severity: "Error", data });
        }
        // If error callback is set, call it
        if (typeof onError === "function") {
            onError((isApolloError ? data : { message: "Unknown error occurred" }) as ApolloError);
        }
    };
    // Start loading spinner
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    fetchData({ endpoint, input, method })
        .then((response: { data?: Output | null | undefined }) => {
            // If response is null or undefined or not an object, then there must be an error
            if (!response || !response.data || typeof response.data !== "object") {
                handleError();
                return;
            }
            // We likely need to go one layer deeper to get the actual data, since the 
            // result from Apollo is wrapped in an object with the endpoint name as the key. 
            // If this is not the case, then we are (hopefully) using a custom hook which already 
            // removes the extra layer.
            const data: Output = (Object.keys(response.data).length === 1 && !["success", "count"].includes(Object.keys(response.data)[0])) ? Object.values(response.data)[0] : response.data;
            // If this is a Count object with count = 0, then there is an error
            if (Object.prototype.hasOwnProperty.call(data, "count") && (data as any).count === 0) {
                handleError(data as any);
                return;
            }
            // If data.success is false, then there is an error
            if (Object.prototype.hasOwnProperty.call(data, "success") && !(data as any).success) {
                handleError(data as any);
                return;
            }
            // If the success condition callback returns false, then there is an error
            if (typeof successCondition === "function" && !successCondition(data)) {
                handleError(data);
                return;
            }
            // If the success message callback is set, publish it
            if (typeof successMessage === "function") {
                PubSub.get().publishSnack(successMessage(data));
            }
            // Stop spinner
            if (spinnerDelay) PubSub.get().publishLoading(false);
            // If the success callback is set, call it
            if (typeof onSuccess === "function") {
                onSuccess(data);
            }
        }).catch((error: ApolloError) => {
            handleError(error);
        });
};
