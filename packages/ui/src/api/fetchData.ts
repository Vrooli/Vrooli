import { ParseSearchParamsResult, stringifySearchParams } from "@local/shared";
import { Method, ServerResponseWithTimestamp } from "api";
import { apiUrlBase, restBase } from "utils/consts";
import { invalidateAIConfigCache } from "./ai";

const SERVER_VERSION_CACHE_KEY = "serverVersionCache";

type FetchDataProps<Input extends object | undefined> = {
    endpoint: string;
    method: Method;
    inputs: Input;
    options?: RequestInit;
    /** Omits rest endpoint base from URL */
    omitRestBase?: boolean;
};

/**
 * `fetchData` is a helper function for making HTTP requests. 
 * It creates the URL, headers, and body for the request, and then makes the request.
 *
 * Variables in the `endpoint` string (prefixed with ":") are replaced with their corresponding values from `inputs`.
 * After replacement, if any `inputs` remain and the method is "GET", they are stringified and appended as query parameters.
 * For "POST" and "PUT" methods, the remaining `inputs` are included in the request body.
 *
 * @param params - An object containing the parameters for the request.
 *                 `params.endpoint`: The URL to make the request to, possibly including variables like ":id".
 *                 `params.method`: The HTTP method to use for the request ("GET", "POST", or "PUT").
 *                 `params.inputs`: An object whose keys are input parameter names and values are input parameter values.
 *                 `params.options`: Additional options to pass to the `fetch` function.
 *
 * @returns A promise that resolves to the Response from the fetch request.
 *
 * @throws Will throw an error if the fetch request fails.
 */
export async function fetchData<Input extends object | undefined, Output>({
    endpoint,
    method,
    inputs,
    options,
    omitRestBase = false,
}: FetchDataProps<Input>): Promise<ServerResponseWithTimestamp<Output>> {

    // Replace variables in the endpoint with their values from inputs.
    if (inputs !== undefined) {
        endpoint = endpoint.replace(/:([a-zA-Z]+)/g, (_, key) => {
            const value = inputs[key];
            delete inputs[key];  // remove substituted values from inputs
            return value;
        });
    }

    let url = `${apiUrlBase}${omitRestBase ? "" : restBase}${endpoint}`;
    let body: string | FormData | null = null;

    // GET requests should have their inputs converted to query parameters.
    if (method === "GET") {
        if (inputs !== undefined && Object.keys(inputs).length !== 0) {
            url += `${stringifySearchParams(inputs as ParseSearchParamsResult)}`;
        }
    }
    // Other requests should have their inputs converted to JSON and sent in the body.
    else if (["DELETE", "POST", "PUT"].includes(method)) {
        if (inputs instanceof FormData) {
            body = inputs;
        } else {
            body = JSON.stringify(inputs);
        }
    }

    const finalOptions: RequestInit = {
        ...options,
        method,
        headers: {
            ...options?.headers,
            // Only set the Content-Type to "application/json" if the body is not FormData.
            ...(!(inputs instanceof FormData) && { "Content-Type": "application/json" }),
        },
        body,
        credentials: "include",
    };

    // Capture the current date and time, which can be useful to determine the order of events. 
    // For example, if you switch quickly between search types, you want to make sure that the 
    // displayed results are from the fetch that was CALLED last, and not the fetch that was RESOLVED last.
    const __fetchTimestamp = Date.now();

    return fetch(url, finalOptions)
        .then(response => response.json())
        .then(data => {
            // Before returning the data, we'll check if the server version has changed and 
            // invalidate things accordingly.
            const existingVersion = localStorage.getItem(SERVER_VERSION_CACHE_KEY);
            const currentVersion = data.version;
            if (existingVersion !== currentVersion) {
                localStorage.setItem(SERVER_VERSION_CACHE_KEY, currentVersion);
                invalidateAIConfigCache();
            }
            return { ...data, __fetchTimestamp };
        });
}

