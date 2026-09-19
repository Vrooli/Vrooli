export const SURFACE_STATES = [
  "loading",
  "empty",
  "ready",
  "partial",
  "stale",
  "saving",
  "success",
  "validation-error",
  "request-error",
  "syncing",
  "refreshing",
  "retrying",
  "offline",
  "permission-denied",
] as const;

export type SurfaceState = (typeof SURFACE_STATES)[number];

export interface SurfaceStateInput {
  query?: {
    isLoading?: boolean;
    isFetching?: boolean;
    isError?: boolean;
    error?: unknown;
  };
  mutation?: {
    isPending?: boolean;
    isSuccess?: boolean;
    isError?: boolean;
    error?: unknown;
  };
  availability?: {
    partial?: boolean;
    stale?: boolean;
    offline?: boolean;
    permissionDenied?: boolean;
  };
  empty?: boolean;
  validationError?: string;
  retrying?: boolean;
  syncing?: boolean;
  refreshing?: boolean;
}

export interface SurfaceStateResult {
  state: SurfaceState;
  reason: string;
}

function hasError(error: unknown) {
  return error !== undefined && error !== null;
}

export function deriveSurfaceState(input: SurfaceStateInput = {}): SurfaceStateResult {
  const query = input.query ?? {};
  const mutation = input.mutation ?? {};
  const availability = input.availability ?? {};

  if (availability.permissionDenied) {
    return { state: "permission-denied", reason: "You do not have permission to view this surface." };
  }
  if (availability.offline) {
    return { state: "offline", reason: "The service is offline. Reconnect and try again." };
  }
  if (input.validationError) {
    return { state: "validation-error", reason: input.validationError };
  }
  if (mutation.isError || hasError(mutation.error) || query.isError || hasError(query.error)) {
    return { state: "request-error", reason: "The request failed. Review the error and try again." };
  }
  if (input.retrying) {
    return { state: "retrying", reason: "Retrying the request." };
  }
  if (mutation.isPending) {
    return { state: "saving", reason: "Saving your changes." };
  }
  if (input.syncing) {
    return { state: "syncing", reason: "Synchronizing the latest data." };
  }
  if (input.refreshing) {
    return { state: "refreshing", reason: "Refreshing the latest data." };
  }
  if (query.isLoading) {
    return { state: "loading", reason: "Loading data." };
  }
  if (availability.stale) {
    return { state: "stale", reason: "This data may be out of date." };
  }
  if (availability.partial) {
    return { state: "partial", reason: "Some sources are unavailable." };
  }
  if (mutation.isSuccess) {
    return { state: "success", reason: "Your changes were saved." };
  }
  if (input.empty) {
    return { state: "empty", reason: "No records are available yet." };
  }
  if (query.isFetching) {
    return { state: "refreshing", reason: "Refreshing the latest data." };
  }
  return { state: "ready", reason: "Data is ready." };
}

export function useSurfaceState(input: SurfaceStateInput = {}): SurfaceStateResult {
  return deriveSurfaceState(input);
}
