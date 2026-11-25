import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import type { JsonObject, JsonValue } from "@bufbuild/protobuf";
import type {
  ProviderResultGroup,
  QueryPartEvidence,
  QueryResponse,
  SearchHit,
  SearchHitReference,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import {
  QueryPartEvidence_Source,
  QueryPartEvidence_Status,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { runQuery } from "../../api/search";

const scenarioSlug = /^[a-z0-9]+(?:[-_][a-z0-9]+)*$/;
const maturityProvider = "scenario-completeness-scoring.requirements";
const requirementsProvider = "test-genie.requirements";
const visibleHitBudget = 4;
const evidenceOwners = [
  { providerId: maturityProvider, title: "Cached maturity" },
  { providerId: requirementsProvider, title: "Requirement status" },
] as const;

/** A narrow two-owner composition for one scenario, routed through Search Hub. */
export function ScenarioEvidencePanel() {
  const [scenarioInput, setScenarioInput] = useState("");
  const [submittedScenario, setSubmittedScenario] = useState("");
  const query = useQuery({
    queryKey: ["scenario-evidence", submittedScenario],
    enabled: submittedScenario !== "",
    queryFn: () => runQuery({
      query: `cached maturity score and requirement status for ${submittedScenario}`,
      types: ["requirements"],
      all: false,
      limit: 5,
      scope: `scenario:${submittedScenario}`,
    }),
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const scenario = scenarioInput.trim();
    if (scenarioSlug.test(scenario)) setSubmittedScenario(scenario);
    else setSubmittedScenario("");
  }

  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="scenario-evidence-title">
      <h3 id="scenario-evidence-title" className="text-lg font-semibold">Scenario requirements and maturity</h3>
      <p className="mt-1 text-sm text-app-muted-foreground">
        Queries Test Genie requirement records and Scenario Completeness Scoring cached evidence for one named scenario. A cached maturity or requirement status does not prove a current test run passed; only a durable Test Genie execution record can establish that.
      </p>
      <form className="mt-3 flex flex-wrap gap-2" onSubmit={submit}>
        <label className="sr-only" htmlFor="scenario-evidence-slug">Scenario name</label>
        <Input id="scenario-evidence-slug" aria-label="Scenario name" value={scenarioInput} onChange={(event) => setScenarioInput(event.target.value)} placeholder="scenario slug" className="min-w-0 flex-1" />
        <Button type="submit" disabled={!scenarioSlug.test(scenarioInput.trim())}>Load evidence</Button>
      </form>
      {query.isFetching ? <p role="status" className="mt-3 text-sm text-app-muted-foreground">Loading scoped provider evidence…</p> : null}
      {query.isError ? <p role="alert" className="mt-3 text-sm text-app-destructive">Scenario evidence is unavailable; no owner result is inferred.</p> : null}
      {query.data ? <EvidenceResults data={query.data} scenario={submittedScenario} /> : null}
    </section>
  );
}

function EvidenceResults({ data, scenario }: { data: QueryResponse; scenario: string }) {
  const plan = planOwnerEvidence(data);
  const visibleKeys = new Set(plan.flatMap((owner) => owner.hits.map(sourceKey)));
  const omitted = plan.reduce((sum, owner) => sum + owner.omitted, 0);
  return <div className="mt-4" aria-label={`Evidence for ${scenario}`}>
    <ResponseCoverage data={data} visibleKeys={visibleKeys} />
    <p className="mb-2 text-xs text-app-muted-foreground">
      Showing up to {visibleHitBudget} distinct sources across both owners{omitted > 0 ? `; ${omitted} additional source${omitted === 1 ? "" : "s"} omitted.` : "."}
    </p>
    <div className="grid gap-3 lg:grid-cols-2">
      {plan.map((owner) => <ProviderEvidence key={owner.providerId} data={data} scenario={scenario} owner={owner} />)}
    </div>
  </div>;
}

type OwnerEvidence = (typeof evidenceOwners)[number] & {
  group: ProviderResultGroup | undefined;
  allHits: SearchHit[];
  hits: SearchHit[];
  omitted: number;
  duplicateOwners: string[];
  duplicateOmitted: number;
};

function planOwnerEvidence(data: QueryResponse): OwnerEvidence[] {
  const seen = new Map<string, { shown: boolean; ownerTitle: string }>();
  let shownCount = 0;
  return evidenceOwners.map((owner) => {
    const group = data.groups.find((item) => item.providerId === owner.providerId);
    const allHits = uniqueHits([
      ...data.ranked.filter((hit) => hit.providerId === owner.providerId),
      ...(group?.hits ?? []),
    ]);
    const hits: SearchHit[] = [];
    const duplicateOwners = new Set<string>();
    let omitted = 0;
    let duplicateOmitted = 0;
    for (const hit of allHits) {
      const key = sourceKey(hit);
      const prior = seen.get(key);
      if (prior) {
        if (prior.shown) duplicateOwners.add(prior.ownerTitle);
        else duplicateOmitted++;
        continue;
      }
      if (shownCount < visibleHitBudget) {
        seen.set(key, { shown: true, ownerTitle: owner.title });
        hits.push(hit);
        shownCount++;
      } else {
        seen.set(key, { shown: false, ownerTitle: owner.title });
        omitted++;
      }
    }
    return { ...owner, group, allHits, hits, omitted, duplicateOwners: [...duplicateOwners], duplicateOmitted };
  });
}

function ProviderEvidence({ data, scenario, owner }: { data: QueryResponse; scenario: string; owner: OwnerEvidence }) {
  const { providerId, title, group, allHits, hits } = owner;
  const providerDegradations = (data.degradations ?? []).filter((item) => item.providerId === providerId);
  const groupDegradations = group?.degradations ?? [];
  const degraded = Boolean(group?.degraded || groupDegradations.length > 0 || providerDegradations.length > 0);
  return <article className="rounded-md border border-app-border p-3">
    <h4 className="font-semibold">{title}</h4>
    {degraded ? <p role="status" className="mt-1 text-sm text-app-warning">Provider degraded: {group?.note || "reason unavailable"}</p> : null}
    {!group && allHits.length === 0 && providerDegradations.length === 0 ? <p className="mt-1 text-sm text-app-muted-foreground">UNKNOWN: no response from this owner in this query.</p> : null}
    {[...groupDegradations, ...providerDegradations].map((item, index) => <p role="status" key={`${item.leg}-${item.reason}-${index}`} className="mt-1 text-xs text-app-warning">{item.leg}: {item.detail || item.reason}</p>)}
    {allHits.length === 0 && group && !degraded ? <p className="mt-1 text-sm text-app-muted-foreground">No matching evidence returned; this is not evidence of a pass or failure.</p> : null}
    {owner.duplicateOwners.map((name) => <p key={name} className="mt-1 text-xs text-app-muted-foreground">Duplicate source is shown under {name}.</p>)}
    {owner.duplicateOmitted > 0 ? <p className="mt-1 text-xs text-app-muted-foreground">{owner.duplicateOmitted} repeated source{owner.duplicateOmitted === 1 ? " was" : "s were"} also omitted by the shared budget.</p> : null}
    {owner.omitted > 0 ? <p className="mt-1 text-xs text-app-muted-foreground">{owner.omitted} source{owner.omitted === 1 ? " was" : "s were"} omitted by the shared budget.</p> : null}
    {hits.map((hit) => <EvidenceHit key={hit.id} hit={hit} scenario={scenario} providerId={providerId} />)}
  </article>;
}

function ResponseCoverage({ data, visibleKeys }: { data: QueryResponse; visibleKeys: Set<string> }) {
  const degradations = data.degradations ?? [];
  const parts = data.queryParts ?? [];
  if (!data.degraded && !data.partial && degradations.length === 0 && parts.length === 0) return null;
  return <aside role="status" className="mb-3 rounded border border-app-warning/40 bg-app-surface-muted p-2 text-sm">
    {data.degraded || data.partial ? <p className="font-medium text-app-warning">{data.partial ? `Partial response; ${data.pendingProviders} provider(s) pending.` : "Search response degraded."}</p> : null}
    {degradations.map((item, index) => <p key={`${item.component}-${item.providerId}-${index}`} className="mt-1 text-xs text-app-muted-foreground">{[item.component, item.providerId, item.reason, item.detail].filter(Boolean).join(": ")}{item.fallbackActive ? " (fallback active)" : ""}</p>)}
    {parts.map((part, index) => <PartCoverage key={part.partId || index} part={part} data={data} visibleKeys={visibleKeys} />)}
  </aside>;
}

function PartCoverage({ part, data, visibleKeys }: { part: QueryPartEvidence; data: QueryResponse; visibleKeys: Set<string> }) {
  const labels: Record<number, string> = {
    [QueryPartEvidence_Status.CANDIDATES_RETURNED]: "Candidates returned",
    [QueryPartEvidence_Status.SEARCHED_EMPTY]: "Searched; no candidates",
    [QueryPartEvidence_Status.PROVIDER_DEGRADED]: "Provider degraded",
    [QueryPartEvidence_Status.NOT_ROUTED]: "Not routed",
  };
  const refs = part.evidence.map((ref) => resolveReference(data, ref));
  const resolved = refs.filter((hit): hit is SearchHit => hit !== undefined);
  const uniqueResolved = uniqueHits(resolved);
  const shown = uniqueResolved.filter((hit) => visibleKeys.has(sourceKey(hit))).length;
  const omitted = uniqueResolved.length - shown;
  const unresolved = uniqueReferenceCount(part.evidence.filter((_, index) => refs[index] === undefined));
  const details = [labels[part.status] || "Status unknown"];
  if (part.source === QueryPartEvidence_Source.DERIVED_SEGMENT) details.push("derived from a request phrase");
  if (part.degraded && part.status !== QueryPartEvidence_Status.PROVIDER_DEGRADED) details.push("partial provider degradation");
  if (part.partial) details.push("partial coverage");
  if (part.searchedProviderIds.length > 0) details.push(`searched: ${part.searchedProviderIds.join(", ")}`);
  if (part.reason) details.push(partReason(part.reason));
  return <p data-testid="scenario-evidence-query-part" className="mt-1 text-xs text-app-muted-foreground"><span className="font-medium">{part.label || part.partId}:</span> {details.join(" · ")}
    {shown > 0 ? ` · ${shown} supporting hit(s) shown` : ""}
    {omitted > 0 ? ` · ${omitted} supporting hit(s) omitted by shared budget` : ""}
    {unresolved > 0 ? ` · ${unresolved} hit reference(s) could not be resolved` : ""}
    {part.evidence.length === 0 && part.status === QueryPartEvidence_Status.CANDIDATES_RETURNED ? " · no hit reference supplied" : ""}
  </p>;
}

function uniqueReferenceCount(references: SearchHitReference[]) {
  return new Set(references.map(referenceKey)).size;
}

function referenceKey(ref: SearchHitReference) {
  const provider = ref.providerId.trim();
  const type = ref.type.trim();
  if (ref.hitId.trim()) return `hit:${provider}:${type}:${ref.hitId.trim()}:${ref.path.trim()}`;
  if (ref.path.trim()) return `path:${provider}:${type}:${ref.path.trim().split("#", 1)[0]}`;
  if (ref.rankedIndex !== undefined) return `ranked:${ref.rankedIndex}:${provider}:${type}`;
  if (ref.groupIndex !== undefined && ref.hitIndex !== undefined) return `group:${ref.groupIndex}:${ref.hitIndex}:${provider}:${type}`;
  return `fallback:${provider}:${type}`;
}

function partReason(reason: string) {
  switch (reason) {
    case "ambiguous_provider_evidence": return "Possible owners could not be distinguished.";
    case "no_positive_provider_evidence": return "No owner had positive evidence for this request part.";
    case "ambiguous_lexical_boundary": return "The request part could not be separated confidently.";
    default: return reason;
  }
}

function resolveReference(data: QueryResponse, ref: SearchHitReference): SearchHit | undefined {
  let hit: SearchHit | undefined;
  if (ref.rankedIndex !== undefined) hit = data.ranked[ref.rankedIndex];
  else if (ref.groupIndex !== undefined && ref.hitIndex !== undefined) hit = data.groups[ref.groupIndex]?.hits[ref.hitIndex];
  if (!hit) return undefined;
  if (ref.hitId && ref.hitId !== hit.id) return undefined;
  if (ref.providerId && ref.providerId !== hit.providerId) return undefined;
  if (ref.type && ref.type !== hit.type) return undefined;
  if (ref.path && ref.path !== hit.path) return undefined;
  return hit;
}

function EvidenceHit({ hit, scenario, providerId }: { hit: SearchHit; scenario: string; providerId: string }) {
  const fields = hit.metadata ?? {};
  const source = objectField(fields.source);
  const get = (name: string) => jsonText(fields[name]);
  const sourceGet = (name: string) => jsonText(source?.[name]) || jsonText(source?.[camelCase(name)]);
  const evidenceScenario = scenarioFromEvidence(hit, source);
  const scenarioIdentity = evidenceScenario === undefined
    ? "UNKNOWN; owner result does not carry a comparable scenario identity"
    : evidenceScenario === scenario
      ? `MATCH: ${evidenceScenario}`
      : `MISMATCH: ${evidenceScenario}; requested ${scenario} cannot be joined`;
  const isMaturity = providerId === maturityProvider;
  const runFlag = sourceGet("current_run_verified");
  const runEvidence = runFlag === "true"
    ? "Owner flag true; durable run receipt is not included here"
    : runFlag === "false"
      ? "No; this requirement snapshot does not verify a current run"
      : "UNKNOWN; no current-run receipt is included";
  return <div className="mt-2 rounded border border-app-border bg-app-surface-muted p-2">
    <h5 className="font-medium">{hit.title || hit.id}</h5>
    {hit.snippet ? <p className="mt-1 text-sm text-app-muted-foreground">{hit.snippet}</p> : null}
    <dl className="mt-2 grid gap-1 text-xs sm:grid-cols-2">
      <EvidenceField label="Scenario identity" value={scenarioIdentity} />
      <EvidenceField label="Source" value={isMaturity ? get("source") : sourceGet("owner")} />
      <EvidenceField label={isMaturity ? "Calculated at" : "Snapshot generated"} value={isMaturity ? get("calculated_at") : sourceGet("generated_at")} />
      {isMaturity ? <EvidenceField label="Source digest" value={get("tree_digest")} /> : <EvidenceField label="Source path" value={sourceGet("path")} />}
      {isMaturity ? <EvidenceField label="Evidence basis" value={get("basis")} /> : <EvidenceField label="Module" value={get("module") || sourceGet("module")} />}
      {isMaturity ? <EvidenceField label="Cached maturity" value={get("maturity")} /> : <EvidenceField label="Recorded status" value={`${get("status") || "UNKNOWN"} / ${get("live_status") || "UNKNOWN"}`} />}
      {isMaturity ? <EvidenceField label="Approximate" value={get("approximate") || "UNKNOWN"} /> : <EvidenceField label="Current test-run evidence" value={runEvidence} />}
    </dl>
    {isMaturity ? <FreshnessEvidence value={fields.freshness ?? source?.freshness} /> : <p className="mt-1 text-xs text-app-muted-foreground">Snapshot time is provenance only; freshness against the current scenario tree is UNKNOWN unless an owner freshness verdict is supplied.</p>}
    {get("calculated_at") === "" && get("source") === "" ? <p className="mt-1 text-xs text-app-muted-foreground">Owner metadata unavailable; content is shown without inferred provenance.</p> : null}
  </div>;
}

function scenarioFromEvidence(hit: SearchHit, source: JsonObject | undefined): string | undefined {
  const fields = hit.metadata ?? {};
  const candidate = jsonText(fields.scenario) || jsonText(fields.scenario_slug)
    || jsonText(source?.scenario) || jsonText(source?.scenarioSlug);
  return candidate || undefined;
}

function FreshnessEvidence({ value }: { value: JsonValue | undefined }) {
  if (value === undefined || value === null) return <p className="mt-1 text-xs text-app-muted-foreground">Cached score freshness: UNKNOWN (no phase freshness attached to this hit).</p>;
  const freshness = objectField(value);
  if (!freshness) return <p className="mt-1 text-xs">Cached score freshness: {jsonText(value)}</p>;
  const phases = Array.isArray(freshness.phases) ? freshness.phases : [];
  const current = jsonText(freshness.current_digest) || jsonText(freshness.currentDigest);
  return <div className="mt-1 text-xs"><p>Cached score freshness; current digest {current || "UNKNOWN"}</p>
    {phases.length ? phases.map((phase, index) => {
      const row = objectField(phase);
      return <p key={`${jsonText(row?.phase) || "phase"}-${index}`}>{jsonText(row?.phase) || "phase"}: {jsonText(row?.verdict) || "UNKNOWN"} (recorded digest {jsonText(row?.last_digest) || jsonText(row?.lastDigest) || "UNKNOWN"})</p>;
    }) : <p>Phase freshness: UNKNOWN</p>}
  </div>;
}

function EvidenceField({ label, value }: { label: string; value: string }) {
  return <div><dt className="text-app-muted-foreground">{label}</dt><dd className="break-words">{value || "UNKNOWN"}</dd></div>;
}

function objectField(value: JsonValue | undefined): JsonObject | undefined {
  return typeof value === "object" && value !== null && !Array.isArray(value) ? value : undefined;
}

function jsonText(value: JsonValue | undefined): string {
  if (value === undefined || value === null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "boolean" || typeof value === "number") return String(value);
  if (Array.isArray(value)) return JSON.stringify(value);
  if (typeof value === "object") return JSON.stringify(value);
  return "";
}

function camelCase(value: string): string {
  return value.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

function uniqueHits(hits: SearchHit[]): SearchHit[] {
  const seen = new Set<string>();
  return hits.filter((hit) => {
    const key = sourceKey(hit);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function sourceKey(hit: SearchHit) {
  const path = hit.path.trim().split("#", 1)[0];
  if (path) return `path:${path}`;
  if (hit.id) return `id:${hit.providerId}:${hit.type}:${hit.id}`;
  return `fallback:${hit.providerId}:${hit.title}:${hit.snippet}`;
}
