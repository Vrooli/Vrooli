import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ItemKind, transferClient, type Item } from "../../api/transfer";

/** Canonical react-query key for the receive list. Shared so SSE invalidation and
 * the mutation `onSuccess` paths target the exact same cache entry. */
export const ITEMS_QUERY_KEY = ["transfer", "items"] as const;

/**
 * Fetch the items the calling device may pull (broadcast + directed-to-it +
 * originated). Server returns newest-first; filtering/sorting/search happen
 * client-side in the receive panel so view changes don't re-hit the network.
 */
export function useItemsQuery(enabled: boolean) {
  return useQuery({
    queryKey: ITEMS_QUERY_KEY,
    queryFn: async (): Promise<Item[]> => {
      const resp = await transferClient.listItems({});
      return resp.items;
    },
    enabled,
  });
}

/** Delete an item (owner-origin items expose this), invalidating the list. */
export function useDeleteItemMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => transferClient.deleteItem({ id }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ITEMS_QUERY_KEY });
    },
  });
}

export { ItemKind };
export type { Item };
