import { stringifySearchParams } from "@local/shared";
import { useCallback, useState } from "react";

type RequestState<TData> = {
    loading: boolean;
    data: TData | null;
    error: any;
};

// Determine origin of API server
let urlBase: string;
// If running locally
if (window.location.host.includes("localhost") || window.location.host.includes("192.168.0.")) {
    urlBase = `http://${window.location.hostname}:${import.meta.env.VITE_PORT_SERVER ?? "5329"}/api/v2`;
}
// If running on server
else {
    urlBase = import.meta.env.VITE_SERVER_URL && import.meta.env.VITE_SERVER_URL.length > 0 ?
        `${import.meta.env.VITE_SERVER_URL}/v2` :
        `http://${import.meta.env.VITE_SITE_IP}:${import.meta.env.VITE_PORT_SERVER ?? "5329"}/api/v2`;
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
export function useLazyFetch<TInput extends Record<string, any>, TData>(endpoint: string, method: "GET" | "POST" = "GET", options: RequestInit = {}): [
    (input: TInput) => void,
    RequestState<TData>
] {
    const [state, setState] = useState<RequestState<TData>>({
        loading: false,
        data: null,
        error: null,
    });

    const makeRequest = useCallback((input: TInput) => {
        setState(s => ({ ...s, loading: true }));

        let url = `${urlBase}${endpoint}`;
        let body: string | null = null;

        if (method === "GET") {
            if (Object.keys(input).length !== 0) {
                url += `?${stringifySearchParams(input)}`;
            }
        } else if (method === "POST") {
            body = JSON.stringify({ input });
        }

        fetch(url, {
            ...options,
            method,
            headers: {
                ...options.headers,
                "Content-Type": "application/json",
            },
            body,
        })
            .then(response => response.json())
            .then(data => {
                setState({ loading: false, data, error: null });
            })
            .catch(error => {
                setState({ loading: false, data: null, error });
            });
    }, [endpoint, method, options]);

    return [makeRequest, state];
}
