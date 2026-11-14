import type { QueryClient } from "@tanstack/react-query";
import { recipientsClient } from "../../api/notifications";

/**
 * The owner-scoped reads the settings screens share. One definition per read
 * keeps the cache key and the fetcher together, so a summary row, a screen and
 * a save's read-back all talk about the same cached value.
 */
export const devicesQuery = {
  queryKey: ["recipient-devices"] as const,
  queryFn: () => recipientsClient.listDevices({}),
};

export const channelAddressesQuery = {
  queryKey: ["recipient-channel-addresses"] as const,
  queryFn: () => recipientsClient.listChannelAddresses({}),
};

export const quietWindowsQuery = {
  queryKey: ["quiet-windows"] as const,
  queryFn: () => recipientsClient.listQuietWindows({}),
};

export const escalationChainQuery = {
  queryKey: ["escalation-chain"] as const,
  queryFn: () => recipientsClient.getEscalationChain({}),
};

/** Cache roots holding one owner's data; dropped when a different owner signs in. */
export const OWNER_QUERY_ROOTS = ["recipient-devices", "recipient-channel-addresses", "quiet-windows", "escalation-chain", "notification-catalog", "device-apps"] as const;

/**
 * Read what the Hub holds right now, ignoring the cache and any read already
 * in flight. A save trusts this read, not the write's own answer: a response
 * can be lost while the change lands.
 */
export async function readFresh<T>(client: QueryClient, query: { queryKey: readonly unknown[]; queryFn: () => Promise<T> }): Promise<T> {
  await client.cancelQueries({ queryKey: query.queryKey, exact: true });
  return client.fetchQuery({ queryKey: query.queryKey, queryFn: query.queryFn, staleTime: 0 });
}
