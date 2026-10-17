/**
 * Per-agent colour, kept apart from the component that draws it.
 *
 * Two reasons this is its own module rather than a constant beside AgentMark:
 * a file that exports both components and values loses fast refresh, and the
 * install/progress card states need the same plate-and-ink pair without
 * rendering a mark.
 *
 * The hues are chosen to stay distinguishable from each other and from the
 * accent cyan the console uses for "selected"; an agent that borrowed the
 * accent would read as chosen when it was not.
 */

export interface AgentAppearance {
  /** Background of the 24px plate. Agent hues sit outside the theme scale. */
  plate: string;
  /** Stroke colour for the mark itself. */
  ink: string;
}

const APPEARANCE: Record<string, AgentAppearance> = {
  claude: { plate: "#241d3a", ink: "#b39bf3" },
  codex: { plate: "#122b38", ink: "#5fd0e8" },
  opencode: { plate: "#152e21", ink: "#6fd79b" },
  grok: { plate: "#331d28", ink: "#eb8fae" },
  agy: { plate: "#2c2617", ink: "#dfc172" },
};

/** The plate for an agent with no catalogue entry — an operator's own command. */
const NEUTRAL: AgentAppearance = { plate: "#232c37", ink: "#8b9cb3" };

/** The warning plate a card wears while its agent is missing. */
export const MISSING_APPEARANCE: AgentAppearance = { plate: "#2f2513", ink: "#f5b23f" };

export function agentAppearance(agentID: string | undefined): AgentAppearance {
  return (agentID && APPEARANCE[agentID]) || NEUTRAL;
}
