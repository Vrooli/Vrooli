import { HttpMethod, ServerResponse, TranslationKeyCommon, TranslationKeyError, exists } from "@local/shared";
import { useCallback } from "react";
import { PubSub } from "../utils/pubsub.js";
import { fetchData } from "./fetchData.js";
import { ServerResponseParser } from "./responseParser.js";
import { LazyRequestWithResult } from "./types.js";

const DEFAULT_SPINNER_DELAY_MS = 1000;

// For some reason, these snack message types break when we omit "severity". So we must redefine them here
type TranslatedSnackMessage<KeyList = TranslationKeyCommon | TranslationKeyError> = {
    messageKey: KeyList;
    messageVariables?: { [key: string]: string | number };
}
type UntranslatedSnackMessage = {
    message: string;
}
type SnackMessage<KeyList = TranslationKeyCommon | TranslationKeyError> = TranslatedSnackMessage<KeyList> | UntranslatedSnackMessage;
type SnackPub<KeyList = TranslationKeyCommon | TranslationKeyError> = SnackMessage<KeyList> & {
    autoHideDuration?: number | "persist";
    buttonClicked?: (event?: any) => unknown;
    buttonKey?: TranslationKeyCommon;
    buttonVariables?: { [key: string]: string | number };
    data?: any;
    /**
     * If ID is set, a snack with the same ID will be replaced
     */
    id?: string;
};

/**
 * Transforms an object into FormData, allowing nested objects and File instances.
 * This function works recursively to accommodate complex objects and arrays.
 *
 * @param {Object} obj - The object to be converted into FormData.
 * @param {FormData} form - The FormData object that is being built recursively. This should be undefined when initially called.
 * @param {string} namespace - The namespace for the formData entry being created. For nested objects or arrays, this will be the parent's formKey. This should be undefined when initially called.
 * @return {FormData} The final FormData object which is constructed from the input object.
 */
function objectToFormData<T extends object>(
    obj: T,
    form?: FormData,
    namespace?: string,
): FormData {
    const formData = form || new FormData();

    Object.entries(obj).forEach(([property, value]) => {
        if (value !== undefined && value !== null) {
            const formKey = namespace ? `${namespace}[${property}]` : property;

            if (value instanceof Date) {
                formData.append(formKey, value.toISOString());
            } else if (typeof value === "object" && !(value instanceof File)) {
                objectToFormData(value, formData, formKey);
            } else {
                formData.append(formKey, value);
            }
        }
    });

    return formData;
}

interface FetchLazyWrapperProps<Input extends object | undefined, Output> {
    fetch: LazyRequestWithResult<Input, Output>;
    /** Input to pass to endpoint */
    inputs?: Input;
    /** Callback to determine if mutation was a success, using mutation's return data */
    successCondition?: (data: Output) => boolean;
    /** Message displayed on success */
    successMessage?: (data: Output) => SnackPub<TranslationKeyCommon>;
    /** Callback triggered on success */
    onSuccess?: (data: Output) => unknown;
    /** Message displayed on error */
    errorMessage?: (response?: ServerResponse) => SnackPub<TranslationKeyError | TranslationKeyCommon>;
    /** If true, display default error snack. Will not display if error message is set */
    showDefaultErrorSnack?: boolean;
    /** Callback triggered on error */
    onError?: (response: ServerResponse) => unknown;
    /** Callback triggered on either success or error */
    onCompleted?: (response: ServerResponse) => unknown;
    /** Milliseconds before showing a spinner. If undefined or null, spinner disabled */
    spinnerDelay?: number | null;
}

export async function fetchLazyWrapper<Input extends object | undefined, Output>({
    fetch,
    inputs,
    successCondition = () => true,
    successMessage,
    onSuccess,
    errorMessage,
    showDefaultErrorSnack = true,
    onError,
    onCompleted,
    spinnerDelay = DEFAULT_SPINNER_DELAY_MS,
}: FetchLazyWrapperProps<Input, Output>): Promise<ServerResponse<Output>> {
    // Helper function to handle errors
    function handleError(data?: ServerResponse | undefined) {
        // Stop spinner
        if (spinnerDelay) PubSub.get().publish("loading", false);
        // Determine if error caused by bad response, or caught error
        const isError: boolean = exists(data) && exists(data.errors);
        // If specific error message is set, display it
        if (typeof errorMessage === "function") {
            PubSub.get().publish("snack", { ...errorMessage(data), severity: "Error", data });
        }
        // Otherwise, if show default error snack is set, display it
        else if (showDefaultErrorSnack) {
            PubSub.get().publish("snack", {
                message: ServerResponseParser.errorToMessage(data as ServerResponse, ["en"]),
                severity: "Error",
                data,
            });
        }
        // If error callback is set, call it
        if (typeof onError === "function") {
            if (isError) {
                onError(data as ServerResponse);
            } else {
                const error = { trace: "0694", message: "Unknown error occurred" };
                onError({ errors: [error] });
            }
        }
        // If completed callback is set, call it
        if (typeof onCompleted === "function") {
            if (isError) {
                onCompleted(data as ServerResponse);
            } else {
                const error = { trace: "0695", message: "Unknown error occurred" };
                onCompleted({ errors: [error] });
            }
        }
    }
    // Start loading spinner
    if (spinnerDelay) PubSub.get().publish("loading", spinnerDelay);
    let result: ServerResponse<Output> = {};
    // Convert inputs to FormData if they contain a File
    const finalInputs = inputs && Object.values(inputs).some(value => value instanceof File)
        ? objectToFormData(inputs)
        : inputs;
    await fetch(finalInputs as Input, { displayError: false })
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
                PubSub.get().publish("snack", { ...successMessage(data), severity: "Success" });
            }
            // If the success callback is set, call it
            if (typeof onSuccess === "function") {
                onSuccess(data);
            }
            // If the completed callback is set, call it
            if (typeof onCompleted === "function") {
                onCompleted(response);
            }
        }).catch((error: ServerResponse<Output>) => {
            result = error;
            handleError(error);
        }).finally(() => {
            // Stop spinner
            if (spinnerDelay) PubSub.get().publish("loading", false);
        });
    return result;
}

interface FetchWrapperProps<Input extends object | undefined, Output> extends Omit<FetchLazyWrapperProps<Input, Output>, "fetch"> {
    // Endpoint to call
    endpoint: string;
    // Method for endpoint
    method: HttpMethod;
}

export function fetchWrapper<Input extends object | undefined, Output>({
    endpoint,
    method,
    ...rest
}: FetchWrapperProps<Input, Output>) {
    return fetchLazyWrapper<Input, Output>({
        fetch: (inputs) => fetchData<Input | undefined, Output>({ endpoint, method, inputs }),
        ...rest,
    });
}

type UseSubmitHelperProps<Input extends object | undefined, Output> = FetchLazyWrapperProps<Input, Output> & {
    disabled?: boolean,
    existing?: object,
    isCreate: boolean,
}

/** Wraps around fetchLazyWrapper to provide common checks before submitting */
export function useSubmitHelper<Input extends object | undefined, Output>({
    disabled,
    existing,
    isCreate,
    ...props
}: UseSubmitHelperProps<Input, Output>) {
    return useCallback(function useSubmitHelperCallback() {
        if (disabled === true) {
            PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && (Array.isArray(existing) ? existing.length === 0 : !exists(existing))) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        fetchLazyWrapper<Input, Output>(props);
    }, [disabled, existing, isCreate, props]);
}
