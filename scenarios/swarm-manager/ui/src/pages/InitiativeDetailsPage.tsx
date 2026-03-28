/**
 * Initiative Details Page
 *
 * Displays detailed information about a single initiative including:
 * - Metadata (title, description, status)
 * - Progress rollup (completed/in_progress/failed/pending)
 * - Member backlog items as clickable chips
 * - Created/updated timestamps
 */

import { useMemo, useState, useRef, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams, Link, useSearchParams } from "react-router-dom";
import { ChevronLeft, Target } from "lucide-react";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { initiativeService } from "../services";
import { selectors } from "../consts/selectors";
import { INITIATIVE_STATUS_CHIP_COLORS, BACKLOG_STATUS_CHIP_COLORS } from "../types";
import type { BacklogStatus } from "../types";
import { useBacklogStore } from "../stores";

/** Parse "kind/name" item ref into parts. */
function parseItemRef(ref: string): { kind: string; name: string } | null {
  const slashIdx = ref.indexOf("/");
  if (slashIdx < 1) return null;
  return { kind: ref.slice(0, slashIdx), name: ref.slice(slashIdx + 1) };
}

export function InitiativeDetailsPage() {
  const { name } = useParams<{ name: string }>();
  const [searchParams] = useSearchParams();
  const returnTo = searchParams.get("returnTo");
  const backLink = returnTo ?? "/graph";

  const backlogItems = useBacklogStore((s) => s.items);

  const {
    data,
    error,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["initiative", name],
    queryFn: () => initiativeService.get(name!),
    enabled: !!name,
    ...defaultQueryOptions,
  });

  const initiative = data?.initiative;
  const rollup = data?.rollup;

  // Resolve member items against the backlog store
  const resolvedItems = useMemo(() => {
    if (!initiative?.items) return [];
    return initiative.items.map((ref) => {
      const parsed = parseItemRef(ref);
      if (!parsed) return { ref, kind: "", name: ref, title: ref, status: "backlog" as BacklogStatus };
      const found = backlogItems.find((bi) => bi.kind === parsed.kind && bi.name === parsed.name);
      return {
        ref,
        kind: parsed.kind,
        name: parsed.name,
        title: found?.title ?? `${parsed.kind}/${parsed.name}`,
        status: (found?.status ?? "archived") as BacklogStatus,
      };
    });
  }, [initiative?.items, backlogItems]);

  // Collapsible description
  const [descExpanded, setDescExpanded] = useState(false);
  const descRef = useRef<HTMLParagraphElement>(null);
  const [descOverflows, setDescOverflows] = useState(false);

  useEffect(() => {
    if (descRef.current) {
      setDescOverflows(descRef.current.scrollHeight > descRef.current.clientHeight);
    }
  }, [initiative?.description]);

  // Rollup total for progress bar
  const rollupTotal = rollup ? rollup.completed + rollup.inProgress + rollup.failed + rollup.pending : 0;

  if (isLoading) {
    return <PageLoadingState label="Loading initiative..." />;
  }

  if (error || !initiative) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-8">
        <Link to={backLink} className="mb-4 inline-flex items-center gap-1 text-sm text-slate-400 hover:text-slate-200">
          <ChevronLeft className="h-4 w-4" /> Back
        </Link>
        <ErrorState
          error={error as Error | undefined}
          title="Failed to load initiative"
          message={`Could not load initiative "${name}".`}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  const statusColors = INITIATIVE_STATUS_CHIP_COLORS[initiative.status] ?? INITIATIVE_STATUS_CHIP_COLORS.active;

  return (
    <div className="mx-auto max-w-3xl space-y-6 px-4 py-6" data-testid={selectors.initiativeDetails.page}>
      {/* Breadcrumb */}
      <nav className="flex items-center gap-2 text-sm">
        <Link
          to={backLink}
          className="flex items-center gap-1 text-slate-400 hover:text-slate-200 transition-colors"
          data-testid={selectors.initiativeDetails.backLink}
        >
          <ChevronLeft className="h-4 w-4" />
          Back
        </Link>
        <span className="text-slate-600">/</span>
        <span className="text-slate-400">Initiatives</span>
        <span className="text-slate-600">/</span>
        <span className="truncate text-slate-200">{initiative.title || initiative.name}</span>
      </nav>

      {/* Metadata Card */}
      <Card className="rounded-lg border-slate-700/60 bg-slate-900/45 p-5">
        <div className="space-y-4">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-2.5">
              <Target className="h-5 w-5 text-sky-400 shrink-0" />
              <h1
                className="text-xl font-semibold text-slate-100"
                data-testid={selectors.initiativeDetails.title}
              >
                {initiative.title || initiative.name}
              </h1>
            </div>
            <span
              className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${statusColors}`}
              data-testid={selectors.initiativeDetails.status}
            >
              {initiative.status}
            </span>
          </div>

          {initiative.description && (
            <div data-testid={selectors.initiativeDetails.description}>
              <p
                ref={descRef}
                className={`text-sm text-slate-300 leading-relaxed ${descExpanded ? "" : "line-clamp-4"}`}
              >
                {initiative.description}
              </p>
              {(descOverflows || descExpanded) && (
                <button
                  type="button"
                  onClick={() => setDescExpanded(!descExpanded)}
                  className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
                >
                  {descExpanded ? "Show less" : "Show more\u2026"}
                </button>
              )}
            </div>
          )}

          <div className="flex gap-6 text-xs text-slate-500">
            <div>
              <span className="uppercase tracking-wider">Created</span>{" "}
              <span className="text-slate-400">{formatRelativeTime(initiative.created)}</span>
            </div>
            <div>
              <span className="uppercase tracking-wider">Updated</span>{" "}
              <span className="text-slate-400">{formatRelativeTime(initiative.updated)}</span>
            </div>
          </div>
        </div>
      </Card>

      {/* Progress Rollup Card */}
      {rollup && rollupTotal > 0 && (
        <Card className="rounded-lg border-slate-700/60 bg-slate-900/45 p-5" data-testid={selectors.initiativeDetails.rollup}>
          <div className="space-y-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-400">
              Progress
            </h2>

            {/* Segmented progress bar */}
            <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-slate-800">
              {rollup.completed > 0 && (
                <div
                  className="bg-emerald-500 transition-all"
                  style={{ width: `${(rollup.completed / rollupTotal) * 100}%` }}
                  title={`${rollup.completed} completed`}
                />
              )}
              {rollup.inProgress > 0 && (
                <div
                  className="bg-purple-500 transition-all"
                  style={{ width: `${(rollup.inProgress / rollupTotal) * 100}%` }}
                  title={`${rollup.inProgress} in progress`}
                />
              )}
              {rollup.failed > 0 && (
                <div
                  className="bg-red-500 transition-all"
                  style={{ width: `${(rollup.failed / rollupTotal) * 100}%` }}
                  title={`${rollup.failed} failed`}
                />
              )}
              {rollup.pending > 0 && (
                <div
                  className="bg-slate-600 transition-all"
                  style={{ width: `${(rollup.pending / rollupTotal) * 100}%` }}
                  title={`${rollup.pending} pending`}
                />
              )}
            </div>

            {/* Numeric breakdown */}
            <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs">
              <span className="text-emerald-400">{rollup.completed} completed</span>
              <span className="text-purple-400">{rollup.inProgress} in progress</span>
              {rollup.failed > 0 && <span className="text-red-400">{rollup.failed} failed</span>}
              <span className="text-slate-400">{rollup.pending} pending</span>
              <span className="text-slate-500">{rollupTotal} total</span>
            </div>
          </div>
        </Card>
      )}

      {/* Member Items Card */}
      {resolvedItems.length > 0 && (
        <Card className="rounded-lg border-slate-700/60 bg-slate-900/45 p-5" data-testid={selectors.initiativeDetails.itemsList}>
          <div className="space-y-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-400">
              Items ({resolvedItems.length})
            </h2>
            <div className="flex flex-wrap gap-1.5">
              {resolvedItems.map((item) => {
                const chipColors = BACKLOG_STATUS_CHIP_COLORS[item.status] ?? "bg-slate-600/20 text-slate-300";
                return (
                  <Link
                    key={item.ref}
                    to={`/details/backlog/${item.kind}/${item.name}?returnTo=${encodeURIComponent(`/details/initiative/${name}`)}`}
                    className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium transition-colors hover:brightness-125 ${chipColors}`}
                  >
                    {item.title}
                  </Link>
                );
              })}
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
