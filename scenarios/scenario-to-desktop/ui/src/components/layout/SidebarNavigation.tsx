/**
 * Section navigation list in sidebar with status indicators.
 */

import { useMemo } from "react";
import {
  Loader2,
  CheckCircle2,
  XCircle,
  Circle,
} from "lucide-react";
import { Button } from "../ui/button";
import {
  useSidebarStore,
  SECTION_IDS,
  SECTION_METADATA,
  SECTION_ICONS,
  SECTION_TO_STAGE,
  type SectionId,
} from "../../store/sidebarStore";
import { usePipelineStore, selectCurrentStage } from "../../store";
import { cn } from "../../lib/utils";

/** Status indicator styles */
const STATUS_STYLES = {
  pending: {
    border: "border-slate-600",
    bg: "bg-slate-950/30",
    icon: Circle,
    iconClass: "text-slate-500",
    numberBg: "bg-slate-800",
  },
  running: {
    border: "border-blue-500",
    bg: "bg-blue-950/30",
    icon: Loader2,
    iconClass: "text-blue-400 animate-spin",
    numberBg: "bg-blue-600",
  },
  completed: {
    border: "border-green-500",
    bg: "bg-green-950/30",
    icon: CheckCircle2,
    iconClass: "text-green-400",
    numberBg: "bg-green-600",
  },
  failed: {
    border: "border-red-500",
    bg: "bg-red-950/30",
    icon: XCircle,
    iconClass: "text-red-400",
    numberBg: "bg-red-600",
  },
  skipped: {
    border: "border-slate-600",
    bg: "bg-slate-950/30",
    icon: Circle,
    iconClass: "text-slate-500",
    numberBg: "bg-slate-700",
  },
} as const;

interface SidebarNavigationProps {
  /** Callback when a section is clicked */
  onSectionClick: (section: SectionId) => void;
  /** Whether sidebar is collapsed (icon-only mode) */
  collapsed?: boolean;
}

export function SidebarNavigation({ onSectionClick, collapsed = false }: SidebarNavigationProps) {
  const activeSection = useSidebarStore((s) => s.activeSection);
  const currentStage = usePipelineStore(selectCurrentStage);

  return (
    <nav className={cn("py-2", collapsed ? "px-1" : "px-3")}>
      <ul className="space-y-1">
        {SECTION_IDS.map((sectionId, index) => (
          <SectionNavItem
            key={sectionId}
            sectionId={sectionId}
            index={index}
            isActive={activeSection === sectionId}
            isCurrentStage={SECTION_TO_STAGE[sectionId] === currentStage}
            onClick={() => onSectionClick(sectionId)}
            collapsed={collapsed}
          />
        ))}
      </ul>
    </nav>
  );
}

interface SectionNavItemProps {
  sectionId: SectionId;
  index: number;
  isActive: boolean;
  isCurrentStage: boolean;
  onClick: () => void;
  collapsed: boolean;
}

function SectionNavItem({ sectionId, index, isActive, isCurrentStage, onClick, collapsed }: SectionNavItemProps) {
  const stage = SECTION_TO_STAGE[sectionId];
  // Memoize the selector to prevent infinite re-renders
  const stageSelector = useMemo(
    () => (state: { pipelineStatus: { stages?: Record<string, { status?: string }> } | null }) =>
      stage ? (state.pipelineStatus?.stages?.[stage]?.status ?? "pending") : null,
    [stage]
  );
  const stageStatus = usePipelineStore(stageSelector);

  // Configuration section doesn't have a stage status
  const status = sectionId === "configuration" ? null : (stageStatus ?? "pending");
  const statusStyle = status ? STATUS_STYLES[status as keyof typeof STATUS_STYLES] : null;

  const Icon = SECTION_ICONS[sectionId];
  const { label, description } = SECTION_METADATA[sectionId];
  const StatusIcon = statusStyle?.icon ?? Circle;

  // Determine the number badge background color based on status
  const numberBgClass = isActive
    ? "bg-blue-500 text-white"
    : statusStyle?.numberBg
      ? `${statusStyle.numberBg} text-white`
      : "bg-slate-800 text-slate-400";

  if (collapsed) {
    return (
      <li>
        <Button
          variant="ghost"
          size="sm"
          onClick={onClick}
          className={cn(
            "w-10 h-10 p-0 relative",
            isActive && "bg-blue-500/20 text-blue-400",
            isCurrentStage && !isActive && "ring-2 ring-blue-500/50",
            statusStyle?.bg
          )}
          title={label}
        >
          <Icon className="h-4 w-4" />
          {status && (
            <span className="absolute -right-0.5 -top-0.5">
              <StatusIcon className={cn("h-3 w-3", statusStyle?.iconClass)} />
            </span>
          )}
        </Button>
      </li>
    );
  }

  return (
    <li>
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "w-full flex items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-all",
          "hover:bg-white/5",
          isActive && "bg-blue-500/10 border-l-4 border-blue-500 pl-2.5",
          !isActive && isCurrentStage && "ring-1 ring-blue-500/50 bg-blue-950/20",
          !isActive && !isCurrentStage && statusStyle?.border && `border-l-2 ${statusStyle.border} pl-3`,
          !isCurrentStage && statusStyle?.bg
        )}
      >
        {/* Index number with status-based coloring */}
        <span
          className={cn(
            "flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-medium transition-colors",
            numberBgClass,
            isCurrentStage && status === "running" && "animate-pulse"
          )}
        >
          {index}
        </span>

        {/* Label and description */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span
              className={cn(
                "font-medium truncate",
                isActive ? "text-blue-400" : "text-slate-200",
                isCurrentStage && !isActive && "text-blue-300"
              )}
            >
              {label}
            </span>
            {status && (
              <StatusIcon className={cn("h-4 w-4 shrink-0", statusStyle?.iconClass)} />
            )}
          </div>
          <p className={cn(
            "text-xs truncate",
            isCurrentStage ? "text-blue-400/70" : "text-slate-500"
          )}>
            {isCurrentStage && status === "running" ? "Running..." : description}
          </p>
        </div>
      </button>
    </li>
  );
}
