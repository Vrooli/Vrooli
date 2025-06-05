import { type HttpMethod, type ServerResponse } from "@vrooli/shared";
import { useCallback, useEffect, useRef, useState } from "react";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { type LazyRequestWithResult, type ServerResponseWithTimestamp } from "../api/types.js";
import { useDebounce } from "./useDebounce.js";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    errors: ServerResponse["errors"] | undefined;
};

type UseFetchProps<TInput extends Record<string, any> | undefined, TData> = {
    debounceMs?: number;
    endpoint: string | undefined;
    inputs?: TInput;
    method?: HttpMethod;
    options?: RequestInit;
    /** Omits rest endpoint base from URL */
    omitRestBase?: boolean;
};

/**
 * Custom React hook for making HTTP requests.
 * It is "non-lazy" in the sense that it runs automatically when the component mounts
 * and whenever the inputs change.
 * This hook handles the lifecycle of the request and provides updates on its status through its return value.
 *
 * @param endpoint - The URL to make the request to.
 * @param method - The HTTP method to use for the request (defaults to 'GET').
 * @param options - Additional options to pass to the `fetch` function.
 * @returns An object containing the current state of the request, and a function to manually re-fetch the data.
 */
export function useFetch<TInput extends Record<string, any> | undefined, TData>({
    endpoint,
    inputs = {} as TInput,
    method = "GET",
    options = {} as RequestInit,
    omitRestBase = false,
    debounceMs = 0,
}: UseFetchProps<TInput, TData>, deps: any[] = []):
    RequestState<TData> & { refetch: (input?: TInput) => unknown } {
    const [state, setState] = useState<RequestState<TData>>({
        loading: false,
        data: undefined,
        errors: undefined,
    });

    const refetch = useCallback<LazyRequestWithResult<TInput, TData>>(async (input, inputOptions) => {
        if (!endpoint && !inputOptions?.endpointOverride) {
            const trace = "0693";
            const message = "No endpoint provided to useLazyFetch";
            console.error(message);
            return { errors: [{ trace, message }] };
        }
        // Update the inputs stored in the ref if a new input is provided
        if (input) {
            inputs = input;
        }

        setState(s => ({ ...s, loading: true }));

        const result = await fetchData({
            endpoint: endpoint ?? inputOptions?.endpointOverride as string,
            inputs,
            method,
            options,
            omitRestBase,
        })
            .then(({ data, errors, version }: ServerResponse) => {
                setState({ loading: false, data, errors });
                if (Array.isArray(errors) && errors.length > 0) {
                    inputOptions?.displayError !== false && ServerResponseParser.displayErrors(errors);
                }
                return { data, errors, version };
            })
            .catch(({ errors, version }: ServerResponse) => {
                setState({ loading: false, data: undefined, errors });
                inputOptions?.displayError !== false && ServerResponseParser.displayErrors(errors);
                return { errors, version };
            });
        return result;
    }, [endpoint, inputs, method, options]);

    const [debouncedRefetch] = useDebounce<TInput | undefined>((input?: TInput) => refetch(input), debounceMs);
    useEffect(() => {
        debouncedRefetch(undefined);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [debouncedRefetch, ...deps]);

    return { ...state, refetch: debouncedRefetch };
}

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

export function uiPathToApi(pathname: string, apiBase = "/api/v2") {
    return apiBase + pathname;           // e.g. "/u/@matt" -> "/api/v2/u/@matt"
}
