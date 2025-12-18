import clsx from "clsx";
import { Image, ListTree, PlayCircle, Terminal } from "lucide-react";
import type { ReactNode } from "react";
import { selectors } from "@constants/selectors";
import type { ViewerTab } from "../ExecutionViewer";

interface ExecutionTabsProps {
  activeTab: ViewerTab;
  onTabChange: (tab: ViewerTab) => void;
  showExecutionSwitcher?: boolean;
  isSwitchingExecution?: boolean;
  hasTimeline: boolean;
  counts: {
    replay: number;
    screenshots: number;
    logs: number;
  };
  tabs: {
    replay: ReactNode;
    screenshots: ReactNode;
    logs: ReactNode;
    executions?: ReactNode;
  };
}

/**
 * ActiveExecutionTabs owns tab navigation and layout for the execution viewer.
 * Content for each tab is supplied by the parent to keep orchestration separate from UI wiring.
 */
export function ActiveExecutionTabs({
  activeTab,
  onTabChange,
  showExecutionSwitcher = false,
  isSwitchingExecution = false,
  hasTimeline,
  counts,
  tabs,
}: ExecutionTabsProps) {
  return (
    <>
      <div className="flex border-b border-gray-800">
        <button
          data-testid={selectors.executions.tabs.replay}
          className={clsx(
            "flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2",
            activeTab === "replay"
              ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
              : hasTimeline
                ? "text-subtle hover:text-surface"
                : "text-gray-500 hover:text-surface/80",
          )}
          onClick={() => onTabChange("replay")}
        >
          <PlayCircle size={14} />
          Replay ({counts.replay})
        </button>
        <button
          data-testid={selectors.executions.tabs.screenshots}
          className={clsx(
            "flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2",
            activeTab === "screenshots"
              ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
              : "text-subtle hover:text-surface",
          )}
          onClick={() => onTabChange("screenshots")}
        >
          <Image size={14} />
          Screenshots ({counts.screenshots})
        </button>
        <button
          data-testid={selectors.executions.tabs.logs}
          className={clsx(
            "flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2",
            activeTab === "logs"
              ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
              : "text-subtle hover:text-surface",
          )}
          onClick={() => onTabChange("logs")}
        >
          <Terminal size={14} />
          Logs ({counts.logs})
        </button>
        {showExecutionSwitcher && (
          <button
            data-testid={selectors.executions.tabs.executions}
            className={clsx(
              "flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2",
              activeTab === "executions"
                ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
                : "text-subtle hover:text-surface",
            )}
            onClick={() => onTabChange("executions")}
          >
            {isSwitchingExecution ? (
              <span className="flex items-center gap-2">
                <Terminal size={14} className="animate-spin" />
                Switching…
              </span>
            ) : (
              <>
                <ListTree size={14} />
                Executions
              </>
            )}
          </button>
        )}
      </div>

      <div className="flex-1 overflow-hidden flex flex-col min-h-0">
        {activeTab === "replay" && tabs.replay}
        {activeTab === "screenshots" && tabs.screenshots}
        {activeTab === "logs" && tabs.logs}
        {activeTab === "executions" && showExecutionSwitcher && tabs.executions}
      </div>
    </>
  );
}

export default ActiveExecutionTabs;
