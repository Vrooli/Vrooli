export interface ApiJsonRequestOptions<TResult> extends RequestInit {
  /**
   * Extracts the final value from the parsed JSON payload.
   * If omitted, the raw parsed value is returned.
   */
  selector?: (payload: unknown) => TResult;
  /**
   * Override the default error message when the response is not ok.
   */
  errorMessage?: string;
}

export interface ApiVoidRequestOptions extends RequestInit {
  /**
   * Override the default error message when the response is not ok.
   */
  errorMessage?: string;
}

function buildError(url: string, status: number, override?: string): Error {
  const message = override ?? `Request to ${url} failed with status ${status}`;
  const error = new Error(message);
  (error as Error & { status?: number }).status = status;
  return error;
}

export async function apiJsonRequest<TResult = unknown>(
  url: string,
  options: ApiJsonRequestOptions<TResult> = {},
): Promise<TResult> {
  const { selector, errorMessage, ...fetchOptions } = options;
  const response = await fetch(url, fetchOptions);

  if (!response.ok) {
    throw buildError(url, response.status, errorMessage);
  }

  const payload = (await response.json()) as unknown;
  return selector ? selector(payload) : (payload as TResult);
}

export async function apiVoidRequest(
  url: string,
  options: ApiVoidRequestOptions = {},
): Promise<void> {
  const { errorMessage, ...fetchOptions } = options;
  const response = await fetch(url, fetchOptions);

  if (!response.ok) {
    throw buildError(url, response.status, errorMessage);
  }
}
