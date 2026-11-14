/**
 * RelatedTab — linked, same-scope, and similar work for a backlog item.
 *
 * The tab bar already says "Related", so this owns no page heading and no
 * horizontal gutters (the detail page body supplies those). Rows are dense by
 * design: a related-work list is scanned, not read, so the title leads and
 * every other fact is secondary.
 */

import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import type { RelatedEntity, RelatedGroup } from "@vrooli/proto-types/swarm-manager/v1/api/related_pb";
import { backlogDetailPath, goalDetailPath, recordDetailPath } from "../../app/routes/route-paths";
import { dynamicSelectors, selectors } from "../../consts/selectors";
import { useUrlState } from "../../hooks/use-url-state";
import { formatDisplayText } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import { ErrorState } from "../ui/error-state";
import { PanelLoadingState } from "../ui/loading-states";
import { CollapsibleSection } from "../ui/collapsible-section";
import { relatedService, type RelatedTarget } from "../../services/related-service";

/** Reasons shown inline before the rest collapse into a "+N" chip. */
const VISIBLE_REASONS = 2;

const FILTER_CHIP_BASE = "rounded-full border px-2.5 py-1 text-xs transition-colors";
const FILTER_CHIP_ON = "border-cyan-400/60 bg-cyan-400/10 text-cyan-100";
const FILTER_CHIP_OFF = "border-slate-800 bg-slate-950/55 text-slate-400 hover:border-slate-700 hover:bg-slate-800/55 hover:text-slate-200";

type Props = { target: RelatedTarget; enabled: boolean };

function pathFor(row: RelatedEntity) {
  if (row.entityKind === "backlog") {
    const [kind, name] = row.key.split("/");
    return backlogDetailPath(kind ?? "", name ?? "");
  }
  if (row.entityKind === "goal") return goalDetailPath(row.key);
  return recordDetailPath(row.key);
}

function groupSelector(name: string) {
  if (name === "linked") return selectors.related.groupLinked;
  if (name === "same_scope") return selectors.related.groupSameScope;
  return selectors.related.groupSimilar;
}

const ENTITY_TYPE_STYLES: Record<RelatedEntity["entityKind"], string> = {
  backlog: "bg-cyan-500/15 text-cyan-300",
  goal: "bg-sky-500/15 text-sky-300",
  record: "bg-emerald-500/15 text-emerald-300",
};

function relatedStorageKey(target: RelatedTarget, groupName: string) {
  const targetKey = target.kind === "backlog" ? `${target.backlogKind}-${target.name}` : target.name;
  return `related-${target.kind}-${targetKey}-${groupName}`;
}

export function RelatedTab({ target, enabled }: Props) {
  const [historical, setHistorical] = useUrlState<"include" | "exclude">("historical", "include", {
    validate: (value): value is "include" | "exclude" => value === "include" || value === "exclude",
  });
  const [entityFilter, setEntityFilter] = useUrlState<string>("relatedEntities", "");
  const entityKinds = entityFilter ? entityFilter.split(",").filter(Boolean) : [];
  const relatedQuery = useQuery({
    queryKey: ["related", target, historical, entityFilter],
    enabled,
    retry: false,
    queryFn: () => relatedService.getRelated(target, {
      excludeHistorical: historical === "exclude",
      entityKinds,
    }),
  });

  const toggleKind = (kind: string) => {
    setEntityFilter(entityKinds.includes(kind)
      ? entityKinds.filter((value) => value !== kind).join(",")
      : [...entityKinds, kind].join(","));
  };

  if (relatedQuery.isLoading) {
    return <PanelLoadingState label="Finding related work…" testId={selectors.related.tab} />;
  }

  if (relatedQuery.error) {
    return (
      <div data-testid={selectors.related.tab}>
        <ErrorState
          error={relatedQuery.error}
          variant="notFound"
          title="Related work is not available yet"
          message="This server has not enabled related-work discovery. Refresh after the server is updated."
          onRetry={() => void relatedQuery.refetch()}
          className="m-4"
        />
      </div>
    );
  }

  return (
    <section className="space-y-3 py-1" data-testid={selectors.related.tab}>
      {/* These were unstyled <button>s, so the row read as one run-on
          sentence and the pressed state was invisible. */}
      <div className="flex flex-wrap items-center gap-1.5">
        <button
          type="button"
          aria-pressed={historical === "include"}
          onClick={() => setHistorical(historical === "include" ? "exclude" : "include")}
          className={cn(FILTER_CHIP_BASE, historical === "include" ? FILTER_CHIP_ON : FILTER_CHIP_OFF)}
          title={historical === "include" ? "Archived and completed work is included" : "Archived and completed work is hidden"}
          data-testid="related-filter-historical"
        >
          Historical
        </button>
        <span className="mx-0.5 h-4 w-px bg-slate-800" aria-hidden />
        {["backlog", "goal", "record"].map((kind) => {
          // No selection means no filter, which shows everything — so an empty
          // filter reads as all-on rather than all-off.
          const active = entityKinds.length === 0 || entityKinds.includes(kind);
          return (
            <button
              key={kind}
              type="button"
              aria-pressed={entityKinds.includes(kind)}
              onClick={() => toggleKind(kind)}
              className={cn(FILTER_CHIP_BASE, active ? FILTER_CHIP_ON : FILTER_CHIP_OFF)}
              data-testid={`related-filter-${kind}`}
            >
              {kind}s
            </button>
          );
        })}
      </div>
      {relatedQuery.data?.groups.map((group: RelatedGroup) => (
        <CollapsibleSection
          key={group.name}
          storageKey={relatedStorageKey(target, group.name)}
          defaultOpen
          className="rounded-lg border border-slate-800/80 bg-slate-900/30"
          headerClassName="px-3 py-2"
          toggleClassName="text-sm capitalize text-slate-200"
          contentClassName="border-t border-slate-800/80"
          label={
            <>
              {group.name.replace("_", " ")}
              <span className="ml-1 text-slate-500">({group.entities.length})</span>
              {group.degraded && <span className="ml-2 normal-case text-amber-400">Similarity unavailable</span>}
            </>
          }
          contentTestId={groupSelector(group.name)}
        >
          {group.entities.length === 0 ? <p className="px-3 py-2 text-sm text-slate-500">No related work.</p> : (
            // Dividers instead of a bordered box per row: five nested cards
            // inside an already-bordered group was all frame and no content.
            <ul className="divide-y divide-slate-800/70">
              {group.entities.map((row) => (
                <li
                  key={`${row.entityKind}:${row.key}`}
                  data-testid={dynamicSelectors.related.rowByEntity({ entity: row.entityKind, id: row.key })}
                  className="px-3 py-2"
                >
                  <div className="flex min-w-0 items-start gap-2">
                    <span className={cn("mt-0.5 shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium leading-tight", ENTITY_TYPE_STYLES[row.entityKind])}>
                      {row.entityKind}
                    </span>
                    <Link
                      className="min-w-0 flex-1 text-sm font-medium text-sky-300 line-clamp-2 hover:text-sky-200 hover:underline"
                      to={pathFor(row)}
                      title={row.title || row.key}
                    >
                      {row.title || row.key}
                    </Link>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-500">
                    {row.status && <span>{formatDisplayText(row.status)}</span>}
                    {row.archived && <span className="text-amber-400">Archived</span>}
                    {row.scorePercent ? <span className="text-sky-300">{row.scorePercent}% similar</span> : null}
                    <RelatedReasons reasons={row.reasons} />
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CollapsibleSection>
      ))}
    </section>
  );
}

/**
 * Match reasons, kept on the meta line rather than stacked as full-width
 * blocks. Long scope paths truncate to a readable width and carry the full
 * text in a tooltip; the overflow count is stated, never silently dropped.
 */
function RelatedReasons({ reasons }: { reasons: string[] }) {
  if (reasons.length === 0) return null;
  const visible = reasons.slice(0, VISIBLE_REASONS);
  const hidden = reasons.slice(VISIBLE_REASONS);

  return (
    <>
      {visible.map((reason) => (
        <span
          key={reason}
          className="max-w-[15rem] truncate rounded bg-slate-800/70 px-1.5 py-0.5 text-[11px] text-slate-300"
          title={reason}
        >
          {reason}
        </span>
      ))}
      {hidden.length > 0 && (
        <span
          className="rounded bg-slate-800/70 px-1.5 py-0.5 text-[11px] text-slate-400"
          title={hidden.join("\n")}
          data-testid="related-reasons-overflow"
        >
          +{hidden.length}
        </span>
      )}
    </>
  );
}
