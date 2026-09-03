import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Dialog } from "@vrooli/react-component-library/Dialog/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { versionLifecycleClient, type CleanupItem } from "../../api/versionLedger";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";

interface Props {
  componentId?: string;
  compact?: boolean;
}

function eligibleItems(items: CleanupItem[]) {
  return items.filter((item) => item.eligible);
}

export function VersionCleanupPanel({ componentId, compact = false }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [olderThanDays, setOlderThanDays] = useState("30");
  const [acknowledged, setAcknowledged] = useState(false);
  const [plan, setPlan] = useState<{ items: CleanupItem[]; planHash: string }>();

  const scope = {
    ...(componentId ? { componentId } : {}),
    olderThanDays: Number(olderThanDays) || 0,
  };
  const planMutation = useMutation({
    mutationFn: async () => {
      const response = await versionLifecycleClient.planCleanup({ scope });
      return { items: response.items, planHash: response.planHash };
    },
    onSuccess: (nextPlan) => {
      setPlan(nextPlan);
      setAcknowledged(false);
    },
  });
  const cleanupMutation = useMutation({
    mutationFn: () =>
      versionLifecycleClient.cleanupVersions({
        scope,
        planHash: plan?.planHash ?? "",
        confirm: true,
      }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["versions"] }),
        queryClient.invalidateQueries({ queryKey: ["version-ledger"] }),
        queryClient.invalidateQueries({ queryKey: ["components"] }),
        queryClient.invalidateQueries({ queryKey: ["catalog-assets"] }),
      ]);
      setPlan(undefined);
      setAcknowledged(false);
    },
  });

  const eligible = plan ? eligibleItems(plan.items) : [];
  const hasValidAge = /^\d+$/.test(olderThanDays);

  const title = componentId ? "Clean up this asset’s versions" : "Clean up unused library versions";

  return (
    <section
      data-testid={componentId ? "asset-version-cleanup" : "library-version-cleanup"}
      className={
        compact
          ? "rounded-control border border-app-border p-space-xs"
          : "rounded-panel border border-app-border bg-app-surface p-space-md"
      }
      aria-labelledby={componentId ? "asset-cleanup-title" : "library-cleanup-title"}
    >
      <div className="flex flex-wrap items-start justify-between gap-space-xs">
        <div>
          <p className="text-[0.7rem] font-semibold uppercase tracking-[0.16em] text-app-muted-foreground">
            Maintenance
          </p>
          <h3
            id={componentId ? "asset-cleanup-title" : "library-cleanup-title"}
            className="mt-space-3xs text-sm font-semibold"
          >
            {title}
          </h3>
          <p className="mt-space-3xs max-w-2xl text-xs text-app-muted-foreground">
            Review versions with no adopters, dependency pins, or surviving source imports. Latest
            and draft versions are always protected.
          </p>
        </div>
        <Button size="sm" variant="secondary" onClick={() => setOpen(true)}>
          Review cleanup
        </Button>
      </div>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title={title}
        closeLabel="Close cleanup dialog"
        description="Review a read-only cleanup plan before retiring any released version folders. Latest, active drafts, adopters, dependency pins, and source imports are protected."
        footer={
          <div className="flex w-full flex-wrap justify-end gap-space-2xs">
            <Button size="sm" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => planMutation.mutate()}
              disabled={!hasValidAge || planMutation.isPending}
            >
              {planMutation.isPending ? "Planning…" : "Preview cleanup"}
            </Button>
            {plan && eligible.length > 0 && (
              <Button
                size="sm"
                variant="danger"
                disabled={!acknowledged || cleanupMutation.isPending}
                onClick={() => cleanupMutation.mutate()}
              >
                {cleanupMutation.isPending
                  ? "Cleaning up…"
                  : `Retire ${eligible.length} version${eligible.length === 1 ? "" : "s"}`}
              </Button>
            )}
          </div>
        }
      >
        <div className="space-y-space-xs">
          <div className="flex flex-wrap items-end gap-space-xs">
            <label className="text-xs text-app-muted-foreground">
              Minimum age in days
              <Input
                aria-label="Minimum age in days"
                inputMode="numeric"
                value={olderThanDays}
                onChange={(event) => setOlderThanDays(event.target.value)}
                className="mt-space-3xs w-control-wide"
              />
            </label>
            <p className="max-w-sm text-xs text-app-muted-foreground">
              Versions younger than this threshold will be shown as protected. The age is calculated
              from the release date, falling back to the creation date.
            </p>
          </div>

          {planMutation.error && (
            <p role="alert" className="text-xs text-app-danger">
              {errorMessage(planMutation.error, t)}
            </p>
          )}

          {plan && (
            <div className="space-y-space-xs rounded-control bg-app-surface-muted p-space-xs">
              <div className="flex flex-wrap items-center justify-between gap-space-xs text-xs">
                <span>
                  {eligible.length} eligible · {plan.items.length - eligible.length} protected
                </span>
                <code className="font-mono text-app-muted-foreground">
                  {plan.planHash.slice(0, 12)}
                </code>
              </div>
              <ul className="max-h-48 space-y-space-3xs overflow-y-auto text-xs">
                {plan.items.map((rawItem) => {
                  const item = rawItem;
                  const references = item.references ?? [];
                  return (
                    <li
                      key={`${item.version?.libraryId}:${item.version?.version}`}
                      className="flex flex-wrap items-center justify-between gap-space-xs"
                    >
                      <span className="font-mono">{item.version?.version}</span>
                      <span className="flex flex-wrap items-center justify-end gap-space-2xs text-right">
                        <StatusBadge tone={item.eligible ? "warning" : "neutral"}>
                          {item.eligible ? "Eligible" : "Protected"}
                        </StatusBadge>
                        <span className="text-app-muted-foreground">
                          {item.ageDays}d old · {item.reason}
                        </span>
                      </span>
                      {references.length > 0 && (
                        <div className="basis-full pl-space-xs text-[0.7rem] text-app-muted-foreground">
                          {references.map((reference) => (
                            <div
                              key={`${reference.ownerLibraryId}:${reference.ownerVersion}:${reference.ownerPath}:${reference.importSpecifier}`}
                            >
                              ← {reference.ownerLibraryId}@{reference.ownerVersion} ·{" "}
                              {reference.ownerPath} imports {reference.importSpecifier}
                            </div>
                          ))}
                        </div>
                      )}
                    </li>
                  );
                })}
              </ul>
              {eligible.length > 0 && (
                <>
                  <label className="flex items-start gap-space-2xs text-xs text-app-muted-foreground">
                    <input
                      type="checkbox"
                      checked={acknowledged}
                      onChange={(event) => setAcknowledged(event.target.checked)}
                      className="mt-0.5"
                    />
                    I reviewed this plan and understand that eligible version source folders will be
                    removed.
                  </label>
                </>
              )}
              {cleanupMutation.error && (
                <p role="alert" className="text-xs text-app-danger">
                  {errorMessage(cleanupMutation.error, t)}
                </p>
              )}
              {cleanupMutation.isSuccess && (
                <p role="status" className="text-xs text-app-success">
                  Cleanup complete. The ledger history was preserved.
                </p>
              )}
            </div>
          )}
        </div>
      </Dialog>
    </section>
  );
}
