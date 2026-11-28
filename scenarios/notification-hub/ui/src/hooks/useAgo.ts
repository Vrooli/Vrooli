import { formatAgo } from "../lib/ago";
import { useNow } from "./useNow";

/** "12 minutes ago" for a timestamp, kept current; null when the value is not a time. */
export function useAgo(value: string | undefined): string | null {
  return formatAgo(value, useNow());
}
