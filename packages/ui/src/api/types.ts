/* c8 ignore start */
// AI_CHECK: TYPE_SAFETY=36 | LAST: 2025-06-28
import { type ServerResponse } from "@vrooli/shared";

export type ServerResponseWithTimestamp<T = unknown> = ServerResponse<T> & { __fetchTimestamp: number };

export type FetchInputOptions = {
    endpointOverride?: string,
    displayError?: boolean,
    onError?: (errors: ServerResponse["errors"]) => unknown,
}

export type LazyRequestWithResult<TInput extends Record<string, unknown> | undefined, TData> = (
    input?: TInput,
    inputOptions?: FetchInputOptions,
) => Promise<ServerResponse<TData>>;
