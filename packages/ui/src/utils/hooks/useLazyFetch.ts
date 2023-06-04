import { fetchData } from "api/utils";
import { useCallback, useEffect, useRef, useState } from "react";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    error: any;
};

type UseLazyFetchProps<TInput extends Record<string, any>, TData> = {
    endpoint: string | undefined;
    inputs?: TInput;
    method?: "GET" | "POST" | "PUT";
    options?: RequestInit;
};

/**
 * `useCustomLazyFetch` is a custom React hook for making HTTP requests.
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
export function useLazyFetch<TInput extends Record<string, any>, TData>({
    endpoint,
    inputs = {} as TInput,
    method = "GET",
    options = {} as RequestInit,
}: UseLazyFetchProps<TInput, TData>): [
        (input?: TInput) => void,
        RequestState<TData>
    ] {
    const [state, setState] = useState<RequestState<TData>>({
        loading: false,
        data: undefined,
        error: null,
    });

    // Store the latest endpoint, method, options and inputs in a ref
    const fetchParamsRef = useRef({ endpoint, method, options, inputs });
    useEffect(() => {
        fetchParamsRef.current = { endpoint, method, options, inputs };
    }, [endpoint, method, options, inputs]); // This will update the ref each time endpoint, method, options or inputs change

    const makeRequest = useCallback((input?: TInput) => {
        if (!fetchParamsRef.current.endpoint) {
            console.error("No endpoint provided to useLazyFetch");
            return;
        }
        // Update the inputs stored in the ref if a new input is provided
        if (input) {
            fetchParamsRef.current.inputs = input;
        }

        setState(s => ({ ...s, loading: true }));

        fetchData(fetchParamsRef.current as any)
            .then(response => response.json())
            .then(data => {
                console.log("useLazyFetch data", data);
                setState({ loading: false, data, error: null });
                // return { data };
            })
            .catch(error => {
                console.log("useLazyFetch error", error);
                setState({ loading: false, data: undefined, error });
                // return { error };
            });
    }, []);

    return [makeRequest, state];
}
