/**
 * useNodeActionContext — Builds ActionContext and returns ItemActions for a backlog graph node.
 *
 */

import { useMemo } from "react";
import { useBacklogStore } from "../../../stores/backlog-store";
import { useExecutionStore } from "../../../stores/execution-store";
import { getItemActions, type ItemActions } from "../../../lib/backlog-queue-utils";
import type { BacklogGraphNodeData } from "../types";

export function useNodeActionContext(nodeData: BacklogGraphNodeData): ItemActions {
  const allItems = useBacklogStore((s) => s.items);
  const blockingMap = useBacklogStore((s) => s.blockingMap);
  const executions = useExecutionStore((s) => s.items);

  return useMemo(() => {
    const key = `${nodeData.kind}/${nodeData.name}`;

    // Find the full backlog item to get dependsOn.
    const fullItem = allItems.find((i) => i.kind === nodeData.kind && i.name === nodeData.name);
    const item = fullItem ?? { kind: nodeData.kind, name: nodeData.name, status: nodeData.status, dependsOn: [] };

    const hasExecutionHistory = executions.some(
      (e) => e.backlogKind === nodeData.kind && e.backlogName === nodeData.name,
    );

    return getItemActions({
      item,
      blockingInfo: blockingMap[key] ?? null,
      agentRunning: false,
      hasPendingDecisions: false,
      hasExecutionHistory,
    });
  }, [nodeData.kind, nodeData.name, nodeData.status, allItems, blockingMap, executions]);
}
