import { vi } from 'vitest';
import { ApiError, type ApiErrorType } from '../api/common';

/**
 * Factory for creating a mock fetch function with proper type handling.
 */
export function createFetchMock() {
  return vi.fn() as ReturnType<typeof vi.fn> & {
    mockResolvedValue: (value: MockFetchResponse) => ReturnType<typeof vi.fn>;
    mockRejectedValue: (error: Error) => ReturnType<typeof vi.fn>;
    mockImplementation: (fn: () => Promise<MockFetchResponse>) => ReturnType<typeof vi.fn>;
  };
}

export interface MockFetchResponse {
  ok: boolean;
  status?: number;
  statusText?: string;
  json?: () => Promise<unknown>;
  text?: () => Promise<string>;
}

/**
 * Mock response factories for common API scenarios.
 */
export const mockResponses = {
  /**
   * Create a successful JSON response.
   */
  success<T>(data: T, status = 200): MockFetchResponse {
    return {
      ok: true,
      status,
      statusText: 'OK',
      json: async () => data,
      text: async () => JSON.stringify(data),
    };
  },

  /**
   * Create an error response with JSON body.
   */
  error(status: number, message: string, errorKey = 'error'): MockFetchResponse {
    const body = { [errorKey]: message };
    return {
      ok: false,
      status,
      statusText: getStatusText(status),
      json: async () => body,
      text: async () => JSON.stringify(body),
    };
  },

  /**
   * Create a network error (fetch rejects).
   */
  networkError(): TypeError {
    return new TypeError('Failed to fetch');
  },

  /**
   * Create a timeout abort error.
   */
  timeout(): Error {
    const error = new Error('Aborted');
    error.name = 'AbortError';
    return error;
  },

  /**
   * Create an empty response (e.g., for 204 No Content).
   */
  empty(status = 204): MockFetchResponse {
    return {
      ok: true,
      status,
      statusText: status === 204 ? 'No Content' : 'OK',
      json: async () => undefined,
      text: async () => '',
    };
  },

  /**
   * Create a response with validation errors.
   */
  validationError(errors: Record<string, string>): MockFetchResponse {
    return {
      ok: false,
      status: 400,
      statusText: 'Bad Request',
      json: async () => ({ error: 'Validation failed', errors }),
      text: async () => JSON.stringify({ error: 'Validation failed', errors }),
    };
  },

  /**
   * Create an unauthorized (401) response.
   */
  unauthorized(message = 'Unauthorized'): MockFetchResponse {
    return mockResponses.error(401, message);
  },

  /**
   * Create a forbidden (403) response.
   */
  forbidden(message = 'Forbidden'): MockFetchResponse {
    return mockResponses.error(403, message);
  },

  /**
   * Create a not found (404) response.
   */
  notFound(message = 'Not found'): MockFetchResponse {
    return mockResponses.error(404, message);
  },

  /**
   * Create a rate limited (429) response.
   */
  rateLimited(retryAfter = 60): MockFetchResponse {
    return {
      ok: false,
      status: 429,
      statusText: 'Too Many Requests',
      json: async () => ({ error: 'Rate limited', retry_after: retryAfter }),
      text: async () => JSON.stringify({ error: 'Rate limited', retry_after: retryAfter }),
    };
  },

  /**
   * Create a server error (500) response.
   */
  serverError(message = 'Internal Server Error'): MockFetchResponse {
    return mockResponses.error(500, message);
  },
};

/**
 * Factory for creating ApiError instances for testing.
 */
export function createApiErrorMock(
  type: ApiErrorType,
  message?: string,
  status?: number,
  userMessage?: string
): ApiError {
  const defaultMessages: Record<ApiErrorType, string> = {
    network: 'Network error',
    timeout: 'Request timed out',
    unauthorized: 'Unauthorized',
    forbidden: 'Forbidden',
    not_found: 'Not found',
    validation: 'Validation error',
    rate_limited: 'Rate limited',
    server_error: 'Server error',
    unknown: 'Unknown error',
  };

  const defaultStatuses: Partial<Record<ApiErrorType, number>> = {
    unauthorized: 401,
    forbidden: 403,
    not_found: 404,
    validation: 400,
    rate_limited: 429,
    server_error: 500,
  };

  return new ApiError(
    message ?? defaultMessages[type],
    type,
    status ?? defaultStatuses[type],
    userMessage
  );
}

/**
 * Helper to install fetch mock on globalThis.
 */
export function installFetchMock(mockFetch: ReturnType<typeof createFetchMock>): void {
  globalThis.fetch = mockFetch as unknown as typeof fetch;
}

/**
 * Helper to get status text for HTTP status codes.
 */
function getStatusText(status: number): string {
  const statusTexts: Record<number, string> = {
    200: 'OK',
    201: 'Created',
    204: 'No Content',
    400: 'Bad Request',
    401: 'Unauthorized',
    403: 'Forbidden',
    404: 'Not Found',
    422: 'Unprocessable Entity',
    429: 'Too Many Requests',
    500: 'Internal Server Error',
    502: 'Bad Gateway',
    503: 'Service Unavailable',
  };
  return statusTexts[status] ?? 'Unknown Status';
}

/**
 * Helper to assert that an error is an ApiError with specific properties.
 */
export function expectApiError(
  error: unknown,
  expectedType: ApiErrorType,
  expectedStatus?: number
): asserts error is ApiError {
  if (!(error instanceof ApiError)) {
    throw new Error(`Expected ApiError but got ${error?.constructor?.name ?? typeof error}`);
  }
  if (error.type !== expectedType) {
    throw new Error(`Expected ApiError type "${expectedType}" but got "${error.type}"`);
  }
  if (expectedStatus !== undefined && error.status !== expectedStatus) {
    throw new Error(`Expected ApiError status ${expectedStatus} but got ${error.status}`);
  }
}

/**
 * Create a mock for window.open, commonly used for export/download functions.
 * Returns the mock function and a cleanup function.
 */
export function createWindowOpenMock(): {
  mock: ReturnType<typeof vi.fn>;
  restore: () => void;
} {
  const originalOpen = window.open;
  const mock = vi.fn();
  window.open = mock;

  return {
    mock,
    restore: () => {
      window.open = originalOpen;
    },
  };
}

/**
 * Create a mock for URL.createObjectURL, used for blob downloads.
 */
export function createObjectURLMock(): {
  mock: ReturnType<typeof vi.fn>;
  restore: () => void;
} {
  const originalCreateObjectURL = URL.createObjectURL;
  const originalRevokeObjectURL = URL.revokeObjectURL;
  const mock = vi.fn(() => 'blob:test-url');
  const revokeMock = vi.fn();
  URL.createObjectURL = mock;
  URL.revokeObjectURL = revokeMock;

  return {
    mock,
    restore: () => {
      URL.createObjectURL = originalCreateObjectURL;
      URL.revokeObjectURL = originalRevokeObjectURL;
    },
  };
}

/**
 * Create a mock anchor element for testing download triggers.
 */
export function createDownloadLinkMock(): {
  element: HTMLAnchorElement;
  clickSpy: ReturnType<typeof vi.fn>;
  appendChildSpy: ReturnType<typeof vi.fn>;
  removeChildSpy: ReturnType<typeof vi.fn>;
  restore: () => void;
} {
  const clickSpy = vi.fn();
  const appendChildSpy = vi.fn();
  const removeChildSpy = vi.fn();

  const originalCreateElement = document.createElement.bind(document);
  const originalAppendChild = document.body.appendChild.bind(document.body);
  const originalRemoveChild = document.body.removeChild.bind(document.body);

  const element = originalCreateElement('a') as HTMLAnchorElement;
  element.click = clickSpy;

  vi.spyOn(document, 'createElement').mockImplementation((tag) => {
    if (tag === 'a') {
      return element;
    }
    return originalCreateElement(tag);
  });

  vi.spyOn(document.body, 'appendChild').mockImplementation((node) => {
    appendChildSpy(node);
    return node as unknown as HTMLAnchorElement;
  });

  vi.spyOn(document.body, 'removeChild').mockImplementation((node) => {
    removeChildSpy(node);
    return node as unknown as HTMLAnchorElement;
  });

  return {
    element,
    clickSpy,
    appendChildSpy,
    removeChildSpy,
    restore: () => {
      vi.restoreAllMocks();
    },
  };
}
