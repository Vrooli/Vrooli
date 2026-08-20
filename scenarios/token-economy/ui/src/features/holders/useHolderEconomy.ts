import { useQuery } from "@tanstack/react-query";
import { EventKind, type CatalogEntry, type Event } from "@vrooli/proto-types/token-economy/v1/access/access_pb";

import { holderClient } from "../../api/tokenEconomy";

export const holderEconomyKey = ["token-economy", "holder", "economy"] as const;
export const holderCatalogKey = ["token-economy", "holder", "catalog"] as const;

export function useHolderEconomy() {
  return useQuery({ queryKey: holderEconomyKey, queryFn: () => holderClient.viewEconomy({}) });
}

export function useHolderCatalog() {
  return useQuery({ queryKey: holderCatalogKey, queryFn: () => holderClient.browseCatalog({}) });
}

export function eligibleGrantId(entry: CatalogEntry, events: Event[]) {
  const credit = events.find(
    (event) =>
      event.kind === EventKind.CREDIT &&
      event.tokenTypeId === entry.tokenTypeId &&
      event.causeReference.startsWith("grant:"),
  );
  return credit?.causeReference.slice("grant:".length) ?? "";
}

export function eventDate(event: Event) {
  if (!event.createdAt) return "";
  return new Date(Number(event.createdAt.seconds) * 1000).toLocaleString();
}
