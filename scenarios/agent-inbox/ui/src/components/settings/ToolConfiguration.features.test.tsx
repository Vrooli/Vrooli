/**
 * Tests for ToolConfiguration - Override indicator, reset, sync, updating, counts, status, category
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

describe("ToolConfiguration - features", () => {
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
