import { dataFetchingConfig } from "../config";

export type LoadStatus = "idle" | "loading" | "success" | "error";

export interface FetchGuard {
  lastFetchedAt: number | null;
  hasData: boolean;
  force?: boolean;
}

export function shouldRefetch({ lastFetchedAt, hasData, force }: FetchGuard): boolean {
  if (force) return true;
  if (!lastFetchedAt) return true;
  if (!hasData) return true;
  return Date.now() - lastFetchedAt > dataFetchingConfig.staleTimeMs;
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export async function fetchWithRetry<T>(operation: () => Promise<T>): Promise<T> {
  const maxRetries = Math.max(0, dataFetchingConfig.retryCount);
  let attempt = 0;

  while (true) {
    try {
      return await operation();
    } catch (error) {
      if (attempt >= maxRetries) {
        throw error;
      }
      const delayMs = dataFetchingConfig.retryDelayMs * Math.pow(2, attempt);
      attempt += 1;
      if (delayMs > 0) {
        await sleep(delayMs);
      }
    }
  }
}
