import { beforeEach, describe, expect, it, vi } from "vitest";
import { createGraphService } from "./graph-service";
import type { IApiClient } from "../lib/api-client";

describe("graphService", () => {
  let mockApiClient: IApiClient;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
  });

  it("fetches the graph projection from the graph endpoint", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      nodes: [],
      edges: [],
      meta: {
        lens: "topology",
        node_count: 0,
        edge_count: 0,
        generated_at: "2026-03-28T00:00:00Z",
      },
    });

    const service = createGraphService(mockApiClient);
    await service.getGraph("topology");

    expect(mockApiClient.get).toHaveBeenCalledWith("/graph?lens=topology", { signal: undefined });
  });

  it("normalizes backend node types into graph entity types", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      nodes: [
        {
          id: "backlog-item/execute/my-feature",
          type: "BacklogItem",
          position: { x: 0, y: 0 },
          data: {
            backlog: {
              kind: "execute",
              name: "my-feature",
              title: "My Feature",
              status: "queued",
              priority: 4,
            },
          },
        },
        {
          id: "initiative/graph-adoption",
          type: "Initiative",
          position: { x: 0, y: 0 },
          data: {
            initiative: {
              name: "graph-adoption",
              title: "Graph Adoption",
              status: "active",
              rollup: {
                total: 3,
                completed: 1,
                in_progress: 1,
                failed: 0,
                pending: 1,
              },
            },
          },
        },
        {
          id: "execution-record/ex-1",
          type: "ExecutionRecord",
          position: { x: 0, y: 0 },
          data: {
            execution: {
              execution_id: "ex-1",
              backlog_kind: "execute",
              backlog_name: "my-feature",
              status: "running",
              mode: "manual",
            },
          },
        },
        {
          id: "agent-activity/act-1",
          type: "AgentActivity",
          position: { x: 0, y: 0 },
          data: {
            activity: {
              activity_id: "act-1",
              owner_type: "backlog",
              owner_kind: "execute",
              owner_name: "my-feature",
              owner_title: "My Feature",
              execution_id: "ex-1",
              purpose: "process",
              interaction_type: "spawn",
              status: "running",
              requested_at: "2026-03-28T00:00:00Z",
              run_id: "run-1",
              task_id: "task-1",
            },
          },
        },
        {
          id: "run/run-1",
          type: "Run",
          position: { x: 0, y: 0 },
          data: {
            run: {
              run_id: "run-1",
              status: "running",
            },
          },
        },
      ],
      edges: [
        {
          id: "member_of:one",
          source: "backlog-item/execute/my-feature",
          target: "initiative/graph-adoption",
          type: "member_of",
        },
      ],
      meta: {
        lens: "topology",
        node_count: 5,
        edge_count: 1,
        generated_at: "2026-03-28T00:00:00Z",
        agent_manager_available: true,
      },
    });

    const service = createGraphService(mockApiClient);
    const graph = await service.getGraph("topology");
    const backlogNode = graph.nodes[0];
    const executionNode = graph.nodes[2];
    const activityNode = graph.nodes[3];
    const runNode = graph.nodes[4];
    const memberEdge = graph.edges[0];

    expect(graph.nodes.map((node) => node.type)).toEqual([
      "backlog",
      "initiative",
      "execution",
      "agent-activity",
      "agent-run",
    ]);
    expect(backlogNode).toBeDefined();
    expect(backlogNode?.data).toMatchObject({
      label: "My Feature",
      entityType: "backlog",
      kind: "execute",
    });
    expect(executionNode).toBeDefined();
    expect(executionNode?.data).toMatchObject({
      label: "execute/my-feature",
      entityType: "execution",
    });
    expect(activityNode).toBeDefined();
    expect(activityNode?.data).toMatchObject({
      label: "My Feature",
      entityType: "agent-activity",
      purpose: "process",
      runId: "run-1",
    });
    expect(runNode).toBeDefined();
    expect(runNode?.data).toMatchObject({
      label: "Run run-1",
      entityType: "agent-run",
    });
    expect(memberEdge).toBeDefined();
    expect(memberEdge).toMatchObject({
      type: "member_of",
      data: { relationship: "member_of" },
    });
    expect(graph.meta).toEqual({
      lens: "topology",
      nodeCount: 5,
      edgeCount: 1,
      generatedAt: "2026-03-28T00:00:00Z",
      agentManagerAvailable: true,
    });
  });

  it("truncates capture labels for readability", async () => {
    vi.mocked(mockApiClient.get).mockResolvedValue({
      nodes: [
        {
          id: "capture/cap-1",
          type: "Capture",
          position: { x: 0, y: 0 },
          data: {
            capture: {
              id: "cap-1",
              text: "This is a deliberately long capture body that should be shortened in the graph label",
              status: "classified",
            },
          },
        },
      ],
      edges: [],
      meta: {
        lens: "topology",
        node_count: 1,
        edge_count: 0,
        generated_at: "2026-03-28T00:00:00Z",
      },
    });

    const service = createGraphService(mockApiClient);
    const graph = await service.getGraph("topology");
    const captureNode = graph.nodes[0];

    expect(captureNode).toBeDefined();
    expect(captureNode?.data.label).toContain("...");
  });
});
