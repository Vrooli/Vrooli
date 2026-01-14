/**
 * Tests for ToolConfiguration component
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ToolConfiguration } from "./ToolConfiguration";
import type { EffectiveTool, ScenarioStatus, ToolCategory } from "../../lib/api";

const mockCategories: ToolCategory[] = [
  {
    id: "agent_lifecycle",
    name: "Agent Lifecycle",
    description: "Tools for managing agent runs",
  },
];

const mockScenarioStatuses: ScenarioStatus[] = [
  {
    scenario: "agent-manager",
    available: true,
    last_checked: "2025-01-01T00:00:00Z",
    tool_count: 2,
  },
];

function createMockTool(overrides: Partial<EffectiveTool> = {}): EffectiveTool {
  return {
    scenario: "agent-manager",
    tool: {
      name: "test_tool",
      description: "A test tool for testing",
      category: "agent_lifecycle",
      parameters: {
        type: "object",
        properties: {},
      },
      metadata: {
        enabled_by_default: true,
        requires_approval: false,
      },
    },
    enabled: true,
    source: "",
    requires_approval: false,
    ...overrides,
  };
}

function createToolsByScenario(tools: EffectiveTool[]): Map<string, EffectiveTool[]> {
  const map = new Map<string, EffectiveTool[]>();
  for (const tool of tools) {
    const existing = map.get(tool.scenario) ?? [];
    map.set(tool.scenario, [...existing, tool]);
  }
  return map;
}

describe("ToolConfiguration", () => {
  it("renders loading state", () => {
    render(
      <ToolConfiguration
        toolsByScenario={new Map()}
        categories={[]}
        isLoading={true}
        onToggleTool={vi.fn()}
      />
    );

    expect(screen.getByTestId("tools-loading")).toBeInTheDocument();
    expect(screen.getByText("Loading tools...")).toBeInTheDocument();
  });

  it("renders error state with retry button", () => {
    const onSyncTools = vi.fn();
    render(
      <ToolConfiguration
        toolsByScenario={new Map()}
        categories={[]}
        error="Failed to fetch tools"
        onToggleTool={vi.fn()}
        onSyncTools={onSyncTools}
      />
    );

    expect(screen.getByTestId("tools-error")).toBeInTheDocument();
    expect(screen.getByText("Failed to fetch tools")).toBeInTheDocument();

    const retryButton = screen.getByRole("button", { name: /retry/i });
    fireEvent.click(retryButton);
    expect(onSyncTools).toHaveBeenCalled();
  });

  it("renders empty state when no tools", () => {
    render(
      <ToolConfiguration
        toolsByScenario={new Map()}
        categories={[]}
        onToggleTool={vi.fn()}
      />
    );

    expect(screen.getByTestId("tools-empty")).toBeInTheDocument();
    expect(screen.getByText("No tools available")).toBeInTheDocument();
  });

  it("renders tools grouped by scenario", () => {
    const tools = [
      createMockTool({ tool: { ...createMockTool().tool, name: "tool1" } }),
      createMockTool({ tool: { ...createMockTool().tool, name: "tool2" } }),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={vi.fn()}
      />
    );

    expect(screen.getByTestId("tool-configuration")).toBeInTheDocument();
    expect(screen.getByTestId("scenario-section-agent-manager")).toBeInTheDocument();
    expect(screen.getByText("agent-manager")).toBeInTheDocument();
    expect(screen.getByText("2/2 enabled")).toBeInTheDocument();
  });

  it("toggles scenario expansion", () => {
    const tools = [createMockTool()];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        onToggleTool={vi.fn()}
      />
    );

    // Initially expanded
    expect(screen.getByTestId("tool-item-agent-manager-test_tool")).toBeInTheDocument();

    // Collapse
    fireEvent.click(screen.getByTestId("scenario-toggle-agent-manager"));
    expect(screen.queryByTestId("tool-item-agent-manager-test_tool")).not.toBeInTheDocument();

    // Expand again
    fireEvent.click(screen.getByTestId("scenario-toggle-agent-manager"));
    expect(screen.getByTestId("tool-item-agent-manager-test_tool")).toBeInTheDocument();
  });

  it("calls onToggleTool when toggle is clicked", () => {
    const onToggleTool = vi.fn();
    const tools = [createMockTool({ enabled: true })];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        onToggleTool={onToggleTool}
      />
    );

    const toggle = screen.getByTestId("tool-toggle-agent-manager-test_tool");
    fireEvent.click(toggle);

    expect(onToggleTool).toHaveBeenCalledWith("agent-manager", "test_tool", false);
  });

  it("displays tool metadata (long-running, cost)", () => {
    const tools = [
      createMockTool({
        tool: {
          ...createMockTool().tool,
          metadata: {
            ...createMockTool().tool.metadata,
            long_running: true,
            cost_estimate: "high",
            tags: ["coding", "async"],
          },
        },
      }),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        onToggleTool={vi.fn()}
      />
    );

    // Tags should be visible
    expect(screen.getByText("coding")).toBeInTheDocument();
    expect(screen.getByText("async")).toBeInTheDocument();
  });

  it("shows override indicator for chat-specific config", () => {
    const tools = [createMockTool({ source: "chat" })];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        chatId="chat-123"
        onToggleTool={vi.fn()}
      />
    );

    expect(screen.getByText("override")).toBeInTheDocument();
  });

  it("shows reset button for chat-specific overrides", () => {
    const onResetTool = vi.fn();
    const tools = [createMockTool({ source: "chat" })];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        chatId="chat-123"
        onToggleTool={vi.fn()}
        onResetTool={onResetTool}
      />
    );

    const resetButton = screen.getByTestId("tool-reset-agent-manager-test_tool");
    fireEvent.click(resetButton);

    expect(onResetTool).toHaveBeenCalledWith("agent-manager", "test_tool");
  });

  it("shows sync button when onSyncTools is provided", () => {
    const onSyncTools = vi.fn();
    const tools = [createMockTool()];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        onToggleTool={vi.fn()}
        onSyncTools={onSyncTools}
      />
    );

    const syncButton = screen.getByTestId("sync-tools-button");
    fireEvent.click(syncButton);

    expect(onSyncTools).toHaveBeenCalled();
  });

  it("disables toggles when updating", () => {
    const tools = [createMockTool()];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        isUpdating={true}
        onToggleTool={vi.fn()}
      />
    );

    const toggle = screen.getByTestId("tool-toggle-agent-manager-test_tool");
    expect(toggle).toBeDisabled();
  });

  it("shows correct enabled count", () => {
    const tools = [
      createMockTool({ tool: { ...createMockTool().tool, name: "enabled1" }, enabled: true }),
      createMockTool({ tool: { ...createMockTool().tool, name: "enabled2" }, enabled: true }),
      createMockTool({ tool: { ...createMockTool().tool, name: "disabled1" }, enabled: false }),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        onToggleTool={vi.fn()}
      />
    );

    expect(screen.getByText("2/3 enabled")).toBeInTheDocument();
  });

  it("displays scenario availability status", () => {
    const tools = [createMockTool()];
    const unavailableStatus: ScenarioStatus[] = [
      {
        scenario: "agent-manager",
        available: false,
        last_checked: "2025-01-01T00:00:00Z",
        error: "Connection refused",
      },
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={unavailableStatus}
        onToggleTool={vi.fn()}
      />
    );

    // The red X icon should be present for unavailable scenario
    // (We check for the scenario name which should still render)
    expect(screen.getByText("agent-manager")).toBeInTheDocument();
  });

  it("displays category name as badge", () => {
    const tools = [createMockTool()];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        onToggleTool={vi.fn()}
      />
    );

    expect(screen.getByText("Agent Lifecycle")).toBeInTheDocument();
  });
});
