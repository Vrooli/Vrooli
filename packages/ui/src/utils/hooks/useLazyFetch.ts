import { fetchData, Method, ServerResponse } from "api";
import { useCallback, useEffect, useRef, useState } from "react";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    errors: ServerResponse["errors"] | undefined;
};

type UseLazyFetchProps<TInput extends Record<string, any> | undefined, TData> = {
    endpoint?: string | undefined;
    inputs?: TInput;
    method?: Method;
    options?: RequestInit;
};

export type MakeLazyRequest<TInput extends Record<string, any> | undefined, TData> = (input?: TInput, endpointOverride?: string) => Promise<ServerResponse<TData>>;

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
}: UseLazyFetchProps<TInput, TData>): [
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

    const makeRequest = useCallback<MakeLazyRequest<TInput, TData>>(async (input?: TInput, endpointOverride?: string) => {
        // Update the inputs stored in the ref if a new input is provided
        if (input) {
            fetchParamsRef.current.inputs = input;
        }
        // Get fetch params from the ref
        const { endpoint, method, options, inputs } = fetchParamsRef.current;
        // Cancel if no endpoint is provided
        if (!endpoint && !endpointOverride) {
            const message = "No endpoint provided to useLazyFetch";
            console.error(message);
            return { errors: [{ message }] };
        }
        // Update the state to loading
        setState(s => ({ ...s, loading: true }));
        // Make the request
        console.log("fetching data", endpointOverride || endpoint, inputs);
        const result = await fetchData({
            endpoint: endpointOverride || endpoint as string,
            inputs,
            method,
            options,
        })
            .then(({ data, errors, version }: ServerResponse) => {
                setState({ loading: false, data, errors });
                return { data, errors, version };
            })
            .catch(({ errors, version }: ServerResponse) => {
                setState({ loading: false, data: undefined, errors });
                return { errors, version };
            });
        return result;
    }, []);

    return [makeRequest, state];
}
