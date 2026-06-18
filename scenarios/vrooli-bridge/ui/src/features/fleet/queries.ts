import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { nodesClient, type Node } from "../../api/nodes";
import { queueClient, type NodeQueue } from "../../api/queue";
import { pairingClient, type IssuePairingCodeResponse } from "../../api/pairing";

/** Canonical react-query key for the owner's fleet node list. */
export const NODES_QUERY_KEY = ["fleet", "nodes"] as const;

/** Canonical react-query key for the fleet-wide live queue overlay. */
export const QUEUE_QUERY_KEY = ["fleet", "queue"] as const;

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
 * The live per-node scheduler overlay (running/queued counts + entries). Polled
 * on a short interval so the dashboard reflects dispatch without a manual
 * refresh. Returns a `nodeId -> NodeQueue` map for O(1) lookup from each row.
 * A queue error must NOT blank the fleet (presence is the primary signal), so
 * the panel treats this as best-effort: it reads `data` and ignores `error`.
 */
export function useFleetQueueQuery() {
  return useQuery({
    queryKey: QUEUE_QUERY_KEY,
    queryFn: async (): Promise<Map<string, NodeQueue>> => {
      const resp = await queueClient.listQueue({});
      return new Map(resp.nodes.map((n) => [n.nodeId, n]));
    },
    refetchInterval: 5_000,
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

/**
 * Mint a single-use pairing code (owner-gated PairingService.IssuePairingCode).
 * The plaintext code + control-plane public key are returned ONCE; the caller
 * surfaces them for out-of-band delivery to the node's bootstrap installer.
 * Refreshes the node list on success so the freshly paired node appears once
 * it dials in.
 */
export function useIssuePairingCodeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string): Promise<IssuePairingCodeResponse> =>
      pairingClient.issuePairingCode({ name }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: NODES_QUERY_KEY }),
  });
}

export type { Node, NodeQueue };
