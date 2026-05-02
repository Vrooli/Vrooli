/**
 * Operating Mode Details Page
 *
 * Shows the catalog metadata for one operating mode (label, description,
 * scope, run strategy, phases) and the list of initiatives currently bound
 * to that mode. Editable fields (label, description) persist via the API
 * overlay store.
 */

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { Layers, Pencil, X } from "lucide-react";
import { Button } from "../components/ui/button";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { initiativeModeService } from "../services";
import { initiativeDetailPath } from "../app/routes/route-paths";
import type { OperatingModeDetail } from "../types/operating-mode";

const EMPTY_LENSES: never[] = [];

export function OperatingModeDetailsPage() {
  const params = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const mode = params.mode ?? "";

  const { data, isLoading, error } = useQuery<OperatingModeDetail>({
    queryKey: ["operating-modes", "detail", mode],
    queryFn: () => initiativeModeService.getMode(mode),
    enabled: Boolean(mode),
  });

  const [isEditing, setEditing] = useState(false);
  const [labelDraft, setLabelDraft] = useState("");
  const [descriptionDraft, setDescriptionDraft] = useState("");

  useEffect(() => {
    if (data) {
      setLabelDraft(data.entry.label);
      setDescriptionDraft(data.entry.description ?? "");
    }
  }, [data]);

  const updateMutation = useMutation({
    mutationFn: (args: { label?: string; description?: string }) =>
      initiativeModeService.updateMode(mode, args),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["operating-modes"] });
      setEditing(false);
    },
  });

  const phases = useMemo(() => data?.entry.phases ?? [], [data]);

  if (!mode) {
    return <ErrorState title="Missing mode" message="The URL does not include a mode identifier." />;
  }
  if (isLoading) return <PageLoadingState label="Loading operating mode..." />;
  if (error || !data) {
    return (
      <ErrorState
        title="Failed to load operating mode"
        message={(error as Error | undefined)?.message ?? "Unknown error"}
      />
    );
  }

  const { entry, linkedInitiatives } = data;

  const handleSave = () => {
    const trimmedLabel = labelDraft.trim();
    if (!trimmedLabel) return;
    const patch: { label?: string; description?: string } = {};
    if (trimmedLabel !== entry.label) patch.label = trimmedLabel;
    if (descriptionDraft !== (entry.description ?? "")) patch.description = descriptionDraft;
    if (Object.keys(patch).length === 0) {
      setEditing(false);
      return;
    }
    updateMutation.mutate(patch);
  };

  const handleCancel = () => {
    setLabelDraft(entry.label);
    setDescriptionDraft(entry.description ?? "");
    setEditing(false);
  };

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="Operating Mode"
          entityIcon={Layers}
          title={entry.label}
          subtitle={entry.mode}
          nodeId={null}
          lenses={EMPTY_LENSES}
          metadata={
            <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[11px] font-medium text-slate-300">
              {linkedInitiatives.length} initiative{linkedInitiatives.length === 1 ? "" : "s"}
            </span>
          }
        />
      }
    >
      <DetailSection
        title="Overview"
        icon={Layers}
        hideDivider
        action={
          isEditing ? (
            <div className="flex gap-2">
              <Button variant="ghost" size="sm" onClick={handleCancel} disabled={updateMutation.isPending}>
                <X className="mr-1 h-3.5 w-3.5" />Cancel
              </Button>
              <Button size="sm" onClick={handleSave} disabled={updateMutation.isPending || !labelDraft.trim()}>
                {updateMutation.isPending ? "Saving..." : "Save"}
              </Button>
            </div>
          ) : (
            <Button variant="ghost" size="sm" onClick={() => setEditing(true)}>
              <Pencil className="mr-1 h-3.5 w-3.5" />Edit
            </Button>
          )
        }
      >
        {isEditing ? (
          <div className="space-y-3">
            <label className="block text-sm">
              <span className="mb-1 block text-xs font-medium text-slate-400">Display label</span>
              <input
                type="text"
                value={labelDraft}
                onChange={(e) => setLabelDraft(e.target.value)}
                className="w-full rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none"
                data-testid="operating-mode-label-input"
              />
            </label>
            <label className="block text-sm">
              <span className="mb-1 block text-xs font-medium text-slate-400">Description</span>
              <textarea
                value={descriptionDraft}
                onChange={(e) => setDescriptionDraft(e.target.value)}
                rows={4}
                className="w-full rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none"
                data-testid="operating-mode-description-input"
              />
            </label>
            {updateMutation.isError && (
              <p className="text-sm text-red-400">{(updateMutation.error as Error).message}</p>
            )}
          </div>
        ) : (
          <div className="space-y-3 text-sm text-slate-200">
            {entry.description ? (
              <p className="whitespace-pre-wrap">{entry.description}</p>
            ) : (
              <p className="italic text-slate-500">No description set.</p>
            )}
            <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-400 md:grid-cols-4">
              <div>
                <dt className="text-slate-500">Scope</dt>
                <dd className="text-slate-200">{entry.scopeKind}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Run strategy</dt>
                <dd className="text-slate-200">{entry.runStrategy}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Default</dt>
                <dd className="text-slate-200">{entry.default ? "yes" : "no"}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Usage</dt>
                <dd className="text-slate-200">{entry.usageCount} initiative{entry.usageCount === 1 ? "" : "s"}</dd>
              </div>
            </dl>
          </div>
        )}
      </DetailSection>

      {phases.length > 0 && (
        <DetailSection title="Phases">
          <ul className="space-y-1.5 text-sm">
            {phases.map((phase) => (
              <li
                key={phase.phase}
                className="flex items-center justify-between rounded-md border border-slate-800 bg-slate-900/40 px-3 py-2"
              >
                <span className="font-medium text-slate-100">{phase.phase}</span>
                <span className="text-xs text-slate-400">
                  {phase.profileKey} · {phase.writesRepo ? "writes repo" : "read-only"}
                  {phase.requiresCriteria ? " · requires criteria" : ""}
                </span>
              </li>
            ))}
          </ul>
        </DetailSection>
      )}

      <DetailSection title="Linked Initiatives">
        {linkedInitiatives.length === 0 ? (
          <p className="text-sm italic text-slate-500">No initiatives currently use this mode.</p>
        ) : (
          <ul className="space-y-1.5">
            {linkedInitiatives.map((init) => (
              <li key={init.name}>
                <button
                  type="button"
                  onClick={() => navigate(initiativeDetailPath(init.name))}
                  className="flex w-full items-start justify-between gap-2 rounded-md border border-slate-800 bg-slate-900/40 px-3 py-2 text-left text-sm transition-colors hover:border-slate-700 hover:bg-slate-800/60"
                  data-testid="operating-mode-linked-initiative"
                >
                  <div>
                    <p className="font-medium text-slate-100">{init.title || init.name}</p>
                    <p className="text-xs text-slate-500">{init.name}</p>
                  </div>
                  {init.status && (
                    <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[11px] font-medium text-slate-300">
                      {init.status}
                    </span>
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
      </DetailSection>
    </DetailPageLayout>
  );
}
