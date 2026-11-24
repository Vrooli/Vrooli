import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import type { JsonObject, JsonValue } from "@bufbuild/protobuf";
import type { QueryResponse, SearchHit } from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { runQuery } from "../../api/search";

const scenarioSlug = /^[a-z0-9]+(?:[-_][a-z0-9]+)*$/;
const maturityProvider = "scenario-completeness-scoring.requirements";
const requirementsProvider = "test-genie.requirements";

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
  return <div className="mt-4 grid gap-3 lg:grid-cols-2" aria-label={`Evidence for ${scenario}`}>
    <ProviderEvidence data={data} scenario={scenario} providerId={maturityProvider} title="Cached maturity" />
    <ProviderEvidence data={data} scenario={scenario} providerId={requirementsProvider} title="Requirement status" />
  </div>;
}

function ProviderEvidence({ data, scenario, providerId, title }: { data: QueryResponse; scenario: string; providerId: string; title: string }) {
  const group = data.groups.find((item) => item.providerId === providerId);
  const hits = uniqueHits([
    ...data.ranked.filter((hit) => hit.providerId === providerId),
    ...(group?.hits ?? []),
  ]);
  const degraded = group?.degraded ?? false;
  return <article className="rounded-md border border-app-border p-3">
    <h4 className="font-semibold">{title}</h4>
    {degraded ? <p role="status" className="mt-1 text-sm text-app-warning">Provider degraded: {group?.note || "reason unavailable"}</p> : null}
    {!group && hits.length === 0 ? <p className="mt-1 text-sm text-app-muted-foreground">UNKNOWN: no response from this owner in this query.</p> : null}
    {group?.degradations.map((item, index) => <p role="status" key={`${item.leg}-${index}`} className="mt-1 text-xs text-app-warning">{item.leg}: {item.detail || item.reason}</p>)}
    {hits.length === 0 && group && !degraded ? <p className="mt-1 text-sm text-app-muted-foreground">No matching evidence returned; this is not evidence of a pass or failure.</p> : null}
    {hits.map((hit) => <EvidenceHit key={hit.id} hit={hit} scenario={scenario} providerId={providerId} />)}
  </article>;
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
    if (seen.has(hit.id)) return false;
    seen.add(hit.id);
    return true;
  });
}
