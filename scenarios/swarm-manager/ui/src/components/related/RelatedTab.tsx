import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Network } from "lucide-react";
import type { RelatedEntity, RelatedGroup } from "@vrooli/proto-types/swarm-manager/v1/api/related_pb";
import { backlogDetailPath, goalDetailPath, recordDetailPath } from "../../app/routes/route-paths";
import { dynamicSelectors, selectors } from "../../consts/selectors";
import { useUrlState } from "../../hooks/use-url-state";
import { ErrorState } from "../ui/error-state";
import { PanelLoadingState } from "../ui/loading-states";
import { CollapsibleSection } from "../ui/collapsible-section";
import { relatedService, type RelatedTarget } from "../../services/related-service";

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
          error={relatedQuery.error as Error}
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
    <section className="space-y-4 p-4" data-testid={selectors.related.tab}>
      <div className="flex items-center gap-2">
        <Network className="h-5 w-5 text-cyan-300" aria-hidden />
        <h2 className="text-lg font-semibold">Related work</h2>
      </div>
      <div className="flex flex-wrap gap-2">
        <button aria-pressed={historical === "include"} onClick={() => setHistorical(historical === "include" ? "exclude" : "include")}>
          {historical === "include" ? "Historical included" : "Historical excluded"}
        </button>
        {["backlog", "goal", "record"].map((kind) => (
          <button key={kind} aria-pressed={entityKinds.includes(kind)} onClick={() => toggleKind(kind)}>{kind}s</button>
        ))}
      </div>
      {relatedQuery.data?.groups.map((group: RelatedGroup) => (
        <CollapsibleSection
          key={group.name}
          storageKey={relatedStorageKey(target, group.name)}
          defaultOpen
          className="rounded-lg border border-slate-800/80 bg-slate-900/30"
          headerClassName="px-3 py-2"
          toggleClassName="text-sm capitalize text-slate-200"
          contentClassName="border-t border-slate-800/80 p-3"
          label={
            <>
              {group.name.replace("_", " ")}
              <span className="ml-1 text-slate-500">({group.entities.length})</span>
              {group.degraded && <span className="ml-2 normal-case text-amber-400">Similarity unavailable</span>}
            </>
          }
          contentTestId={groupSelector(group.name)}
        >
          {group.entities.length === 0 ? <p className="text-sm text-slate-500">No related work.</p> : (
            <ul className="space-y-2">
              {group.entities.map((row) => (
                <li key={`${row.entityKind}:${row.key}`} data-testid={dynamicSelectors.related.rowByEntity({ entity: row.entityKind, id: row.key })} className="rounded border border-slate-800 p-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <Link className="text-sky-300" to={pathFor(row)}>{row.title || row.key}</Link>
                    <span className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${ENTITY_TYPE_STYLES[row.entityKind]}`}>{row.entityKind}</span>
                    {row.status && <span className="text-xs text-slate-400">Status: {row.status}</span>}
                    {row.archived && <span className="text-xs text-amber-400">Archived</span>}
                    {row.scorePercent ? <span className="text-xs text-sky-300">{row.scorePercent}% similar</span> : null}
                  </div>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {row.reasons.map((reason) => <span key={reason} className="rounded bg-slate-800 px-1.5 text-xs text-slate-300">{reason}</span>)}
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
