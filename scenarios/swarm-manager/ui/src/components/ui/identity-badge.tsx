/**
 * IdentityBadge
 *
 * Inline pill displaying provenance attribution. Distinguishes operator
 * (user icon) from agent (bot icon + profile_key) sources.
 */

import { Bot, User } from "lucide-react";
import { memo } from "react";
import { buildAgentRunUrl } from "../../services/external-links";

interface IdentityBadgeProps {
  value?: string;
  agentManagerUiUrl?: string | null;
}

/**
 * Parse a started_by / created_by string into structured parts.
 * - "agent:<profile_key>/<run_id>" → { type: "agent", profileKey, runId }
 * - anything else → { type: "operator" }
 */
function parseProvenance(value: string): { type: "operator" } | { type: "agent"; profileKey: string; runId: string } {
  if (value.startsWith("agent:")) {
    const rest = value.slice(6);
    const slashIdx = rest.indexOf("/");
    if (slashIdx > 0) {
      return {
        type: "agent",
        profileKey: rest.slice(0, slashIdx),
        runId: rest.slice(slashIdx + 1),
      };
    }
  }
  return { type: "operator" };
}

export const IdentityBadge = memo(function IdentityBadge({ value, agentManagerUiUrl }: IdentityBadgeProps) {
  if (!value) return null;

  const parsed = parseProvenance(value);

  if (parsed.type === "agent") {
    const label = parsed.profileKey.length > 20 ? parsed.profileKey.slice(0, 18) + "\u2026" : parsed.profileKey;
    const runUrl = buildAgentRunUrl(agentManagerUiUrl, parsed.runId);
    const content = (
      <span
        className="inline-flex items-center gap-1 rounded-full bg-violet-500/15 px-2 py-0.5 text-[10px] font-medium text-violet-400"
        title={`Agent: ${parsed.profileKey} (run: ${parsed.runId})`}
      >
        <Bot className="h-3 w-3" />
        {label}
      </span>
    );
    if (runUrl) {
      return (
        <a href={runUrl} target="_blank" rel="noopener noreferrer" className="hover:opacity-80 transition-opacity">
          {content}
        </a>
      );
    }
    return content;
  }

  // Operator or legacy value
  const label = value === "operator" ? "operator" : value;
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full bg-slate-500/15 px-2 py-0.5 text-[10px] font-medium text-slate-400"
      title={`Source: ${value}`}
    >
      <User className="h-3 w-3" />
      {label}
    </span>
  );
});
