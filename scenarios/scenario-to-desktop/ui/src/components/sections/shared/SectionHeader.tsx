/**
 * Section header with title, subtitle, optional icon, optional status badge, and collapse toggle.
 * Supports both pipeline (with section numbers and status) and simple variants.
 */

import { useMemo } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import {
  SECTION_IDS,
  SECTION_TO_STAGE,
  type SectionId,
} from "../../../store/sidebarStore";
import { usePipelineStore } from "../../../store";
import { cn } from "../../../lib/utils";
import {
  type StageStatus,
  type StatusDisplayConfig,
  STAGE_STATUS_CONFIG as STATUS_CONFIG,
  HEADER_STATUS_CONFIG as HEADER_STATUS_DISPLAY,
  getStageStatusDisplay as getStatusDisplay,
} from "../../../lib/status-display";

// Re-export types for backwards compatibility
export type { StageStatus, StatusDisplayConfig };
export { STATUS_CONFIG, getStatusDisplay };

interface SectionHeaderProps {
  /** Section title */
  title: string;
  /** Optional subtitle */
  subtitle?: string;
  /** Optional icon component */
  icon?: LucideIcon;
  /** Section identifier (optional - used for section number and status) */
  sectionId?: SectionId;
  /** Variant - pipeline shows section numbers and status badges */
  variant?: "pipeline" | "simple";
  /** Whether the section is collapsible */
  collapsible?: boolean;
  /** Whether the section is currently collapsed */
  collapsed?: boolean;
  /** Callback when collapse toggle is clicked */
  onToggleCollapse?: () => void;
}

export function SectionHeader({
  title,
  subtitle,
  icon: Icon,
  sectionId,
  variant = "simple",
  collapsible,
  collapsed,
  onToggleCollapse,
}: SectionHeaderProps) {
  // Only show section number for pipeline variant
  const showSectionNumber = variant === "pipeline" && sectionId;
  const sectionIndex = sectionId ? SECTION_IDS.indexOf(sectionId) : -1;

  // Get stage status for pipeline variant (non-configuration sections)
  const stage = sectionId ? SECTION_TO_STAGE[sectionId] : undefined;
  const stageSelector = useMemo(
    () => (state: { pipelineStatus: { stages?: Record<string, { status?: string }> } | null }) =>
      stage ? (state.pipelineStatus?.stages?.[stage]?.status ?? "pending") : null,
    [stage]
  );
  const stageStatus = usePipelineStore(stageSelector);

  // Only show status for pipeline variant with non-configuration sections
  const showStatus = variant === "pipeline" && sectionId && sectionId !== "configuration";
  const status = showStatus ? (stageStatus ?? "pending") : null;
  const statusDisplay = status ? HEADER_STATUS_DISPLAY[status as keyof typeof HEADER_STATUS_DISPLAY] : null;

  return (
    <div className="flex items-start justify-between gap-4 p-4">
      <div className="flex items-start gap-3">
        {/* Section number (pipeline variant only) */}
        {showSectionNumber && sectionIndex >= 0 && (
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-800 text-sm font-semibold text-slate-300">
            {sectionIndex}
          </div>
        )}

        {/* Title and subtitle */}
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            {Icon && <Icon className="h-5 w-5 text-slate-400" />}
            <h2 className="text-lg font-semibold text-slate-100">{title}</h2>
          </div>
          {subtitle && (
            <p className="text-sm text-slate-400">{subtitle}</p>
          )}
        </div>
      </div>

      {/* Right side: status badge and collapse toggle */}
      <div className="flex items-center gap-2 shrink-0">
        {/* Status badge (pipeline variant only) */}
        {statusDisplay && (
          <Badge variant="outline" className={cn("flex items-center gap-1.5", statusDisplay.badgeClass)}>
            <statusDisplay.icon className={cn("h-3 w-3", statusDisplay.iconClass)} />
            {statusDisplay.label}
          </Badge>
        )}

        {/* Collapse toggle button */}
        {collapsible && onToggleCollapse && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onToggleCollapse}
            className="h-8 w-8 p-0 text-slate-400 hover:text-slate-200"
            aria-label={collapsed ? "Expand section" : "Collapse section"}
          >
            {collapsed ? (
              <ChevronDown className="h-5 w-5" />
            ) : (
              <ChevronUp className="h-5 w-5" />
            )}
          </Button>
        )}
      </div>
    </div>
  );
}
