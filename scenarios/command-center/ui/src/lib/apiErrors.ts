export class ApiRequestError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = code;
  }
}

export async function apiRequestError(response: Response, operation: string): Promise<ApiRequestError> {
  let code: string | undefined;
  try {
    const body: unknown = await response.clone().json();
    if (body && typeof body === "object" && "error" in body && typeof body.error === "string") {
      code = body.error;
    }
  } catch {
    // Keep transport errors useful even when the server did not return JSON.
  }
  return new ApiRequestError(`${operation}: ${response.status} ${response.statusText}`, response.status, code);
}
