/**
 * Integration tests for ToolConfiguration component.
 * These tests verify the component's toggle behavior with multiple tools.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { ToolConfiguration } from "./ToolConfiguration";
import type { EffectiveTool, ScenarioStatus, ToolCategory } from "../../lib/api";

// Helper to create mock tools
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

  it("clicking one toggle should only call onToggleTool for that specific tool", async () => {
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

    // Click the toggle for tool2
    const toggle = screen.getByTestId("tool-toggle-test-scenario-tool2");
    fireEvent.click(toggle);

    // Verify onToggleTool was called exactly once with the correct arguments
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

    // Click toggle for tool1
    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool1"));
    expect(onToggleTool).toHaveBeenLastCalledWith("test-scenario", "tool1", false);

    // Click toggle for tool2
    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool2"));
    expect(onToggleTool).toHaveBeenLastCalledWith("test-scenario", "tool2", true);

    // Click toggle for tool3
    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool3"));
    expect(onToggleTool).toHaveBeenLastCalledWith("test-scenario", "tool3", false);

    // Verify total call count
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

    // Click toggle for scenario-a's tool
    fireEvent.click(screen.getByTestId("tool-toggle-scenario-a-common-tool"));
    expect(onToggleTool).toHaveBeenCalledWith("scenario-a", "common-tool", false);

    // Click toggle for scenario-b's tool
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

    // Click the scenario toggle (all tools enabled, so this should disable all)
    const scenarioToggle = screen.getByTestId("scenario-toggle-all-test-scenario");
    await act(async () => {
      fireEvent.click(scenarioToggle);
    });

    // Wait for all calls to complete
    await waitFor(() => {
      expect(onToggleTool).toHaveBeenCalledTimes(3);
    });

    // Verify each tool was toggled to disabled
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool1", false);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool2", false);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool3", false);
  });

  it("scenario toggle should only toggle tools that need changing", async () => {
    const onToggleTool = vi.fn().mockResolvedValue(undefined);
    const tools = [
      createMockTool("test-scenario", "tool1", false), // Already disabled
      createMockTool("test-scenario", "tool2", true),  // Needs to be disabled
      createMockTool("test-scenario", "tool3", false), // Already disabled
    ];

    render(
      <ToolConfiguration
        toolsByScenario={createToolsByScenario(tools)}
        categories={mockCategories}
        scenarioStatuses={mockScenarioStatuses}
        onToggleTool={onToggleTool}
      />
    );

    // Click scenario toggle (not all enabled, so we'll enable all)
    const scenarioToggle = screen.getByTestId("scenario-toggle-all-test-scenario");
    await act(async () => {
      fireEvent.click(scenarioToggle);
    });

    // Wait for calls to complete
    await waitFor(() => {
      expect(onToggleTool).toHaveBeenCalledTimes(2);
    });

    // Should only toggle the disabled ones to enabled
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

    // Click scenario-a toggle
    const scenarioToggle = screen.getByTestId("scenario-toggle-all-scenario-a");
    await act(async () => {
      fireEvent.click(scenarioToggle);
    });

    // Wait for calls
    await waitFor(() => {
      expect(onToggleTool).toHaveBeenCalledTimes(2);
    });

    // Should only toggle scenario-a tools
    const calls = onToggleTool.mock.calls as Array<[string, string, boolean]>;
    const scenarioAcalls = calls.filter((call) => call[0] === "scenario-a");
    const scenarioBcalls = calls.filter((call) => call[0] === "scenario-b");

    expect(scenarioAcalls).toHaveLength(2);
    expect(scenarioBcalls).toHaveLength(0);
  });
});

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

    // Initially shows 2/2 enabled
    expect(screen.getByText("2/2 enabled")).toBeInTheDocument();

    // Update props with one tool disabled
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

    // Should now show 1/2 enabled
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

    // Simulate props update (as if optimistic update happened)
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

    // Clear the mock to track only new calls
    onToggleTool.mockClear();

    // Click tool2 toggle
    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool2"));

    // Should call with correct arguments for tool2
    expect(onToggleTool).toHaveBeenCalledTimes(1);
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool2", false);
  });
});

describe("ToolConfiguration - Checkbox Element Behavior", () => {
  const mockCategories: ToolCategory[] = [];
  const mockScenarioStatuses: ScenarioStatus[] = [
    {
      scenario: "test-scenario",
      available: true,
      last_checked: new Date().toISOString(),
      tool_count: 2,
    },
  ];

  it("checkbox should have correct checked state based on enabled prop", () => {
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

    const toggle1 = screen.getByTestId("tool-toggle-test-scenario-tool1") as HTMLInputElement;
    const toggle2 = screen.getByTestId("tool-toggle-test-scenario-tool2") as HTMLInputElement;

    expect(toggle1.checked).toBe(true);
    expect(toggle2.checked).toBe(false);
  });

  it("clicking checkbox should pass opposite of current checked state", () => {
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

    // Click enabled tool (should disable)
    fireEvent.click(screen.getByTestId("tool-toggle-test-scenario-tool1"));
    expect(onToggleTool).toHaveBeenCalledWith("test-scenario", "tool1", false);

    // Click disabled tool (should enable)
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

    // All should be found uniquely
    expect(screen.getByTestId("tool-toggle-scenario-a-tool1")).toBeInTheDocument();
    expect(screen.getByTestId("tool-toggle-scenario-a-tool2")).toBeInTheDocument();
    expect(screen.getByTestId("tool-toggle-scenario-b-tool1")).toBeInTheDocument();
  });
});
