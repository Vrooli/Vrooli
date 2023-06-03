import { stringifySearchParams } from "@local/shared";
import { useCallback, useEffect, useRef, useState } from "react";

type RequestState<TData> = {
    loading: boolean;
    data: TData | undefined;
    error: any;
};

// Determine origin of API server
let urlBase: string;
// If running locally
if (window.location.host.includes("localhost") || window.location.host.includes("192.168.0.")) {
    urlBase = `http://${window.location.hostname}:${import.meta.env.VITE_PORT_SERVER ?? "5329"}/api/v2/rest`;
}
// If running on server
else {
    urlBase = import.meta.env.VITE_SERVER_URL && import.meta.env.VITE_SERVER_URL.length > 0 ?
        `${import.meta.env.VITE_SERVER_URL}/v2` :
        `http://${import.meta.env.VITE_SITE_IP}:${import.meta.env.VITE_PORT_SERVER ?? "5329"}/api/v2/rest`;
}

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
export function useLazyFetch<TInput extends Record<string, any>, TData>(
    endpoint: string | undefined,
    initialInputs: TInput = {} as TInput,
    method: "GET" | "POST" | "PUT" = "GET",
    options: RequestInit = {},
): [
        (input?: TInput) => void,
        RequestState<TData>
    ] {
    const [state, setState] = useState<RequestState<TData>>({
        loading: false,
        data: undefined,
        error: null,
    });

    // Store the latest endpoint, method, options and inputs in a ref
    const fetchParamsRef = useRef({ endpoint, method, options, inputs: initialInputs });
    useEffect(() => {
        fetchParamsRef.current = { endpoint, method, options, inputs: initialInputs };
    }, [endpoint, method, options, initialInputs]); // This will update the ref each time endpoint, method, options or inputs change

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

        let url = `${urlBase}${fetchParamsRef.current.endpoint}`;
        let body: string | null = null;

        if (fetchParamsRef.current.method === "GET") {
            if (Object.keys(fetchParamsRef.current.inputs).length !== 0) {
                url += `?${stringifySearchParams(fetchParamsRef.current.inputs)}`;
            }
        } else if (["POST", "PUT"].includes(fetchParamsRef.current.method)) {
            body = JSON.stringify({ input: fetchParamsRef.current.inputs });
        }

        fetch(url, {
            ...fetchParamsRef.current.options,
            method: fetchParamsRef.current.method,
            headers: {
                ...fetchParamsRef.current.options.headers,
                "Content-Type": "application/json",
            },
            body,
            credentials: "include",
        })
            .then(response => response.json())
            .then(data => {
                console.log("useLazyFetch data", data);
                setState({ loading: false, data, error: null });
            })
            .catch(error => {
                console.log("useLazyFetch error", error);
                setState({ loading: false, data: undefined, error });
            });
    }, []);

    return [makeRequest, state];
}
