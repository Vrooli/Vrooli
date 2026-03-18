/**
 * Tests for tool toggle logic - applyToggleUpdate and verifyOnlyOneToolChanged
 */

import { describe, it, expect } from "vitest";
import {
  applyToggleUpdate,
  verifyOnlyOneToolChanged,
} from "./toolToggleLogic";
import type { EffectiveTool, ToolSet } from "./api";

// Helper to get array element with type safety
function getElement<T>(arr: T[], index: number): T {
  const element = arr[index];
  if (element === undefined) throw new Error(`Expected element at index ${index}`);
  return element;
}

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

    expect(getElement(updated.tools, 0).enabled).toBe(false);
    expect(getElement(updated.tools, 1).enabled).toBe(true);
    expect(getElement(updated.tools, 2).enabled).toBe(false);
  });

  it("should not affect tools with same name in different scenario", () => {
    const tools = [
      createMockTool("scenario-a", "tool1", true),
      createMockTool("scenario-b", "tool1", true),
    ];
    const toolSet = createMockToolSet(tools);

    const updated = applyToggleUpdate(toolSet, "scenario-a", "tool1", false);

    expect(getElement(updated.tools, 0).enabled).toBe(false);
    expect(getElement(updated.tools, 1).enabled).toBe(true);
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

    expect(getElement(updated.tools, 0)).not.toBe(getElement(toolSet.tools, 0));
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

    toolSet = applyToggleUpdate(toolSet, "scenario-a", "tool1", false);
    expect(getElement(toolSet.tools, 0).enabled).toBe(false);
    expect(getElement(toolSet.tools, 1).enabled).toBe(true);
    expect(getElement(toolSet.tools, 2).enabled).toBe(true);

    toolSet = applyToggleUpdate(toolSet, "scenario-a", "tool2", false);
    expect(getElement(toolSet.tools, 0).enabled).toBe(false);
    expect(getElement(toolSet.tools, 1).enabled).toBe(false);
    expect(getElement(toolSet.tools, 2).enabled).toBe(true);

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
