import { ServerResponse } from "api";

export type FetchInputOptions = {
    endpointOverride?: string,
    displayError?: boolean,
    onError?: (errors: ServerResponse["errors"]) => unknown,
}

export type LazyRequestWithResult<TInput extends Record<string, any> | undefined, TData> = (
    input?: TInput,
    inputOptions?: FetchInputOptions,
) => Promise<ServerResponse<TData>>;
