import { TranslationKeyError } from "@local/shared";

export type Method = "GET" | "POST" | "PUT" | "DELETE";

export type ServerResponse<Output = any> = {
    errors?: {
        message: string;
        code?: TranslationKeyError;
    }[];
    data?: Output;
    version?: string;
};
export type ServerResponseWithTimestamp<T = any> = ServerResponse<T> & { __fetchTimestamp: number };

export type FetchInputOptions = {
    endpointOverride?: string,
    displayError?: boolean,
    onError?: (errors: ServerResponse["errors"]) => unknown,
}

export type LazyRequestWithResult<TInput extends Record<string, any> | undefined, TData> = (
    input?: TInput,
    inputOptions?: FetchInputOptions,
) => Promise<ServerResponse<TData>>;
