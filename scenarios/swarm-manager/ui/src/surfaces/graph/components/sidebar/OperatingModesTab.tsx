/**
 * OperatingModesTab — Lists registered operating modes with usage counts.
 *
 * Sources data from the catalog endpoint (which already includes per-mode
 * usage_count) so the sidebar issues a single network call regardless of
 * how many modes are registered.
 */

import { memo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { HelpCircle } from "lucide-react";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { matchesSearch } from "./useSidebarSearch";
import { initiativeModeService } from "../../../../services";
import type { OperatingModeCatalogEntry } from "../../../../types/operating-mode";
import { OperatingModeCard } from "../../../../components/initiative/operating-mode/operating-mode-card";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { ConceptExplainerDialog } from "../../../../components/ui/concept-explainer-dialog";
import { OPERATING_MODE_INTRO_EXPLAINER } from "../../../../components/initiative/operating-mode/concept-explainers";
import { selectors } from "../../../../consts/selectors";

interface OperatingModesTabProps {
  searchQuery: string;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
}

function LoadingSkeleton() {
  return (
    <div className="space-y-1.5">
      {[1, 2, 3].map((i) => (
        <div key={i} className="animate-pulse rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5">
          <div className="h-4 w-3/4 rounded bg-slate-800" />
          <div className="mt-2 h-3 w-1/2 rounded bg-slate-800" />
        </div>
      ))}
    </div>
  );
}

/**
 * Top-level "what is an operating mode" entry point. It renders above the mode
 * list so an operator who has never seen operating modes can understand the
 * concept — including the resolution ladder — without drilling into a specific
 * mode's detail page. Hidden while searching to keep results uncluttered.
 */
function ModesIntro() {
  const [open, setOpen] = useState(false);
  return (
    <div className="rounded-lg border border-slate-800/80 bg-slate-900/40 px-2.5 py-2">
      <p className="text-[11px] leading-relaxed text-slate-400">
        An <span className="text-slate-200">operating mode</span> is a reusable, inspectable,
        testable methodology loop for driving coding agents — authored as data and run by one
        generic engine.
      </p>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="mt-1.5 inline-flex items-center gap-1.5 rounded text-[11px] font-medium text-cyan-300 transition-colors hover:text-cyan-200"
        data-testid={selectors.initiativeDetails.modesIntroButton}
      >
        <HelpCircle className="h-3.5 w-3.5" aria-hidden="true" />
        What is an operating mode?
      </button>
      <ConceptExplainerDialog
        isOpen={open}
        onClose={() => setOpen(false)}
        title={OPERATING_MODE_INTRO_EXPLAINER.title}
        intro={OPERATING_MODE_INTRO_EXPLAINER.intro}
        sections={OPERATING_MODE_INTRO_EXPLAINER.sections}
        testId={selectors.initiativeDetails.modesIntroDialog}
      />
    </div>
  );
}

function OperatingModesTabImpl({ searchQuery, onItemClick, onClearSearch }: OperatingModesTabProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["operating-modes", "catalog"],
    queryFn: () => initiativeModeService.catalog(),
  });

  // The concept intro is data-independent, so it anchors the tab in every state
  // (loading, error, empty, populated) except an active search.
  const intro = searchQuery ? null : <ModesIntro />;

  const body = (() => {
    if (isLoading) return <LoadingSkeleton />;
    if (error) {
      return (
        <div className="px-2 py-4 text-sm text-red-400">
          Failed to load operating modes: {(error).message}
        </div>
      );
    }

    const modes: OperatingModeCatalogEntry[] = data?.modes ?? [];
    const filtered = searchQuery
      ? modes.filter((m) => matchesSearch(searchQuery, m.label, m.mode, m.description ?? ""))
      : modes;

    if (filtered.length === 0) {
      return (
        <SidebarEmptyState
          icon={SIDEBAR_TAB_ICONS.operatingModes}
          title="No operating modes registered."
          hint="Modes appear here as the system learns new methodologies."
          query={searchQuery}
          onClearSearch={onClearSearch}
        />
      );
    }

    return (
      <div className="space-y-1.5">
        {filtered.map((mode) => (
          <OperatingModeCard
            key={mode.mode}
            mode={mode}
            compact
            onClick={() => onItemClick(`operatingMode/${mode.mode}`)}
            data-testid="sidebar-operating-mode-item"
          />
        ))}
      </div>
    );
  })();

  return (
    <div className="space-y-2">
      {intro}
      {body}
    </div>
  );
}

export const OperatingModesTab = memo(OperatingModesTabImpl);
