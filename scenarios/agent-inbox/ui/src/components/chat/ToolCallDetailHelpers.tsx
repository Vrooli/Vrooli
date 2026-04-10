/**
 * Helper components and utilities for ToolCallDetailModal.
 *
 * Extracted from ToolCallDetailModal.tsx for modularity.
 */
import {
  CheckCircle2,
  XCircle,
  Loader2,
  ShieldAlert,
  Play,
  BookOpen,
  Package,
  ExternalLink,
} from "lucide-react";
import * as LucideIcons from "lucide-react";
import { Button } from "../ui/button";
import { openScenarioViewerInNewTab } from "../scenarios/ScenarioViewer";
import type { ScenarioInfo } from "../../lib/api";
import type { SkillAttachment } from "../../lib/tool-utils";
import type { ComponentType, SVGProps } from "react";

type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

export function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

/** Get status display information */
export function getStatusDisplay(status: string) {
  switch (status) {
    case "completed":
      return { icon: CheckCircle2, color: "text-green-400", bgColor: "bg-green-500/10", label: "Completed" };
    case "failed": case "error": case "timeout":
      return { icon: XCircle, color: "text-red-400", bgColor: "bg-red-500/10", label: status === "timeout" ? "Timed Out" : "Failed" };
    case "rejected":
      return { icon: XCircle, color: "text-red-400", bgColor: "bg-red-500/10", label: "Rejected" };
    case "cancelled":
      return { icon: XCircle, color: "text-slate-400", bgColor: "bg-slate-500/10", label: "Cancelled" };
    case "running":
      return { icon: Loader2, color: "text-amber-400", bgColor: "bg-amber-500/10", label: "Running", animate: true };
    case "pending_approval":
      return { icon: ShieldAlert, color: "text-yellow-400", bgColor: "bg-yellow-500/10", label: "Pending Approval" };
    default:
      return { icon: Play, color: "text-amber-400", bgColor: "bg-amber-500/10", label: "Pending" };
  }
}

/** Format tool name for display */
export function formatToolName(name: string): string {
  return name.split("_").map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join(" ");
}

/** Format scenario name for display */
export function formatScenarioName(name: string): string {
  return name.split("-").map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join(" ");
}

/** Skill chip component */
export function SkillChip({ skill, onClick }: { skill: SkillAttachment; onClick: () => void }) {
  const iconName = skill.tags?.[0] || "BookOpen";
  const IconComp = getIconComponent(
    iconName.charAt(0).toUpperCase() + iconName.slice(1).replace(/-/g, "")
  );

  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium
        bg-indigo-500/20 text-indigo-300 border border-indigo-500/30
        hover:bg-indigo-500/30 hover:border-indigo-500/50 transition-colors"
    >
      <IconComp className="h-3 w-3" />
      <span>{skill.label}</span>
    </button>
  );
}

/** Section header component */
export function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2 text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
      {children}
    </div>
  );
}

/** Source Scenario section component */
export function SourceScenarioSection({
  scenarioName,
  scenarioInfo,
  isLoading,
}: {
  scenarioName: string;
  scenarioInfo: ScenarioInfo | null;
  isLoading: boolean;
}) {
  const handleOpenScenario = () => {
    openScenarioViewerInNewTab(scenarioName);
  };

  return (
    <div>
      <SectionHeader>Source Scenario</SectionHeader>
      <div className="p-3 rounded-lg bg-slate-800/50 border border-slate-700/50">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-3 min-w-0">
            <div className="p-2 rounded-lg bg-indigo-500/10 shrink-0">
              <Package className="h-4 w-4 text-indigo-400" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-sm font-medium text-white">
                  {formatScenarioName(scenarioName)}
                </span>
                {isLoading ? (
                  <Loader2 className="h-3 w-3 text-slate-400 animate-spin" />
                ) : scenarioInfo?.version ? (
                  <span className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-300">
                    v{scenarioInfo.version}
                  </span>
                ) : null}
              </div>
              {scenarioInfo?.description && (
                <p className="text-xs text-slate-400 mt-1 line-clamp-2">{scenarioInfo.description}</p>
              )}
              {!scenarioInfo && !isLoading && (
                <p className="text-xs text-slate-500 mt-1 italic">Scenario info not available</p>
              )}
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleOpenScenario}
            className="gap-1.5 text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10 shrink-0"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">Open Scenario</span>
          </Button>
        </div>
      </div>
    </div>
  );
}
