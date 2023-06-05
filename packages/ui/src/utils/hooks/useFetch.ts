import { fetchData, Method, ServerResponse } from "api";
import { useCallback, useEffect, useRef, useState } from "react";
import { useDebounce } from "./useDebounce";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    errors: ServerResponse["errors"] | undefined;
};

type UseFetchProps<TInput extends Record<string, any> | undefined, TData> = {
    endpoint: string | undefined;
    inputs?: TInput;
    method?: Method;
    options?: RequestInit;
    debounceMs?: number;
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
    debounceMs = 0,
}: UseFetchProps<TInput, TData>): RequestState<TData> & { refetch: (input?: TInput) => void } {
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

    const refetch = useCallback<(input?: TInput) => Promise<ServerResponse<TData>>>(async (input?: TInput) => {
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

    const debouncedRefetch = useDebounce<TInput | undefined>((input?: TInput) => refetch(input), debounceMs);

    useEffect(() => {
        debouncedRefetch(undefined);
    }, [debouncedRefetch]); // This will automatically make the request when the component mounts or the inputs change

    return { ...state, refetch: debouncedRefetch };
}
