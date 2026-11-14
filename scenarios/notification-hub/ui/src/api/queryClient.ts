import { QueryClient } from "@tanstack/react-query";

import { isUnauthenticated } from "../lib/apiError";

/** A transient failure is tried once more; a refused session is not tried again, since retrying cannot sign anyone in. */
export const retryTransientOnce = (failureCount: number, error: unknown): boolean =>
  failureCount < 1 && !isUnauthenticated(error);

/**
 * The app's query client. React Query's own default (three retries with
 * growing delays) keeps a skeleton on screen for seven seconds before a failure
 * is admitted; here a failure is shown after one quick retry.
 */
export const createQueryClient = () =>
  new QueryClient({ defaultOptions: { queries: { retry: retryTransientOnce } } });
