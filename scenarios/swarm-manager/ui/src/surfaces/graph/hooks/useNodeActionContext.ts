/**
 * useNodeActionContext — Adapts the server next-action projection for a graph node.
 *
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useBacklogStore } from "../../../stores/backlog-store";
import { backlogService } from "../../../services";
import { defaultQueryOptions } from "../../../lib";
import { itemActionsFromNextAction, type ItemActions } from "../../../lib/backlog-queue-utils";
import type { BacklogGraphNodeData } from "../types";

export function useNodeActionContext(nodeData: BacklogGraphNodeData): ItemActions {
  const allItems = useBacklogStore((s) => s.items);
  const { data: nextAction } = useQuery({
    queryKey: ["backlog", nodeData.kind, nodeData.name, "next-action"],
    queryFn: () => backlogService.getNextAction(nodeData.kind, nodeData.name),
    ...defaultQueryOptions,
  });

  return useMemo(() => {
    // Find the full backlog item for status; the server owns eligibility.
    const fullItem = allItems.find((i) => i.kind === nodeData.kind && i.name === nodeData.name);
    const item = fullItem ?? { kind: nodeData.kind, name: nodeData.name, status: nodeData.status, dependsOn: [] };
    return itemActionsFromNextAction(item, nextAction);
  }, [nodeData.kind, nodeData.name, nodeData.status, allItems, nextAction]);
}
