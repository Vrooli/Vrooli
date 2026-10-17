import { createContext, useContext } from "react";
import type { BacklogItem } from "../../types";

export type ItemsByKey = ReadonlyMap<string, BacklogItem>;

export const EMPTY_BACKLOG_ITEM_LOOKUP: ItemsByKey = new Map();

export const BacklogItemsContext = createContext<ItemsByKey>(EMPTY_BACKLOG_ITEM_LOOKUP);

export function useBacklogItemLookup(): ItemsByKey {
  return useContext(BacklogItemsContext);
}
