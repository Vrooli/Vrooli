/**
 * Integration tests for ToolConfiguration - Individual toggle isolation and scenario toggles
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
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

describe("ToolConfiguration - Individual Toggle Isolation", () => {
  const mockCategories: ToolCategory[] = [];
  const mockScenarioStatuses: ScenarioStatus[] = [
    {
      scenario: "test-scenario",
      available: true,
      last_checked: new Date().toISOString(),
      tool_count: 3,
    },
  ];

  it("clicking one toggle should only call onToggleTool for that specific tool", () => {
    const onToggleTool = vi.fn();
    const tools = [
      createMockTool("test-scenario", "tool1", true),
      createMockTool("test-scenario", "tool2", true),
      createMockTool("test-scenario", "tool3", true),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    const toggle = screen.getByTestId("tool-toggle-test-scenario-tool2");
    fireEvent.click(toggle);

    expect(onToggleTool).toHaveBeenCalledTimes(1);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool2", false);
  });

  it("clicking toggles for different tools should call onToggleTool separately", () => {
    const onToggleTool = vi.fn();
    const tools = [
      createMockTool("test-scenario", "tool1", true),
      createMockTool("test-scenario", "tool2", false),
      createMockTool("test-scenario", "tool3", true),
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
    expect(onToggleTool).toHaveBeenLastCalledWith("test-scenario", "tool1", false);

    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool2"));
    expect(onToggleTool).toHaveBeenLastCalledWith("test-scenario", "tool2", true);

    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool3"));
    expect(onToggleTool).toHaveBeenLastCalledWith("test-scenario", "tool3", false);

    expect(onToggleTool).toHaveBeenCalledTimes(3);
  });

  it("should handle tools with same name in different scenarios correctly", () => {
    const onToggleTool = vi.fn();
    const tools = [
      createMockTool("scenario-a", "common-tool", true),
      createMockTool("scenario-b", "common-tool", true),
    ];

    const scenarioStatuses: ScenarioStatus[] = [
      { scenario: "scenario-a", available: true, last_checked: new Date().toISOString(), tool_count: 1 },
      { scenario: "scenario-b", available: true, last_checked: new Date().toISOString(), tool_count: 1 },
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={scenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    fireEvent.click(screen.getByTestId("tool-toggle-scenario-a-common-tool"));
    expect(onToggleTool).toHaveBeenCalledWith("scenario-a", "common-tool", false);

    fireEvent.click(screen.getByTestId("tool-toggle-scenario-b-common-tool"));
    expect(onToggleTool).toHaveBeenCalledWith("scenario-b", "common-tool", false);

    expect(onToggleTool).toHaveBeenCalledTimes(2);
  });
});

describe("ToolConfiguration - Scenario Toggle", () => {
  const mockCategories: ToolCategory[] = [];
  const mockScenarioStatuses: ScenarioStatus[] = [
    {
      scenario: "test-scenario",
      available: true,
      last_checked: new Date().toISOString(),
      tool_count: 3,
    },
  ];

  it("scenario toggle should call onToggleTool for each tool that needs changing", async () => {
    const onToggleTool = vi.fn().mockResolvedValue(undefined);
    const tools = [
      createMockTool("test-scenario", "tool1", true),
      createMockTool("test-scenario", "tool2", true),
      createMockTool("test-scenario", "tool3", true),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    const scenarioToggle = screen.getByTestId("scenario-toggle-all-test-scenario");
    act(() => {
      fireEvent.click(scenarioToggle);
    });

    await waitFor(() => {
      expect(onToggleTool).toHaveBeenCalledTimes(3);
    });

    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool1", false);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool2", false);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool3", false);
  });

  it("scenario toggle should only toggle tools that need changing", async () => {
    const onToggleTool = vi.fn().mockResolvedValue(undefined);
    const tools = [
      createMockTool("test-scenario", "tool1", false),
      createMockTool("test-scenario", "tool2", true),
      createMockTool("test-scenario", "tool3", false),
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    const scenarioToggle = screen.getByTestId("scenario-toggle-all-test-scenario");
    act(() => {
      fireEvent.click(scenarioToggle);
    });

    await waitFor(() => {
      expect(onToggleTool).toHaveBeenCalledTimes(2);
    });

    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool1", true);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool3", true);
  });

  it("scenario toggle should not affect other scenarios", async () => {
    const onToggleTool = vi.fn().mockResolvedValue(undefined);
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-b", "tool1", true),
      createMockTool("scenario-b", "tool2", true),
    ];

    const scenarioStatuses: ScenarioStatus[] = [
      { scenario: "scenario-a", available: true, last_checked: new Date().toISOString(), tool_count: 2 },
      { scenario: "scenario-b", available: true, last_checked: new Date().toISOString(), tool_count: 2 },
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={scenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    const scenarioToggle = screen.getByTestId("scenario-toggle-all-scenario-a");
    act(() => {
      fireEvent.click(scenarioToggle);
    });

    await waitFor(() => {
      expect(onToggleTool).toHaveBeenCalledTimes(2);
    });

    const calls = onToggleTool.mock.calls as Array<[string, string, boolean]>;
    const scenarioAcalls = calls.filter((call) => call[0] === "scenario-a");
    const scenarioBcalls = calls.filter((call) => call[0] === "scenario-b");

    expect(scenarioAcalls).toHaveLength(2);
    expect(scenarioBcalls).toHaveLength(0);
  });
});
