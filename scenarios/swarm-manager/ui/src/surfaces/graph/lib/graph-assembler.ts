/**
 * Graph Data Assembler
 *
 * Client-side assembler that maps existing store data (backlog, scenarios,
 * captures, execution, agent-runs) into React Flow nodes and edges.
 *
 * This is throwaway code — it gets replaced when the dedicated graph API lands.
 */

import type { Node, Edge } from "@xyflow/react";
import type { BacklogItem, Capture, ExecutionRecord, Scenario } from "../../../types";
import type { AgentRunRecord } from "../../../stores/agent-runs-store";

/**
 * Assemble graph nodes and edges from all entity stores.
 */
export function assembleGraphData(
  backlogItems: BacklogItem[],
  scenarios: Scenario[],
  executions: ExecutionRecord[],
  captures: Capture[],
  agentRuns: AgentRunRecord[],
): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  // Scenario nodes.
  for (const scenario of scenarios) {
    nodes.push({
      id: `scenario/${scenario.name}`,
      type: "scenario",
      position: { x: 0, y: 0 },
      data: {
        label: scenario.name,
        status: scenario.status,
        entityType: "scenario",
        entity: scenario,
      },
    });
  }

  // Backlog nodes + edges to scenarios.
  for (const item of backlogItems) {
    const nodeId = `${item.kind}/${item.name}`;
    nodes.push({
      id: nodeId,
      type: "backlog",
      position: { x: 0, y: 0 },
      data: {
        label: item.title || item.name,
        status: item.status,
        kind: item.kind,
        entityType: "backlog",
        entity: item,
      },
    });

    // Edge: backlog item depends on other items.
    if (item.dependsOn) {
      for (const dep of item.dependsOn) {
        edges.push({
          id: `dep:${nodeId}->${dep}`,
          source: dep,
          target: nodeId,
          type: "default",
          data: { relationship: "depends-on" },
        });
      }
    }
  }

  // Execution nodes + edges to backlog items.
  for (const exec of executions) {
    const nodeId = `execution/${exec.executionId}`;
    nodes.push({
      id: nodeId,
      type: "execution",
      position: { x: 0, y: 0 },
      data: {
        label: exec.executionId.slice(0, 8),
        status: exec.status,
        entityType: "execution",
        entity: exec,
      },
    });

    if (exec.backlogKind && exec.backlogName) {
      edges.push({
        id: `exec:${nodeId}->${exec.backlogKind}/${exec.backlogName}`,
        source: `${exec.backlogKind}/${exec.backlogName}`,
        target: nodeId,
        type: "default",
        data: { relationship: "executes" },
      });
    }
  }

  // Capture nodes.
  for (const capture of captures) {
    nodes.push({
      id: `capture/${capture.id}`,
      type: "capture",
      position: { x: 0, y: 0 },
      data: {
        label: capture.text.slice(0, 40) + (capture.text.length > 40 ? "..." : ""),
        status: capture.status,
        entityType: "capture",
        entity: capture,
      },
    });
  }

  // Agent-run nodes + edges to backlog items.
  for (const run of agentRuns) {
    const nodeId = `agent-run/${run.runId}`;
    nodes.push({
      id: nodeId,
      type: "agent-run",
      position: { x: 0, y: 0 },
      data: {
        label: run.runId.slice(0, 8),
        status: run.status,
        entityType: "agent-run",
        entity: run,
      },
    });

    if (run.backlogKind && run.backlogName) {
      edges.push({
        id: `run:${nodeId}->${run.backlogKind}/${run.backlogName}`,
        source: `${run.backlogKind}/${run.backlogName}`,
        target: nodeId,
        type: "default",
        data: { relationship: "runs" },
      });
    }
  }

  return { nodes, edges };
}
