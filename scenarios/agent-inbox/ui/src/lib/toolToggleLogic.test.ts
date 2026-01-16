/**
 * Comprehensive tests for tool toggle logic.
 * These tests verify the core toggle behavior in isolation from React.
 */

import { describe, it, expect } from "vitest";
import {
  applyToggleUpdate,
  groupToolsByScenario,
  getToolsNeedingToggle,
  countEnabledTools,
  areAllToolsEnabled,
  areSomeToolsEnabled,
  verifyOnlyOneToolChanged,
} from "./toolToggleLogic";
import type { EffectiveTool, ToolSet } from "./api";

// Helper to get array element with type safety
function getElement<T>(arr: T[], index: number): T {
  const element = arr[index];
  if (element === undefined) throw new Error(`Expected element at index ${index}`);
  return element;
}

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

function createMockToolSet(tools: EffectiveTool[]): ToolSet {
  return {
    scenarios: [],
    tools,
    categories: [],
    generated_at: new Date().toISOString(),
  };
}

describe("applyToggleUpdate", () => {
  it("should toggle only the specified tool", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-b", "tool1", false),
    ];
    const toolSet = createMockToolSet(tools);

    const updated = applyToggleUpdate(toolSet, "scenario-a", "tool1", false);

    expect(getElement(updated.tools, 0).enabled).toBe(false); // Changed
    expect(getElement(updated.tools, 1).enabled).toBe(true); // Unchanged
    expect(getElement(updated.tools, 2).enabled).toBe(false); // Unchanged
  });

  it("should not affect tools with same name in different scenario", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-b", "tool1", true),
    ];
    const toolSet = createMockToolSet(tools);

    const updated = applyToggleUpdate(toolSet, "scenario-a", "tool1", false);

    expect(getElement(updated.tools, 0).enabled).toBe(false); // Changed
    expect(getElement(updated.tools, 1).enabled).toBe(true); // Unchanged - same name, different scenario
  });

  it("should set source to 'global' when no chatId provided", () => {
    const tools = [createMockTool("scenario-a", "tool1", false)];
    const toolSet = createMockToolSet(tools);

    const updated = applyToggleUpdate(toolSet, "scenario-a", "tool1", true);

    expect(getElement(updated.tools, 0).source).toBe("global");
  });

  it("should set source to 'chat' when chatId is provided", () => {
    const tools = [createMockTool("scenario-a", "tool1", false)];
    const toolSet = createMockToolSet(tools);

    const updated = applyToggleUpdate(toolSet, "scenario-a", "tool1", true, "chat-123");

    expect(getElement(updated.tools, 0).source).toBe("chat");
  });

  it("should create new tool objects only for changed tool", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ];
    const toolSet = createMockToolSet(tools);

    const updated = applyToggleUpdate(toolSet, "scenario-a", "tool1", false);

    // Changed tool should be a new object
    expect(getElement(updated.tools, 0)).not.toBe(getElement(toolSet.tools, 0));
    // Unchanged tool should be a new object too (due to map creating new array)
    // But importantly, the VALUES should be unchanged
    expect(getElement(updated.tools, 1).enabled).toBe(getElement(toolSet.tools, 1).enabled);
    expect(getElement(updated.tools, 1).scenario).toBe(getElement(toolSet.tools, 1).scenario);
    expect(getElement(updated.tools, 1).tool.name).toBe(getElement(toolSet.tools, 1).tool.name);
  });

  it("should handle sequential updates correctly", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", true),
    ];
    let toolSet = createMockToolSet(tools);

    // Disable tool1
    toolSet = applyToggleUpdate(toolSet, "scenario-a", "tool1", false);
    expect(getElement(toolSet.tools, 0).enabled).toBe(false);
    expect(getElement(toolSet.tools, 1).enabled).toBe(true);
    expect(getElement(toolSet.tools, 2).enabled).toBe(true);

    // Disable tool2
    toolSet = applyToggleUpdate(toolSet, "scenario-a", "tool2", false);
    expect(getElement(toolSet.tools, 0).enabled).toBe(false);
    expect(getElement(toolSet.tools, 1).enabled).toBe(false);
    expect(getElement(toolSet.tools, 2).enabled).toBe(true);

    // Disable tool3
    toolSet = applyToggleUpdate(toolSet, "scenario-a", "tool3", false);
    expect(getElement(toolSet.tools, 0).enabled).toBe(false);
    expect(getElement(toolSet.tools, 1).enabled).toBe(false);
    expect(getElement(toolSet.tools, 2).enabled).toBe(false);
  });
});

describe("verifyOnlyOneToolChanged", () => {
  it("should return valid when exactly one expected tool changed", () => {
    const before = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ]);
    const after = applyToggleUpdate(before, "scenario-a", "tool1", false);

    const result = verifyOnlyOneToolChanged(before, after, "scenario-a", "tool1");

    expect(result.valid).toBe(true);
    expect(result.changedCount).toBe(1);
    expect(result.unexpectedChanges).toHaveLength(0);
  });

  it("should return invalid when wrong tool changed", () => {
    const before = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ]);
    const after = applyToggleUpdate(before, "scenario-a", "tool2", false);

    const result = verifyOnlyOneToolChanged(before, after, "scenario-a", "tool1");

    expect(result.valid).toBe(false);
    expect(result.changedCount).toBe(1);
    expect(result.unexpectedChanges).toContain("scenario-a:tool2");
  });

  it("should return invalid when multiple tools changed", () => {
    const before = createMockToolSet([
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ]);
    // Manually create after state with multiple changes (simulating a bug)
    const after: ToolSet = {
      ...before,
      tools: before.tools.map((t) => ({ ...t, enabled: false })),
    };

    const result = verifyOnlyOneToolChanged(before, after, "scenario-a", "tool1");

    expect(result.valid).toBe(false);
    expect(result.changedCount).toBe(2);
    expect(result.unexpectedChanges).toContain("scenario-a:tool2");
  });
});

describe("groupToolsByScenario", () => {
  it("should group tools by their scenario", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", false),
      createMockTool("scenario-b", "tool1", true),
    ];

    const grouped = groupToolsByScenario(tools);

    expect(grouped.size).toBe(2);
    expect(grouped.get("scenario-a")).toHaveLength(2);
    expect(grouped.get("scenario-b")).toHaveLength(1);
  });

  it("should create new Map instance each time", () => {
    const tools = [createMockTool("scenario-a", "tool1", true)];

    const grouped1 = groupToolsByScenario(tools);
    const grouped2 = groupToolsByScenario(tools);

    expect(grouped1).not.toBe(grouped2); // Different Map instances
    expect(grouped1.get("scenario-a")).not.toBe(grouped2.get("scenario-a")); // Different array instances
  });
});

describe("getToolsNeedingToggle", () => {
  it("should return tools that need to be enabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", false),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", false),
    ];

    const needingToggle = getToolsNeedingToggle(tools, true);

    expect(needingToggle).toHaveLength(2);
    expect(needingToggle[0].tool.name).toBe("tool1");
    expect(needingToggle[1].tool.name).toBe("tool3");
  });

  it("should return tools that need to be disabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", false),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", true),
    ];

    const needingToggle = getToolsNeedingToggle(tools, false);

    expect(needingToggle).toHaveLength(2);
    expect(needingToggle[0].tool.name).toBe("tool2");
    expect(needingToggle[1].tool.name).toBe("tool3");
  });

  it("should return empty array when no changes needed", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ];

    const needingToggle = getToolsNeedingToggle(tools, true);

    expect(needingToggle).toHaveLength(0);
  });
});

describe("countEnabledTools", () => {
  it("should count enabled tools correctly", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", false),
      createMockTool("scenario-a", "tool3", true),
    ];

    expect(countEnabledTools(tools)).toBe(2);
  });

  it("should return 0 for empty array", () => {
    expect(countEnabledTools([])).toBe(0);
  });
});

describe("areAllToolsEnabled", () => {
  it("should return true when all tools are enabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ];

    expect(areAllToolsEnabled(tools)).toBe(true);
  });

  it("should return false when some tools are disabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", false),
    ];

    expect(areAllToolsEnabled(tools)).toBe(false);
  });

  it("should return false for empty array", () => {
    expect(areAllToolsEnabled([])).toBe(false);
  });
});

describe("areSomeToolsEnabled", () => {
  it("should return true when some (but not all) tools are enabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", false),
    ];

    expect(areSomeToolsEnabled(tools)).toBe(true);
  });

  it("should return false when all tools are enabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
    ];

    expect(areSomeToolsEnabled(tools)).toBe(false);
  });

  it("should return false when no tools are enabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", false),
      createMockTool("scenario-a", "tool2", false),
    ];

    expect(areSomeToolsEnabled(tools)).toBe(false);
  });
});

describe("Sequential toggle simulation (batch scenario toggle)", () => {
  it("should correctly toggle all tools in a scenario one by one", () => {
    const initialTools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", true),
    ];
    let toolSet = createMockToolSet(initialTools);

    // Get tools that need to be disabled
    const toolsToToggle = getToolsNeedingToggle(toolSet.tools, false);
    expect(toolsToToggle).toHaveLength(3);

    // Toggle each one sequentially
    for (const tool of toolsToToggle) {
      const previousToolSet = toolSet;
      toolSet = applyToggleUpdate(toolSet, tool.scenario, tool.tool.name, false);

      // Verify only one tool changed
      const verification = verifyOnlyOneToolChanged(
        previousToolSet,
        toolSet,
        tool.scenario,
        tool.tool.name
      );
      expect(verification.valid).toBe(true);
      expect(verification.changedCount).toBe(1);
    }

    // Final verification: all tools should be disabled
    expect(toolSet.tools[0].enabled).toBe(false);
    expect(toolSet.tools[1].enabled).toBe(false);
    expect(toolSet.tools[2].enabled).toBe(false);
  });

  it("should correctly toggle only tools in the target scenario", () => {
    const initialTools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-b", "tool1", true),
      createMockTool("scenario-b", "tool2", true),
    ];
    let toolSet = createMockToolSet(initialTools);

    // Get scenario-a tools only
    const scenarioATools = toolSet.tools.filter((t) => t.scenario === "scenario-a");
    const toolsToToggle = getToolsNeedingToggle(scenarioATools, false);
    expect(toolsToToggle).toHaveLength(2);

    // Toggle scenario-a tools
    for (const tool of toolsToToggle) {
      toolSet = applyToggleUpdate(toolSet, tool.scenario, tool.tool.name, false);
    }

    // Scenario-a tools should be disabled
    expect(toolSet.tools[0].enabled).toBe(false);
    expect(toolSet.tools[1].enabled).toBe(false);

    // Scenario-b tools should be UNCHANGED
    expect(toolSet.tools[2].enabled).toBe(true);
    expect(toolSet.tools[3].enabled).toBe(true);
  });
});
