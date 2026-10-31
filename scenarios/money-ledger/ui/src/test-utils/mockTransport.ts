import type { SurfaceState, SurfaceStateInput } from "../hooks/useSurfaceState";

export interface MockTransport {
  surfaceInput: SurfaceStateInput;
  request<T>(value: T): Promise<T>;
}

export function createMockTransport(state: SurfaceState): MockTransport {
  const inputs: Record<SurfaceState, SurfaceStateInput> = {
    loading: { query: { isLoading: true } },
    empty: { empty: true },
    ready: {},
    partial: { availability: { partial: true } },
    stale: { availability: { stale: true } },
    saving: { mutation: { isPending: true } },
    success: { mutation: { isSuccess: true } },
    "validation-error": { validationError: "Review the highlighted fields." },
    "request-error": { query: { isError: true, error: new Error("request failed") } },
    syncing: { syncing: true },
    refreshing: { refreshing: true },
    retrying: { retrying: true },
    offline: { availability: { offline: true } },
    "permission-denied": { availability: { permissionDenied: true } },
  };
  return {
    surfaceInput: inputs[state],
    request: <T>(value: T) => Promise.resolve(value),
  };
}
