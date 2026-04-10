/**
 * Pure functions for tool toggle logic.
 * Extracted for testability and to isolate state management from UI.
 */

import type { EffectiveTool, ToolSet, ToolConfigurationScope } from "./api";

/**
 * Apply an optimistic toggle update to a tool set.
 * Returns a new ToolSet with only the specified tool's enabled state changed.
 *
 * @param toolSet - The current tool set
 * @param scenario - The scenario name
 * @param toolName - The tool name
 * @param enabled - The new enabled state
 * @param chatId - Optional chat ID (affects source field)
 * @returns A new ToolSet with the update applied, or null if no change needed
 */
export function applyToggleUpdate(
  toolSet: ToolSet,
  scenario: string,
  toolName: string,
  enabled: boolean,
  chatId?: string
): ToolSet {
  return {
    ...toolSet,
    tools: toolSet.tools.map((t) =>
      t.scenario === scenario && t.tool.name === toolName
        ? { ...t, enabled, source: (chatId ? "chat" : "global") as ToolConfigurationScope }
        : t
    ),
  };
}

/**
 * Find a tool in a tool set by scenario and name.
 */
export function findTool(
  toolSet: ToolSet,
  scenario: string,
  toolName: string
): EffectiveTool | undefined {
  return toolSet.tools.find(
    (t) => t.scenario === scenario && t.tool.name === toolName
  );
}

/**
 * Group tools by scenario into a Map.
 * Returns a new Map each time to ensure React detects changes.
 */
export function groupToolsByScenario(
  tools: EffectiveTool[]
): Map<string, EffectiveTool[]> {
  const map = new Map<string, EffectiveTool[]>();
  for (const tool of tools) {
    const existing = map.get(tool.scenario) ?? [];
    map.set(tool.scenario, [...existing, tool]);
  }
  return map;
}

/**
 * Get tools that need to be toggled to achieve a target enabled state.
 * Used by scenario-level toggle to find which tools need updates.
 */
export function getToolsNeedingToggle(
  tools: EffectiveTool[],
  targetEnabled: boolean
): EffectiveTool[] {
  return tools.filter((t) => t.enabled !== targetEnabled);
}

/**
 * Count enabled tools in a list.
 */
export function countEnabledTools(tools: EffectiveTool[]): number {
  return tools.filter((t) => t.enabled).length;
}

/**
 * Check if all tools in a list are enabled.
 */
export function areAllToolsEnabled(tools: EffectiveTool[]): boolean {
  return tools.length > 0 && tools.every((t) => t.enabled);
}

/**
 * Check if some (but not all) tools in a list are enabled.
 */
export function areSomeToolsEnabled(tools: EffectiveTool[]): boolean {
  const enabledCount = countEnabledTools(tools);
  return enabledCount > 0 && enabledCount < tools.length;
}

/**
 * Verify that exactly one tool changed between two tool sets.
 * Returns the changed tool info or null if validation fails.
 */
export function verifyOnlyOneToolChanged(
  before: ToolSet,
  after: ToolSet,
  expectedScenario: string,
  expectedToolName: string
): { valid: boolean; changedCount: number; unexpectedChanges: string[] } {
  const changedTools: string[] = [];

  for (let i = 0; i < before.tools.length; i++) {
    const beforeTool = before.tools[i];
    const afterTool = after.tools[i];

    if (beforeTool && afterTool && beforeTool.enabled !== afterTool.enabled) {
      changedTools.push(`${afterTool.scenario}:${afterTool.tool.name}`);
    }
  }

  const expectedKey = `${expectedScenario}:${expectedToolName}`;
  const unexpectedChanges = changedTools.filter((key) => key !== expectedKey);

  return {
    valid: changedTools.length === 1 && changedTools[0] === expectedKey,
    changedCount: changedTools.length,
    unexpectedChanges,
  };
}
