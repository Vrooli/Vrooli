/**
 * EntityLink — Shared clickable chip for cross-entity navigation.
 *
 * Navigates through canonical detail routes.
 * Provides consistent per-entity-type styling across all detail pages.
 */

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "../../lib/utils";
import { backlogDetailPath, executionDetailPath, initiativeDetailPath, scenarioDetailPath } from "../../app/routes/route-paths";

/** Entity types that EntityLink supports navigating to. */
export type LinkableEntityType = "backlog" | "initiative" | "scenario" | "execution";

export interface EntityLinkProps {
  entityType: LinkableEntityType;
  /** Display label for the chip. */
  label: string;
  /** Backlog kind (required when entityType is "backlog"). */
  kind?: string;
  /** Entity name (required for backlog, initiative, scenario). */
  name?: string;
  /** Execution ID (required when entityType is "execution"). */
  executionId?: string;
  /** Optional tab to open in the detail panel. */
  tab?: string;
  /** Override the default color scheme. */
  className?: string;
  "data-testid"?: string;
}

/**
 * Color themes per entity type.
 * bg/text pairs designed for dark backgrounds.
 */
const ENTITY_COLORS: Record<LinkableEntityType, string> = {
  backlog: "bg-cyan-500/15 text-cyan-400 hover:bg-cyan-500/25 hover:text-cyan-300",
  initiative: "bg-sky-500/15 text-sky-400 hover:bg-sky-500/25 hover:text-sky-300",
  scenario: "bg-violet-500/15 text-violet-400 hover:bg-violet-500/25 hover:text-violet-300",
  execution: "bg-amber-500/15 text-amber-400 hover:bg-amber-500/25 hover:text-amber-300",
};

export function EntityLink({
  entityType,
  label,
  kind,
  name,
  executionId,
  tab,
  className,
  "data-testid": testId,
}: EntityLinkProps) {
  const navigate = useNavigate();

  const handleClick = useCallback(() => {
    switch (entityType) {
      case "backlog":
        if (kind && name) navigate(backlogDetailPath(kind, name, tab ? { tab } : undefined));
        break;
      case "initiative":
        if (name) navigate(initiativeDetailPath(name, tab ? { tab } : undefined));
        break;
      case "scenario":
        if (name) navigate(scenarioDetailPath(name, tab ? { tab } : undefined));
        break;
      case "execution":
        if (executionId) navigate(executionDetailPath(executionId, tab ? { tab } : undefined));
        break;
    }
  }, [entityType, kind, name, executionId, tab, navigate]);

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium transition-colors",
        ENTITY_COLORS[entityType],
        className,
      )}
      data-testid={testId}
    >
      {label}
    </button>
  );
}
