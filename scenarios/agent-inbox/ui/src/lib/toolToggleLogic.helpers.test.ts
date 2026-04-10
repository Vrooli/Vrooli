/**
 * Tests for tool toggle logic - Helper functions and batch toggle simulation
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

    expect(grouped1).not.toBe(grouped2);
    expect(grouped1.get("scenario-a")).not.toBe(grouped2.get("scenario-a"));
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
    expect(needingToggle[0]!.tool.name).toBe("tool1");
    expect(needingToggle[1]!.tool.name).toBe("tool3");
  });

  it("should return tools that need to be disabled", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", false),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-a", "tool3", true),
    ];

    const needingToggle = getToolsNeedingToggle(tools, false);

    expect(needingToggle).toHaveLength(2);
    expect(needingToggle[0]!.tool.name).toBe("tool2");
    expect(needingToggle[1]!.tool.name).toBe("tool3");
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

    const toolsToToggle = getToolsNeedingToggle(toolSet.tools, false);
    expect(toolsToToggle).toHaveLength(3);

    for (const tool of toolsToToggle) {
      const previousToolSet = toolSet;
      toolSet = applyToggleUpdate(toolSet, tool.scenario, tool.tool.name, false);

      const verification = verifyOnlyOneToolChanged(
        previousToolSet,
        toolSet,
        tool.scenario,
        tool.tool.name
      );
      expect(verification.valid).toBe(true);
      expect(verification.changedCount).toBe(1);
    }

    expect(toolSet.tools[0]!.enabled).toBe(false);
    expect(toolSet.tools[1]!.enabled).toBe(false);
    expect(toolSet.tools[2]!.enabled).toBe(false);
  });

  it("should correctly toggle only tools in the target scenario", () => {
    const initialTools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-a", "tool2", true),
      createMockTool("scenario-b", "tool1", true),
      createMockTool("scenario-b", "tool2", true),
    ];
    let toolSet = createMockToolSet(initialTools);

    const scenarioATools = toolSet.tools.filter((t) => t.scenario === "scenario-a");
    const toolsToToggle = getToolsNeedingToggle(scenarioATools, false);
    expect(toolsToToggle).toHaveLength(2);

    for (const tool of toolsToToggle) {
      toolSet = applyToggleUpdate(toolSet, tool.scenario, tool.tool.name, false);
    }

    expect(toolSet.tools[0]!.enabled).toBe(false);
    expect(toolSet.tools[1]!.enabled).toBe(false);

    expect(toolSet.tools[2]!.enabled).toBe(true);
    expect(toolSet.tools[3]!.enabled).toBe(true);
  });
});
