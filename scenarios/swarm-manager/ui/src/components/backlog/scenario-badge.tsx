/**
 * Scenario Badge
 *
 * Small clickable pill showing which scenario(s) a backlog item targets.
 * Navigates to the scenario detail page on click.
 */

import { memo } from "react";
import { Link } from "react-router-dom";
import { scenariosFromGlobs } from "../../lib/scenario-utils";

interface ScenarioBadgeProps {
  acceptanceAllow?: string[];
}

export const ScenarioBadge = memo(function ScenarioBadge({ acceptanceAllow }: ScenarioBadgeProps) {
  const scenarios = scenariosFromGlobs(acceptanceAllow);
  if (scenarios.length === 0) return null;

  const first = scenarios[0]!;
  const label = first.length > 20 ? first.slice(0, 18) + "\u2026" : first;
  const suffix = scenarios.length > 1 ? ` +${scenarios.length - 1}` : "";
  const tooltip = scenarios.length === 1
    ? `Scenario: ${first}`
    : `Scenarios: ${scenarios.join(", ")}`;

  return (
    <Link
      to={`/scenarios/${first}`}
      onClick={(e) => e.stopPropagation()}
      className="inline-flex items-center rounded-full bg-violet-500/15 px-2 py-0.5 text-[10px] font-medium text-violet-400 hover:bg-violet-500/25 transition-colors"
      title={tooltip}
    >
      {label}{suffix}
    </Link>
  );
});
