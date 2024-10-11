import { errorToMessage } from "api/errorParser";
import { fetchData } from "api/fetchData";
import { LazyRequestWithResult, Method, ServerResponse } from "api/types";
import { useCallback, useEffect, useState } from "react";
import { PubSub } from "utils/pubsub";
import { useDebounce } from "./useDebounce";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    errors: ServerResponse["errors"] | undefined;
};

type UseFetchProps<TInput extends Record<string, any> | undefined, TData> = {
    debounceMs?: number;
    endpoint: string | undefined;
    inputs?: TInput;
    method?: Method;
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

    const displayErrors = (errors?: ServerResponse["errors"]) => {
        if (!errors) return;
        for (const error of errors) {
            const message = errorToMessage({ errors: [error] }, ["en"]);
            PubSub.get().publish("snack", { message, severity: "Error" });
        }
    };

    const refetch = useCallback<LazyRequestWithResult<TInput, TData>>(async (input, inputOptions) => {
        if (!endpoint && !inputOptions?.endpointOverride) {
            const message = "No endpoint provided to useLazyFetch";
            console.error(message);
            return { errors: [{ message }] };
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
                    inputOptions?.displayError !== false && displayErrors(errors);
                }
                return { data, errors, version };
            })
            .catch(({ errors, version }: ServerResponse) => {
                setState({ loading: false, data: undefined, errors });
                inputOptions?.displayError !== false && displayErrors(errors);
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
