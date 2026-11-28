/** Theme-aware marks for the agent chooser. Shape and label identify the agent;
 * semantic theme colors keep the small plates legible in both light and dark. */
export interface AgentAppearance {
  plate: string;
  ink: string;
}

const appearance = (token: string): AgentAppearance => ({
  plate: `rgb(var(${token}) / 0.14)`,
  ink: `rgb(var(${token}))`,
});

const APPEARANCE: Record<string, AgentAppearance> = {
  claude: appearance("--wc-accent"),
  codex: appearance("--wc-accent"),
  opencode: appearance("--wc-success"),
  grok: appearance("--wc-error-text"),
  agy: appearance("--wc-warning"),
};
const NEUTRAL = appearance("--wc-text-muted");
export const MISSING_APPEARANCE = appearance("--wc-warning");
export function agentAppearance(agentID: string | undefined): AgentAppearance {
  return (agentID && APPEARANCE[agentID]) || NEUTRAL;
}
