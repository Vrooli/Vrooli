import { fetchData, Method, ServerResponse } from "api";
import { useCallback, useEffect, useRef, useState } from "react";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    errors: ServerResponse["errors"] | undefined;
};

type UseLazyFetchProps<TInput extends Record<string, any>, TData> = {
    endpoint: string | undefined;
    inputs?: TInput;
    method?: Method;
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
        (input?: TInput) => Promise<ServerResponse<TData>>,
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

    const makeRequest = useCallback<(input?: TInput) => Promise<ServerResponse<TData>>>(async (input?: TInput) => {
        if (!fetchParamsRef.current.endpoint) {
            const message = "No endpoint provided to useLazyFetch";
            console.error(message);
            return { errors: [{ message }] };
        }
        // Update the inputs stored in the ref if a new input is provided
        if (input) {
            fetchParamsRef.current.inputs = input;
        }

        setState(s => ({ ...s, loading: true }));

        const result = await fetchData(fetchParamsRef.current as any)
            .then(({ data, errors, version }: ServerResponse) => {
                console.log("useLazyFetch data", data, errors);
                setState({ loading: false, data, errors });
                return { data, errors, version };
            })
            .catch(({ errors, version }: ServerResponse) => {
                console.log("useLazyFetch error", errors);
                setState({ loading: false, data: undefined, errors });
                return { errors, version };
            });
        return result;
    }, []);

    return [makeRequest, state];
}
