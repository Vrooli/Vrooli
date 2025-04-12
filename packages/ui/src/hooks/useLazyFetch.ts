import { HttpMethod, ServerResponse } from "@local/shared";
import { useCallback, useRef, useState } from "react";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { LazyRequestWithResult, ServerResponseWithTimestamp } from "../api/types.js";

type RequestState<TData> = {
    data: TData | undefined;
    errors: ServerResponse["errors"] | undefined;
    loading: boolean;
};

export type UseLazyFetchProps<TInput extends Record<string, any> | undefined> = {
    /** The URL to make the request to */
    endpoint?: string | undefined;
    inputs?: TInput;
    /** The HTTP method to use for the request (defaults to 'GET') */
    method?: HttpMethod;
    /** Additional options to pass to the `fetch` function */
    options?: RequestInit;
};

/**
 * Custom React hook for making HTTP requests.
 * It is "lazy" in the sense that it doesn't run automatically when the component mounts,
 * but instead provides a function that can be called to initiate the request.
 * This hook handles the lifecycle of the request and provides updates on its status through its return value.
 * @returns A tuple where the first element is a function to initiate the request and 
 * the second element is an object representing the current state of the request.
 */
export function useLazyFetch<TInput extends Record<string, any> | undefined, TData>({
    endpoint,
    inputs,
    method = "GET",
    options = {} as RequestInit,
}: UseLazyFetchProps<TInput>): [
        LazyRequestWithResult<TInput, TData>,
        RequestState<TData>
    ] {
    const [state, setState] = useState<RequestState<TData>>({
        loading: false,
        data: undefined,
        errors: undefined,
    });
    const stateRef = useRef(state);
    stateRef.current = state;

    // Store the latest endpoint, method, options and inputs in a ref
    const fetchParamsRef = useRef({ endpoint, method, options, inputs });
    fetchParamsRef.current = { endpoint, method, options, inputs };

    // Store the last time the fetch was called
    const lastFetchTimestampRef = useRef<number>(0);

    const getData = useCallback<LazyRequestWithResult<TInput, TData>>(async (input, inputOptions) => {
        // Update the inputs stored in the ref if a new input is provided
        if (input) {
            fetchParamsRef.current.inputs = input;
        }
        // Get fetch params from the ref
        const { endpoint, method, options, inputs } = fetchParamsRef.current;
        // Cancel if no endpoint is provided
        if (!endpoint && !inputOptions?.endpointOverride) {
            const trace = "0692";
            const message = "No endpoint provided to useLazyFetch";
            console.error(message);
            stateRef.current.errors = [{ trace, message }];
            setState(stateRef.current);
            return stateRef.current;
        }
        // Update the state to loading
        setState(s => ({ ...s, loading: true }));
        // Helper function for handling response
        function handleResponse({ data, errors, __fetchTimestamp }: ServerResponseWithTimestamp) {
            // If timestamp is older than the current fetch, the data is outdated and should be ignored
            const isOld = __fetchTimestamp < lastFetchTimestampRef.current;
            if (isOld) {
                console.warn(`Ignoring outdated fetch response. Latest fetch timestamp: ${lastFetchTimestampRef.current}, response timestamp: ${__fetchTimestamp}`);
                return stateRef.current;
            }
            // Update the last fetch timestamp
            lastFetchTimestampRef.current = __fetchTimestamp;
            // Update the state with the response
            stateRef.current = { loading: false, data, errors };
            setState(stateRef.current);
            // Display any errors
            if (Array.isArray(errors) && errors.length > 0) {
                if (inputOptions?.onError) {
                    inputOptions.onError(errors);
                }
                if (inputOptions?.displayError !== false) {
                    ServerResponseParser.displayErrors(errors);
                }
            }
            return stateRef.current;
        }
        // Make the request
        const result = await fetchData({
            endpoint: inputOptions?.endpointOverride || endpoint as string,
            inputs,
            method,
            options,
        })
            .then(handleResponse)
            .catch(handleResponse);
        return result;
    }, []);

    return [getData, state];
}
