// BaselinesTab (Plan B §4.2) — the cross-surface baseline management surface
// and the primary entry point for review. All create/edit/delete/compare lives
// here (Decision 1); other tabs only READ baseline diffs.

import { useState } from "react";
import { Anchor, Plus } from "lucide-react";
import { Button } from "../../components/ui/button";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useBaselines, useDeleteBaseline, useDefaultBaseline } from "../../lib/hooks-baselines";
import { BaselineRow } from "./BaselineRow";
import { BaselineCompareView } from "./BaselineCompareView";
import { SetBaselineModal } from "./SetBaselineModal";
import { BaselineEditModal } from "./BaselineEditModal";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

// CrossTabTarget is the subset of review tabs the compare view can jump to
// ("Open Workflows tab", etc.).
export type CrossTabTarget = "workflows" | "tests" | "screenshots";

interface BaselinesTabProps {
  scenarioSlug: string;
  repoId?: string | null;
  onOpenTab?: (tab: CrossTabTarget) => void;
}

export function BaselinesTab({ scenarioSlug, repoId, onOpenTab }: BaselinesTabProps) {
  const baselinesQuery = useBaselines(scenarioSlug, { repoId });
  const del = useDeleteBaseline(repoId);
  const { defaultBaselineName, setDefaultBaseline } = useDefaultBaseline(scenarioSlug);

  const [comparing, setComparing] = useState<BaselineManifest | null>(null);
  const [editing, setEditing] = useState<BaselineManifest | null>(null);
  const [isSetOpen, setIsSetOpen] = useState(false);
  const [deletingName, setDeletingName] = useState<string | null>(null);

  const baselines = baselinesQuery.data ?? [];

  const handleDelete = (b: BaselineManifest) => {
    setDeletingName(b.name);
    del.mutate(
      { scenario: scenarioSlug, name: b.name, branch: b.branch },
      {
        onSuccess: () => {
          if (defaultBaselineName === b.name) setDefaultBaseline(null);
          if (comparing?.name === b.name) setComparing(null);
        },
        onSettled: () => setDeletingName(null),
      },
    );
  };

  // Compare view replaces the list while active.
  if (comparing) {
    return (
      <>
        <BaselineCompareView
          scenario={scenarioSlug}
          baseline={comparing}
          repoId={repoId}
          onBack={() => setComparing(null)}
          onOpenWorkflows={() => onOpenTab?.("workflows")}
          onOpenTests={() => onOpenTab?.("tests")}
          onOpenScreenshots={() => onOpenTab?.("screenshots")}
        />
        {editing && (
          <BaselineEditModal
            isOpen
            scenario={scenarioSlug}
            baseline={editing}
            repoId={repoId}
            onClose={() => setEditing(null)}
          />
        )}
      </>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-200">Baselines for {scenarioSlug}</h2>
        {baselines.length > 0 && (
          <Button size="sm" onClick={() => setIsSetOpen(true)} className="h-8 px-3 gap-1.5">
            <Plus className="h-3.5 w-3.5" />
            Set baseline
          </Button>
        )}
      </div>

      <MutationErrorBanner error={del.error} onDismiss={() => del.reset()} />

      {baselinesQuery.isLoading ? (
        <div className="space-y-2">
          <div className="h-20 animate-pulse rounded-lg bg-slate-800/60" />
          <div className="h-20 animate-pulse rounded-lg bg-slate-800/60" />
        </div>
      ) : baselinesQuery.error ? (
        <MutationErrorBanner error={baselinesQuery.error} />
      ) : baselines.length === 0 ? (
        <EmptyState onSetBaseline={() => setIsSetOpen(true)} />
      ) : (
        <div className="space-y-2">
          {baselines.map((b) => (
            <BaselineRow
              key={`${b.branch}/${b.name}`}
              baseline={b}
              isDefault={defaultBaselineName === b.name}
              isDeleting={deletingName === b.name}
              onCompare={() => setComparing(b)}
              onEdit={() => setEditing(b)}
              onDelete={() => handleDelete(b)}
              onSetDefault={() => setDefaultBaseline(defaultBaselineName === b.name ? null : b.name)}
            />
          ))}
        </div>
      )}

      <SetBaselineModal
        isOpen={isSetOpen}
        scenario={scenarioSlug}
        repoId={repoId}
        onClose={() => setIsSetOpen(false)}
        onCreated={() => baselinesQuery.refetch()}
      />
      {editing && (
        <BaselineEditModal
          isOpen
          scenario={scenarioSlug}
          baseline={editing}
          repoId={repoId}
          onClose={() => setEditing(null)}
        />
      )}
    </div>
  );
}

function EmptyState({ onSetBaseline }: { onSetBaseline: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-slate-800 py-12 px-6 text-center">
      <Anchor className="h-8 w-8 text-slate-600" />
      <div className="space-y-1">
        <p className="text-sm text-slate-300">No baselines yet</p>
        <p className="text-xs text-slate-500 max-w-sm">
          A baseline captures this scenario's review surfaces (workflows, tests, structure, visuals,
          rules) at a point in time so you can ask "did my change cause this, or was it already
          failing?" without touching the working tree.
        </p>
      </div>
      <Button size="sm" onClick={onSetBaseline} className="h-8 px-3 gap-1.5">
        <Plus className="h-3.5 w-3.5" />
        Set first baseline
      </Button>
    </div>
  );
}
