/**
 * Integration tests for ToolConfiguration - Props updates and switch element behavior
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ToolConfiguration } from "./ToolConfiguration";
import type { EffectiveTool, ScenarioStatus, ToolCategory } from "../../lib/api";

function createMockTool(
  scenario: string,
  name: string,
  enabled: boolean
): EffectiveTool {
  return {
    scenario,
    tool: {
      name,
      description: `Test tool ${name}`,
      parameters: { type: "object", properties: {} },
      metadata: {
        enabled_by_default: true,
        requires_approval: false,
      },
    },
    enabled,
    source: "",
    requires_approval: false,
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

describe("ToolConfiguration - State After Props Update", () => {
  const mockCategories: ToolCategory[] = [];
  const mockScenarioStatuses: ScenarioStatus[] = [
    {
      scenario: "test-scenario",
      available: true,
      last_checked: new Date().toISOString(),
      tool_count: 2,
    },
  ];

  it("should reflect updated props correctly", () => {
    const onToggleTool = vi.fn();
    const initialTools = [
      createMockTool("test-scenario", "tool1", true),
      createMockTool("test-scenario", "tool2", true),
    ];

    const { rerender } = render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(initialTools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    expect(screen.getByText("2/2 enabled")).toBeInTheDocument();

    const updatedTools = [
      createMockTool("test-scenario", "tool1", false),
      createMockTool("test-scenario", "tool2", true),
    ];

    rerender(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(updatedTools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    expect(screen.getByText("1/2 enabled")).toBeInTheDocument();
  });

  it("should maintain correct toggle behavior after props update", () => {
    const onToggleTool = vi.fn();
    const initialTools = [
      createMockTool("test-scenario", "tool1", true),
      createMockTool("test-scenario", "tool2", true),
    ];

    const { rerender } = render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(initialTools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    const updatedTools = [
      createMockTool("test-scenario", "tool1", false),
      createMockTool("test-scenario", "tool2", true),
    ];

    rerender(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(updatedTools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    onToggleTool.mockClear();

    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool2"));

    expect(onToggleTool).toHaveBeenCalledTimes(1);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool2", false);
  });
});

describe("ToolConfiguration - Switch Element Behavior", () => {
  const mockCategories: ToolCategory[] = [];
  const mockScenarioStatuses: ScenarioStatus[] = [
    {
      scenario: "test-scenario",
      available: true,
      last_checked: new Date().toISOString(),
      tool_count: 2,
    },
  ];

  it("switch should have correct checked state based on enabled prop", () => {
    const tools = [
      createMockTool("test-scenario", "tool1", true),
      createMockTool("test-scenario", "tool2", false),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={vi.fn()}
      />
    );

    const toggle1 = screen.getByTestId("tool-toggle-test-scenario-tool1");
    const toggle2 = screen.getByTestId("tool-toggle-test-scenario-tool2");

    expect(toggle1).toHaveAttribute("aria-checked", "true");
    expect(toggle2).toHaveAttribute("aria-checked", "false");
  });

  it("clicking switch should pass opposite of current checked state", () => {
    const onToggleTool = vi.fn();
    const tools = [
      createMockTool("test-scenario", "tool1", true),
      createMockTool("test-scenario", "tool2", false),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool1"));
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool1", false);

    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool2"));
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool2", true);
  });

  it("each checkbox should have unique data-testid", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-b", "tool1", true),
    ];

    const scenarioStatuses: ScenarioStatus[] = [
      { scenario: "scenario-a", available: true, last_checked: new Date().toISOString(), tool_count: 2 },
      { scenario: "scenario-b", available: true, last_checked: new Date().toISOString(), tool_count: 1 },
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={scenarioStatuses}
        onToggleTool={vi.fn()}
      />
    );

    expect(screen.getByTestId("tool-toggle-scenario-a-tool1")).toBeInTheDocument();
    expect(screen.getByTestId("tool-toggle-scenario-a-tool2")).toBeInTheDocument();
    expect(screen.getByTestId("tool-toggle-scenario-b-tool1")).toBeInTheDocument();
  });
});
