import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { nodesClient, type Node } from "../../api/nodes";

/** Canonical react-query key for the owner's fleet node list. */
export const NODES_QUERY_KEY = ["fleet", "nodes"] as const;

/**
 * List the owner's fleet nodes, with the live presence overlay the server
 * stamps at read time. Owner-gated; the management surface renders an error
 * (handled by the panel) when the owner token is absent.
 */
export function useNodesQuery() {
  return useQuery({
    queryKey: NODES_QUERY_KEY,
    queryFn: async (): Promise<Node[]> => {
      const resp = await nodesClient.listNodes({});
      return resp.nodes;
    },
  });
}

/**
 * Revoke a node — severs it atomically server-side (registry RevokeNode;
 * Phase 2 also destroys credentials + kills its channel). Refreshes the list
 * on success so the row flips to REVOKED immediately.
 */
export function useRevokeNodeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => nodesClient.revokeNode({ id }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: NODES_QUERY_KEY }),
  });
}

export type { Node };
