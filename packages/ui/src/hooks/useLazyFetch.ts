import { displayServerErrors, fetchData, Method, ServerResponse } from "api";
import { useCallback, useEffect, useRef, useState } from "react";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    errors: ServerResponse["errors"] | undefined;
};

export type UseLazyFetchProps<TInput extends Record<string, any> | undefined> = {
    endpoint?: string | undefined;
    inputs?: TInput;
    method?: Method;
    options?: RequestInit;
};

export type FetchInputOptions = {
    endpointOverride?: string,
    displayError?: boolean,
    onError?: (errors: ServerResponse["errors"]) => unknown,
}

export type MakeLazyRequest<TInput extends Record<string, any> | undefined, TData> = (
    input?: TInput,
    inputOptions?: FetchInputOptions,
) => Promise<ServerResponse<TData>>;

/**
 * Custom React hook for making HTTP requests.
 * It is "lazy" in the sense that it doesn't run automatically when the component mounts,
 * but instead provides a function that can be called to initiate the request.
 * This hook handles the lifecycle of the request and provides updates on its status through its return value.
 *
 * @param endpoint - The URL to make the request to.
 * @param method - The HTTP method to use for the request (defaults to 'GET').
 * @param options - Additional options to pass to the `fetch` function.
 * @returns A tuple where the first element is a function
 * to initiate the request and the second element is an object representing the current state of the request.
 */
export function useLazyFetch<TInput extends Record<string, any> | undefined, TData>({
    endpoint,
    inputs = {} as TInput,
    method = "GET",
    options = {} as RequestInit,
}: UseLazyFetchProps<TInput>): [
        MakeLazyRequest<TInput, TData>,
        RequestState<TData>
    ] {
    const [state, setState] = useState<RequestState<TData>>({
        loading: false,
        data: undefined,
        errors: undefined,
    });

    // Store the latest endpoint, method, options and inputs in a ref
    const fetchParamsRef = useRef({ endpoint, method, options, inputs });
    useEffect(() => {
        fetchParamsRef.current = { endpoint, method, options, inputs };
    }, [endpoint, method, options, inputs]); // This will update the ref each time endpoint, method, options or inputs change

    const getData = useCallback<MakeLazyRequest<TInput, TData>>(async (input, inputOptions) => {
        // Update the inputs stored in the ref if a new input is provided
        if (input) {
            fetchParamsRef.current.inputs = input;
        }
        // Get fetch params from the ref
        const { endpoint, method, options, inputs } = fetchParamsRef.current;
        // Cancel if no endpoint is provided
        if (!endpoint && !inputOptions?.endpointOverride) {
            const message = "No endpoint provided to useLazyFetch";
            console.error(message);
            return { errors: [{ message }] };
        }
        // Update the state to loading
        setState(s => ({ ...s, loading: true }));
        // Helper function for handling response
        const handleResponse = ({ data, errors, version }: ServerResponse) => {
            console.log("got data", data, errors, version);
            setState({ loading: false, data, errors });
            if (Array.isArray(errors) && errors.length > 0) {
                if (inputOptions?.onError) {
                    inputOptions.onError(errors);
                }
                if (inputOptions?.displayError !== false) {
                    displayServerErrors(errors);
                }
            }
            return { data, errors, version };
        };
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
