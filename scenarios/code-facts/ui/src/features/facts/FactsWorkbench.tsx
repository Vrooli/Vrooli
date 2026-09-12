import { useMutation, useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  ArrowRight,
  Braces,
  CheckCircle2,
  CircleDot,
  Database,
  FileSearch,
  Gauge,
  GitBranch,
  Layers3,
  Loader2,
  Play,
  RefreshCcw,
  Search,
  ShieldCheck,
  Sparkles,
  X,
} from "lucide-react";
import { FormEvent, useMemo, useState } from "react";

import {
  FactFamily,
  TargetKind,
  describeCodeFacts,
  getIndexStatus,
  moduleTarget,
  pathTarget,
  projectTarget,
  scenarioTarget,
  searchCodeFacts,
} from "../../api/facts";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import type {
  CodeFactsReport,
  CodeTarget,
  Evidence,
  GenericFact,
  ParseUnit,
  Warning,
  IndexStatus,
  SearchHit,
} from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";
import {
  EvidenceStatus,
  SurfaceKind,
  SurfaceStatus,
} from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";

type TargetMode = "scenario" | "path" | "module" | "project";
type FamilyLabelKey = (typeof strings.facts.family)[keyof typeof strings.facts.family];

interface FamilyOption {
  family: FactFamily;
  labelKey: FamilyLabelKey;
}

const FAMILY_OPTIONS: FamilyOption[] = [
  { family: FactFamily.SURFACES, labelKey: strings.facts.family.surfaces },
  { family: FactFamily.PARSE_UNITS, labelKey: strings.facts.family.parseUnits },
  { family: FactFamily.IMPORTS, labelKey: strings.facts.family.imports },
  { family: FactFamily.SYMBOLS, labelKey: strings.facts.family.symbols },
  { family: FactFamily.REFERENCES, labelKey: strings.facts.family.references },
  { family: FactFamily.CALLS, labelKey: strings.facts.family.calls },
  { family: FactFamily.PROTO_ADOPTION, labelKey: strings.facts.family.protoAdoption },
  { family: FactFamily.ENDPOINT_PROOFS, labelKey: strings.facts.family.endpointProofs },
];

const DEFAULT_FAMILIES = new Set<FactFamily>([
  FactFamily.SURFACES,
  FactFamily.PARSE_UNITS,
  FactFamily.IMPORTS,
  FactFamily.SYMBOLS,
  FactFamily.REFERENCES,
  FactFamily.CALLS,
  FactFamily.PROTO_ADOPTION,
  FactFamily.ENDPOINT_PROOFS,
]);

export function FactsWorkbench() {
  return (
    <div className="grid gap-5">
      <EvidenceSearchWorkspace />
      <EvidenceReportWorkbench />
    </div>
  );
}

function EvidenceSearchWorkspace() {
  const { t } = useTranslation();
  const [query, setQuery] = useState("provider demotion");
  const [scenario, setScenario] = useState("code-facts");
  const [scope, setScope] = useState("");
  const [role, setRole] = useState("");
  const [family, setFamily] = useState<FactFamily | "all">("all");
  const [selected, setSelected] = useState<SearchHit>();

  const statusQuery = useQuery({
    queryKey: ["code-facts-index-status"],
    queryFn: getIndexStatus,
    refetchInterval: 5_000,
  });
  const searchMutation = useMutation({
    mutationFn: () =>
      searchCodeFacts({
        query: query.trim(),
        target: scenarioTarget(scenario.trim()),
        families: family === "all" ? [] : [family],
        roles: role ? [role] : [],
        scope,
      }),
    onSuccess: (response) => setSelected(response.results[0]),
  });
  const status = statusQuery.data;
  const response = searchMutation.data;
  const degraded = [...(status?.degradedStages ?? []), ...(response?.degradedStages ?? [])];
  const search = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (query.trim() && scenario.trim()) searchMutation.mutate();
  };

  return (
    <section
      data-testid={selectors.facts.searchWorkspace}
      aria-labelledby="evidence-search-heading"
      className="overflow-hidden rounded-panel border border-app-border bg-app-surface"
    >
      <div className="border-b border-app-border bg-gradient-to-br from-app-primary/15 via-app-surface to-app-surface p-5 sm:p-7">
        <div className="flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
          <div className="max-w-3xl">
            <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">
              <Sparkles aria-hidden="true" className="h-4 w-4" /> Evidence workspace
            </div>
            <h2 id="evidence-search-heading" className="text-balance text-2xl font-semibold text-app-foreground sm:text-3xl">
              Find code by meaning. Verify every claim.
            </h2>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-app-muted-foreground">
              Hybrid retrieval ranks lexical and semantic evidence together, while preserving the analyzer, source range, generation, and proof state behind each result.
            </p>
          </div>
          <IndexPulse status={status} loading={statusQuery.isPending} />
        </div>

        <form onSubmit={search} className="mt-6 grid gap-3">
          <div className="flex min-w-0 flex-col gap-2 sm:flex-row">
            <label className="relative min-w-0 flex-1">
              <span className="sr-only">Search code evidence</span>
              <Search aria-hidden="true" className="pointer-events-none absolute left-3 top-3 h-5 w-5 text-app-muted-foreground" />
              <Input
                data-testid={selectors.facts.searchInput}
                value={query}
                onChange={(event) => setQuery(event.currentTarget.value)}
                placeholder="Try “where is provider demotion decided?”"
                className="h-11 border-app-border bg-app-background pl-10 text-app-foreground placeholder:text-app-muted-foreground"
              />
            </label>
            <Button
              data-testid={selectors.facts.searchButton}
              type="submit"
              disabled={searchMutation.isPending || !query.trim() || !scenario.trim()}
              className="h-11 gap-2 px-5"
            >
              {searchMutation.isPending ? <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" /> : <Search aria-hidden="true" className="h-4 w-4" />}
              Search evidence
            </Button>
          </div>
          <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
            <FilterField label="Corpus">
              <Input value={scenario} onChange={(event) => setScenario(event.currentTarget.value)} className="h-9 border-app-border bg-app-background" />
            </FilterField>
            <FilterField label="Scope">
              <select value={scope} onChange={(event) => setScope(event.currentTarget.value)} className="h-9 w-full rounded-control border border-app-border bg-app-background px-3 text-sm text-app-foreground">
                <option value="">Any scope</option><option value="api">API</option><option value="ui">UI</option><option value="cli">CLI</option>
              </select>
            </FilterField>
            <FilterField label="Role">
              <select value={role} onChange={(event) => setRole(event.currentTarget.value)} className="h-9 w-full rounded-control border border-app-border bg-app-background px-3 text-sm text-app-foreground">
                <option value="">Any role</option><option value="definition">Definition</option><option value="reference">Reference</option><option value="proof">Proof</option>
              </select>
            </FilterField>
            <FilterField label="Fact family">
              <select value={family} onChange={(event) => setFamily(event.currentTarget.value === "all" ? "all" : Number(event.currentTarget.value) as FactFamily)} className="h-9 w-full rounded-control border border-app-border bg-app-background px-3 text-sm text-app-foreground">
                <option value="all">All families</option><option value={FactFamily.SYMBOLS}>Symbols</option><option value={FactFamily.REFERENCES}>References</option><option value={FactFamily.CALLS}>Calls</option><option value={FactFamily.ENDPOINT_PROOFS}>Contract proofs</option>
              </select>
            </FilterField>
          </div>
        </form>
      </div>

      {degraded.length > 0 && (
        <div data-testid={selectors.facts.degradedBanner} role="status" className="flex gap-3 border-b border-amber-500/30 bg-amber-500/10 px-5 py-3 text-sm text-app-foreground">
          <AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
          <div><span className="font-semibold">Partial evidence.</span> {Array.from(new Set(degraded)).join(", ")}. Narrow the corpus or rebuild the index before treating absence as proof.</div>
        </div>
      )}
      {statusQuery.error && <InlineFailure message={`Index status unavailable: ${errorMessage(statusQuery.error, t)}`} />}
      {searchMutation.error && <InlineFailure message={errorMessage(searchMutation.error, t)} />}
      <CorpusDashboard status={status} />

      <div className="grid min-h-64 xl:grid-cols-[minmax(0,1fr)_23rem]">
        <div data-testid={selectors.facts.searchResults} className="min-w-0 p-4 sm:p-5">
          <ResultHeader response={response} pending={searchMutation.isPending} />
          {!response && !searchMutation.isPending && (
            <div className="grid place-items-center rounded-control border border-dashed border-app-border bg-app-surface-muted px-5 py-12 text-center">
              <Search aria-hidden="true" className="h-8 w-8 text-app-muted-foreground" />
              <p className="mt-3 font-medium text-app-foreground">Start with a behavior, symbol, or architectural question.</p>
              <p className="mt-1 max-w-md text-sm text-app-muted-foreground">Results expose why they ranked, where they came from, and whether their proof is current.</p>
            </div>
          )}
          {response && response.results.length === 0 && (
            <div className="rounded-control border border-dashed border-app-border p-8 text-center text-sm text-app-muted-foreground">No indexed evidence matched. Try fewer filters or inspect index readiness.</div>
          )}
          <div className="grid gap-3">
            {response?.results.map((hit, index) => (
              <ResultCard key={hit.id || `${hit.path}-${index}`} hit={hit} rank={index + 1} active={selected?.id === hit.id} onSelect={() => setSelected(hit)} />
            ))}
          </div>
        </div>
        <ProvenancePanel hit={selected} onClose={() => setSelected(undefined)} />
      </div>
    </section>
  );
}

function EvidenceReportWorkbench() {
  const { t } = useTranslation();
  const [targetMode, setTargetMode] = useState<TargetMode>("scenario");
  const [targetValue, setTargetValue] = useState("code-facts");
  const [useCache, setUseCache] = useState(true);
  const [selectedFamilies, setSelectedFamilies] = useState<Set<FactFamily>>(
    () => new Set(DEFAULT_FAMILIES),
  );

  const selectedFamilyList = useMemo(
    () => FAMILY_OPTIONS.map((option) => option.family).filter((family) => selectedFamilies.has(family)),
    [selectedFamilies],
  );

  const factsMutation = useMutation({
    mutationFn: () =>
      describeCodeFacts({
        target: buildTarget(targetMode, targetValue.trim()),
        include: selectedFamilyList.length > 0 ? selectedFamilyList : [FactFamily.SURFACES],
        useCache,
      }),
  });

  const report = factsMutation.data;
  const allEvidence = useMemo(() => collectEvidence(report), [report]);
  const degradedWarnings = report?.warnings.filter((warning) => warning.status !== EvidenceStatus.PROVEN) ?? [];

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!targetValue.trim()) return;
    factsMutation.mutate();
  };

  const toggleFamily = (family: FactFamily) => {
    setSelectedFamilies((current) => {
      const next = new Set(current);
      if (next.has(family)) {
        next.delete(family);
      } else {
        next.add(family);
      }
      return next;
    });
  };

  return (
    <section
      data-testid={selectors.facts.workbench}
      aria-label={t(strings.facts.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <form onSubmit={submit} className="grid gap-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <h2 className="flex items-center gap-2 text-sm font-semibold text-app-foreground">
              <FileSearch aria-hidden="true" className="h-4 w-4" />
              {t(strings.facts.title)}
            </h2>
            <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">{t(strings.facts.description)}</p>
          </div>
          <div
            data-testid={selectors.facts.cache}
            className="inline-flex w-fit items-center gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-xs text-app-muted-foreground"
          >
            <Database aria-hidden="true" className="h-3.5 w-3.5" />
            {report?.cache?.hit ? t(strings.facts.cacheHit) : t(strings.facts.cacheMiss)}
          </div>
        </div>

        <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto]">
          <div className="grid gap-3 md:grid-cols-[12rem_minmax(0,1fr)]">
            <label className="grid gap-1 text-xs font-medium text-app-muted-foreground">
              {t(strings.facts.targetKind)}
              <select
                data-testid={selectors.facts.targetKind}
                className="h-10 rounded-control border border-app-border bg-app-surface-muted px-3 text-sm text-app-foreground"
                value={targetMode}
                onChange={(event) => setTargetMode(event.currentTarget.value as TargetMode)}
              >
                <option value="scenario">{t(strings.facts.target.scenario)}</option>
                <option value="path">{t(strings.facts.target.path)}</option>
                <option value="module">{t(strings.facts.target.module)}</option>
                <option value="project">{t(strings.facts.target.project)}</option>
              </select>
            </label>
            <label className="grid gap-1 text-xs font-medium text-app-muted-foreground">
              {t(strings.facts.targetValue)}
              <Input
                data-testid={selectors.facts.targetInput}
                value={targetValue}
                onChange={(event) => setTargetValue(event.currentTarget.value)}
                placeholder={t(strings.facts.targetPlaceholder)}
                className="border-app-border bg-app-surface-muted text-app-foreground placeholder:text-app-muted-foreground"
              />
            </label>
          </div>

          <div className="flex flex-wrap items-end gap-2">
            <label className="inline-flex h-10 items-center gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 text-sm text-app-foreground">
              <input
                data-testid={selectors.facts.cacheToggle}
                type="checkbox"
                checked={useCache}
                onChange={(event) => setUseCache(event.currentTarget.checked)}
              />
              {t(strings.facts.useCache)}
            </label>
            <Button
              data-testid={selectors.facts.analyzeButton}
              type="submit"
              disabled={factsMutation.isPending || !targetValue.trim()}
              className="gap-2"
            >
              {factsMutation.isPending ? (
                <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
              ) : (
                <Play aria-hidden="true" className="h-4 w-4" />
              )}
              {factsMutation.isPending ? t(strings.facts.loading) : t(strings.facts.analyze)}
            </Button>
          </div>
        </div>

        <div
          data-testid={selectors.facts.familyControls}
          className="flex flex-wrap gap-2"
          aria-label={t(strings.facts.familyControls)}
        >
          {FAMILY_OPTIONS.map((option) => {
            const active = selectedFamilies.has(option.family);
            return (
              <button
                key={option.family}
                type="button"
                data-testid={selectors.facts.familyToggle({ family: String(option.family) })}
                className={`h-9 rounded-control border px-3 text-sm transition-colors ${
                  active
                    ? "border-app-primary bg-app-primary text-app-primary-foreground"
                    : "border-app-border bg-app-surface-muted text-app-muted-foreground hover:text-app-foreground"
                }`}
                aria-pressed={active}
                onClick={() => toggleFamily(option.family)}
              >
                {t(option.labelKey)}
              </button>
            );
          })}
        </div>
      </form>

      {factsMutation.isPending && (
        <p data-testid={selectors.facts.loading} className="mt-4 text-sm text-app-muted-foreground">
          {t(strings.facts.loadingDetail)}
        </p>
      )}
      {factsMutation.error && (
        <div
          data-testid={selectors.facts.error}
          className="mt-4 rounded-control border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {errorMessage(factsMutation.error, t)}
        </div>
      )}
      {!factsMutation.isPending && !factsMutation.error && !report && (
        <div
          data-testid={selectors.facts.empty}
          className="mt-4 rounded-control border border-dashed border-app-border bg-app-surface-muted p-4 text-sm text-app-muted-foreground"
        >
          {t(strings.facts.empty)}
        </div>
      )}

      {report && (
        <div className="mt-4 grid gap-4">
          <SummaryGrid report={report} evidenceCount={allEvidence.length} warningCount={degradedWarnings.length} />
          <TargetContextPanel report={report} />
          <CachePanel report={report} />
          <InventoryPanel report={report} />
          <FactsTable facts={report.facts} />
          <EvidenceTable evidence={allEvidence} />
          <WarningsPanel warnings={report.warnings} />
          <RawJsonPanel report={report} />
        </div>
      )}
    </section>
  );
}

function FilterField({ label, children }: { label: string; children: React.ReactNode }) {
  return <label className="grid min-w-0 gap-1 text-xs font-medium text-app-muted-foreground"><span>{label}</span>{children}</label>;
}

function IndexPulse({ status, loading }: { status?: IndexStatus; loading: boolean }) {
  const ready = Boolean(status?.activeGeneration);
  const count = status?.searchDocuments ?? 0n;
  return (
    <div data-testid={selectors.facts.indexStatus} className="grid min-w-64 grid-cols-[auto_1fr] gap-x-3 gap-y-1 rounded-control border border-app-border bg-app-background/80 px-4 py-3 shadow-sm">
      {loading ? <Loader2 aria-hidden="true" className="row-span-2 mt-1 h-5 w-5 animate-spin text-app-primary" /> : ready ? <CheckCircle2 aria-hidden="true" className="row-span-2 mt-1 h-5 w-5 text-emerald-500" /> : <AlertTriangle aria-hidden="true" className="row-span-2 mt-1 h-5 w-5 text-amber-500" />}
      <span className="text-sm font-semibold text-app-foreground">{loading ? "Checking index" : ready ? "Index ready" : "Index not promoted"}</span>
      <span className="text-xs text-app-muted-foreground">{ready ? `${count.toLocaleString()} documents · ${status?.state || "ready"}` : "Search may use bounded transitional evidence"}</span>
    </div>
  );
}

function ResultHeader({ response, pending }: { response?: { results: SearchHit[]; retrievalRegime: string; generation: string }; pending: boolean }) {
  return (
    <div className="mb-3 flex min-h-7 flex-wrap items-center justify-between gap-2">
      <h3 className="text-sm font-semibold text-app-foreground">{pending ? "Ranking evidence…" : response ? `${response.results.length} ranked results` : "Ranked evidence"}</h3>
      {response && <div className="flex flex-wrap gap-2 text-xs text-app-muted-foreground"><Badge icon={<Gauge className="h-3 w-3" />} label={response.retrievalRegime || "lexical"} /><Badge icon={<Database className="h-3 w-3" />} label={response.generation || "transitional"} /></div>}
    </div>
  );
}

function CorpusDashboard({ status }: { status?: IndexStatus }) {
  const jobs = status?.activeJobs ?? [];
  return (
    <details className="border-b border-app-border bg-app-background/50 px-5 py-3">
      <summary className="flex cursor-pointer list-none flex-wrap items-center justify-between gap-3 text-sm font-semibold text-app-foreground">
        <span className="flex items-center gap-2"><Database aria-hidden="true" className="h-4 w-4 text-app-primary" />Corpus readiness</span>
        <span className="font-normal text-app-muted-foreground">{status ? `${status.sourceFiles.toLocaleString()} files · ${formatBytes(status.storageBytes)}` : "Status unavailable"}</span>
      </summary>
      <div className="mt-4 grid gap-3 lg:grid-cols-4">
        <CorpusStat label="Search documents" value={status?.searchDocuments} />
        <CorpusStat label="Semantic cards" value={status?.semanticCards} />
        <CorpusStat label="Graph facts" value={status?.graphFacts} />
        <CorpusStat label="Last reconcile" value={status?.lastReconcileAtUnix ? relativeTime(status.lastReconcileAtUnix) : "Never"} />
      </div>
      <div className="mt-4 flex flex-col gap-3 rounded-control border border-app-border bg-app-surface p-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <p className="text-sm font-medium text-app-foreground">Generation controls</p>
          <p className="mt-1 text-xs text-app-muted-foreground">Rebuilds are confirmation-gated and publish only after a complete shadow generation passes validation.</p>
        </div>
        <Button type="button" variant="outline" size="sm" disabled title="Index mutation runtime will become available after indexed-provider cutover" className="shrink-0 gap-2"><RefreshCcw aria-hidden="true" className="h-4 w-4" />Rebuild unavailable</Button>
      </div>
      {jobs.length > 0 && <div className="mt-3 grid gap-2">{jobs.map((job) => {
        const total = Number(job.total || 0n);
        const processed = Number(job.processed || 0n);
        const percent = total > 0 ? Math.min(100, Math.round((processed / total) * 100)) : 0;
        return <div key={job.id} className="rounded-control border border-app-border bg-app-surface p-3 text-xs"><div className="flex justify-between gap-3"><span className="font-medium text-app-foreground">{job.kind} · {job.id}</span><span className="font-mono text-app-muted-foreground">{percent}%</span></div><div className="mt-2 h-1.5 overflow-hidden rounded-full bg-app-surface-muted" role="progressbar" aria-label={`${job.kind} progress`} aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}><div className="h-full bg-app-primary" style={{ width: `${percent}%` }} /></div></div>;
      })}</div>}
    </details>
  );
}

function CorpusStat({ label, value }: { label: string; value?: bigint | string }) {
  return <div className="rounded-control border border-app-border bg-app-surface p-3"><p className="text-xs text-app-muted-foreground">{label}</p><p className="mt-1 text-lg font-semibold text-app-foreground">{typeof value === "bigint" ? value.toLocaleString() : value ?? "—"}</p></div>;
}

function ResultCard({ hit, rank, active, onSelect }: { hit: SearchHit; rank: number; active: boolean; onSelect: () => void }) {
  return (
    <button
      type="button"
      data-testid={selectors.facts.resultCard}
      onClick={onSelect}
      aria-pressed={active}
      className={`w-full rounded-control border p-4 text-left transition ${active ? "border-app-primary bg-app-primary/5 shadow-sm" : "border-app-border bg-app-surface-muted hover:border-app-primary/50"}`}
    >
      <div className="flex min-w-0 items-start gap-3">
        <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-app-background text-xs font-semibold text-app-muted-foreground">{rank}</span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <h4 className="min-w-0 truncate font-semibold text-app-foreground">{hit.title || hit.id}</h4>
            <span className="shrink-0 font-mono text-xs font-semibold text-app-primary">{hit.score.toFixed(3)}</span>
          </div>
          <p className="mt-1 line-clamp-2 text-sm leading-5 text-app-muted-foreground">{hit.text || "No excerpt returned."}</p>
          <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-app-muted-foreground">
            <Badge icon={<ShieldCheck className="h-3 w-3" />} label={hit.proofStatus || enumLabel(EvidenceStatus, hit.evidenceStatus)} />
            <Badge icon={<CircleDot className="h-3 w-3" />} label={hit.factKind || hit.role || "fact"} />
            <span className="min-w-0 truncate font-mono">{sourceLabel(hit)}</span>
            <ArrowRight aria-hidden="true" className="ml-auto h-4 w-4" />
          </div>
        </div>
      </div>
    </button>
  );
}

function ProvenancePanel({ hit, onClose }: { hit?: SearchHit; onClose: () => void }) {
  return (
    <div data-testid={selectors.facts.provenancePanel} role="region" aria-label="Evidence provenance" className="min-w-0 border-t border-app-border bg-app-surface-muted p-4 xl:border-l xl:border-t-0">
      {!hit ? (
        <div className="grid h-full min-h-48 place-items-center text-center text-sm text-app-muted-foreground"><div><GitBranch aria-hidden="true" className="mx-auto h-7 w-7" /><p className="mt-2">Select a result to inspect provenance and relationships.</p></div></div>
      ) : (
        <div className="grid gap-5">
          <div className="flex items-start justify-between gap-3">
            <div><p className="text-xs font-semibold uppercase tracking-wider text-app-primary">Evidence detail</p><h3 className="mt-1 break-words font-semibold text-app-foreground">{hit.title || hit.id}</h3></div>
            <button type="button" aria-label="Close evidence detail" onClick={onClose} className="rounded-control p-2 text-app-muted-foreground hover:bg-app-background hover:text-app-foreground"><X aria-hidden="true" className="h-4 w-4" /></button>
          </div>
          <dl className="grid gap-3 text-xs">
            <Detail label="Source" value={sourceLabel(hit)} />
            <Detail label="Analyzer" value={hit.analyzer || "unknown"} />
            <Detail label="Generation" value={hit.generation || "transitional"} />
            <Detail label="Retrieval" value={hit.retrievalExplanation || hit.retrievalRegime || "lexical match"} />
            <Detail label="Source hash" value={hit.sourceHash || "not supplied"} />
          </dl>
          <section>
            <h4 className="flex items-center gap-2 text-sm font-semibold text-app-foreground"><Gauge aria-hidden="true" className="h-4 w-4" />Why this ranked</h4>
            {hit.rankFactors.length === 0 ? <p className="mt-2 text-xs text-app-muted-foreground">The active provider did not expose component scores.</p> : <ul className="mt-2 grid gap-2">{hit.rankFactors.map((factor) => <li key={`${factor.leg}-${factor.name}`} className="flex justify-between gap-3 rounded-control border border-app-border bg-app-background px-3 py-2 text-xs"><span>{factor.name} · {factor.leg}</span><span className="font-mono font-semibold">{factor.value.toFixed(3)}</span></li>)}</ul>}
          </section>
          <section>
            <h4 className="flex items-center gap-2 text-sm font-semibold text-app-foreground"><GitBranch aria-hidden="true" className="h-4 w-4" />Relationships</h4>
            {hit.edgeExpansions.length === 0 ? <p className="mt-2 text-xs text-app-muted-foreground">No bounded graph neighbors were returned.</p> : <ul className="mt-2 grid gap-2">{hit.edgeExpansions.map((edge) => <li key={edge.id} className="rounded-control border border-app-border bg-app-background p-3"><p className="text-sm font-medium text-app-foreground">{edge.title || edge.id}</p><p className="mt-1 truncate font-mono text-xs text-app-muted-foreground">{edge.path}</p></li>)}</ul>}
          </section>
        </div>
      )}
    </div>
  );
}

function Badge({ icon, label }: { icon: React.ReactNode; label: string }) {
  return <span className="inline-flex max-w-full items-center gap-1 rounded-full border border-app-border bg-app-background px-2 py-1"><span aria-hidden="true">{icon}</span><span className="truncate">{label}</span></span>;
}

function Detail({ label, value }: { label: string; value: string }) {
  return <div className="min-w-0"><dt className="text-app-muted-foreground">{label}</dt><dd className="mt-0.5 break-all font-mono text-app-foreground">{value}</dd></div>;
}

function InlineFailure({ message }: { message: string }) {
  return <div role="alert" className="flex gap-2 border-b border-red-500/30 bg-red-500/10 px-5 py-3 text-sm text-app-foreground"><AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-red-500" />{message}</div>;
}

function sourceLabel(hit: SearchHit): string {
  if (!hit.path) return "source unavailable";
  if (hit.startLine <= 0) return hit.path;
  return `${hit.path}:${hit.startLine}${hit.endLine > hit.startLine ? `–${hit.endLine}` : ""}`;
}

function formatBytes(value: bigint): string {
  const bytes = Number(value);
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB"];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / 1024 ** exponent).toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}

function relativeTime(unixSeconds: bigint): string {
  const elapsed = Math.max(0, Date.now() - Number(unixSeconds) * 1_000);
  const minutes = Math.floor(elapsed / 60_000);
  if (minutes < 1) return "Just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function buildTarget(mode: TargetMode, value: string): CodeTarget {
  if (mode === "path") return pathTarget(value);
  if (mode === "module") return moduleTarget(value);
  if (mode === "project") return projectTarget(value);
  return scenarioTarget(value);
}

function collectEvidence(report?: CodeFactsReport): Evidence[] {
  if (!report) return [];
  return [
    ...report.evidence,
    ...report.surfaces.flatMap((surface) => surface.evidence),
    ...report.parseUnits.flatMap((unit) => unit.evidence),
    ...report.facts.flatMap((fact) => fact.evidence),
  ];
}

function SummaryGrid({
  report,
  evidenceCount,
  warningCount,
}: {
  report: CodeFactsReport;
  evidenceCount: number;
  warningCount: number;
}) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.facts.summary} className="grid gap-3 md:grid-cols-5">
      <FactStat label={t(strings.facts.surfaces)} value={report.surfaces.length} />
      <FactStat label={t(strings.facts.parseUnits)} value={report.parseUnits.length} />
      <FactStat label={t(strings.facts.facts)} value={report.facts.length} />
      <FactStat label={t(strings.facts.evidence)} value={evidenceCount} />
      <FactStat label={t(strings.facts.warnings)} value={warningCount} />
    </div>
  );
}

function TargetContextPanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.facts.targetContext}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
      aria-label={t(strings.facts.targetContext)}
    >
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-app-foreground">
        <Layers3 aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.targetContext)}
      </div>
      <dl className="grid gap-2 text-xs md:grid-cols-4">
        <InfoField label={t(strings.facts.resolvedKind)} value={enumLabel(TargetKind, report.target?.resolvedKind)} />
        <InfoField label={t(strings.facts.scenario)} value={report.target?.scenario || "-"} />
        <InfoField label={t(strings.facts.scenarioAware)} value={report.target?.scenarioAware ? "true" : "false"} />
        <InfoField label={t(strings.facts.rootPath)} value={report.target?.rootPath || "-"} wide />
      </dl>
    </section>
  );
}

function CachePanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.facts.cachePanel}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
      aria-label={t(strings.facts.cachePanel)}
    >
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-app-foreground">
        <RefreshCcw aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.cachePanel)}
      </div>
      <dl className="grid gap-2 text-xs md:grid-cols-4">
        <InfoField label={t(strings.facts.cacheState)} value={report.cache?.state || "miss"} />
        <InfoField label={t(strings.facts.cacheKey)} value={report.cache?.cacheKey || "-"} wide />
        <InfoField label={t(strings.facts.cacheReason)} value={report.cache?.reason || "-"} />
        <InfoField label={t(strings.facts.cacheScope)} value={report.cache?.scope || "-"} />
        <InfoField label={t(strings.facts.sourceHash)} value={report.cache?.sourceHash || "-"} />
        <InfoField label={t(strings.facts.configHash)} value={report.cache?.configHash || "-"} />
        <InfoField label={t(strings.facts.providerVersion)} value={report.cache?.providerVersion || "-"} />
        <InfoField label={t(strings.facts.hitCount)} value={String(report.cache?.hitCount ?? 0)} />
      </dl>
    </section>
  );
}

function InventoryPanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  return (
    <section className="grid gap-4 lg:grid-cols-2">
      <DataTable
        testId={selectors.facts.surfacesTable}
        title={t(strings.facts.surfaceInventory)}
        empty={t(strings.facts.noSurfaces)}
        headers={[
          t(strings.facts.table.id),
          t(strings.facts.table.kind),
          t(strings.facts.table.status),
          t(strings.facts.table.path),
        ]}
        rows={report.surfaces.map((surface) => [
          surface.id,
          enumLabel(SurfaceKind, surface.kind),
          enumLabel(SurfaceStatus, surface.status),
          surface.path,
        ])}
      />
      <DataTable
        testId={selectors.facts.parseUnitsTable}
        title={t(strings.facts.parseUnitInventory)}
        empty={t(strings.facts.noParseUnits)}
        headers={[
          t(strings.facts.table.id),
          t(strings.facts.table.language),
          t(strings.facts.table.status),
          t(strings.facts.table.path),
        ]}
        rows={report.parseUnits.map((unit) => parseUnitRow(unit))}
      />
    </section>
  );
}

function FactsTable({ facts }: { facts: GenericFact[] }) {
  const { t } = useTranslation();
  return (
    <DataTable
      testId={selectors.facts.factsTable}
      title={t(strings.facts.factsTable)}
      empty={t(strings.facts.noFacts)}
      headers={[
        t(strings.facts.table.family),
        t(strings.facts.table.kind),
        t(strings.facts.table.subject),
        t(strings.facts.table.status),
        t(strings.facts.table.attributes),
      ]}
      rows={facts.map((fact) => [
        enumLabel(FactFamily, fact.family),
        fact.kind,
        fact.subject,
        summarizeEvidenceStatus(fact.evidence),
        attributesSummary(fact.attributes),
      ])}
    />
  );
}

function EvidenceTable({ evidence }: { evidence: Evidence[] }) {
  const { t } = useTranslation();
  return (
    <DataTable
      testId={selectors.facts.evidenceTable}
      title={t(strings.facts.evidenceTable)}
      empty={t(strings.facts.noEvidence)}
      headers={[
        t(strings.facts.table.status),
        t(strings.facts.table.file),
        t(strings.facts.table.range),
        t(strings.facts.table.analyzer),
        t(strings.facts.table.message),
      ]}
      rows={evidence.map((entry) => [
        enumLabel(EvidenceStatus, entry.status),
        entry.range?.file || "-",
        formatRange(entry),
        entry.analyzer || "-",
        entry.message || entry.symbol || "-",
      ])}
    />
  );
}

function WarningsPanel({ warnings }: { warnings: Warning[] }) {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.facts.warningsPanel}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
      aria-label={t(strings.facts.warningsPanel)}
    >
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-app-foreground">
        <AlertTriangle aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.warningsPanel)}
      </div>
      {warnings.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">{t(strings.facts.noWarnings)}</p>
      ) : (
        <ul className="grid gap-2 text-sm">
          {warnings.map((warning, index) => (
            <li key={`${warning.code}-${index}`} className="rounded-control border border-app-border bg-app-surface p-2">
              <span className="font-mono text-xs text-app-muted-foreground">{warning.code}</span>
              <span className="ml-2 font-medium text-app-foreground">{enumLabel(EvidenceStatus, warning.status)}</span>
              <p className="mt-1 text-app-muted-foreground">{warning.message}</p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function RawJsonPanel({ report }: { report: CodeFactsReport }) {
  const { t } = useTranslation();
  const json = useMemo(() => JSON.stringify(report, jsonReplacer, 2), [report]);
  return (
    <details
      data-testid={selectors.facts.rawJson}
      className="rounded-control border border-app-border bg-app-surface-muted p-3"
    >
      <summary className="flex cursor-pointer items-center gap-2 text-sm font-semibold text-app-foreground">
        <Braces aria-hidden="true" className="h-4 w-4" />
        {t(strings.facts.rawJson)}
      </summary>
      <pre className="mt-3 max-h-96 overflow-auto rounded-control border border-app-border bg-app-background p-3 text-xs text-app-foreground">
        {json}
      </pre>
    </details>
  );
}

function DataTable({
  testId,
  title,
  empty,
  headers,
  rows,
}: {
  testId: string;
  title: string;
  empty: string;
  headers: string[];
  rows: string[][];
}) {
  return (
    <section data-testid={testId} className="rounded-control border border-app-border bg-app-surface-muted p-3">
      <h3 className="text-sm font-semibold text-app-foreground">{title}</h3>
      {rows.length === 0 ? (
        <p className="mt-2 text-sm text-app-muted-foreground">{empty}</p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full table-fixed text-left text-xs">
            <thead className="text-app-muted-foreground">
              <tr>
                {headers.map((header) => (
                  <th key={header} scope="col" className="border-b border-app-border px-2 py-2 font-medium">
                    {header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, rowIndex) => (
                <tr key={rowIndex} className="border-b border-app-border/60 last:border-0">
                  {row.map((cell, cellIndex) => (
                    <td key={`${rowIndex}-${cellIndex}`} className="max-w-64 truncate px-2 py-2" title={cell}>
                      {cell || "-"}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function InfoField({
  label,
  value,
  wide = false,
}: {
  label: string;
  value: string;
  wide?: boolean;
}) {
  return (
    <div className={`min-w-0 ${wide ? "md:col-span-2" : ""}`}>
      <dt className="text-app-muted-foreground">{label}</dt>
      <dd className="truncate font-mono text-app-foreground" title={value}>
        {value || "-"}
      </dd>
    </div>
  );
}

function FactStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
      <p className="truncate text-xs uppercase text-app-muted-foreground" title={label}>
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function parseUnitRow(unit: ParseUnit): string[] {
  return [
    unit.id,
    unit.language,
    enumLabel(EvidenceStatus, unit.status),
    unit.configPath || unit.rootPath,
  ];
}

function enumLabel(enumObject: Record<string, string | number>, value: number | undefined): string {
  if (value === undefined) return "-";
  const raw = enumObject[value];
  if (typeof raw !== "string") return String(value);
  return raw.toLowerCase().replace(/_/g, " ");
}

function summarizeEvidenceStatus(evidence: Evidence[]): string {
  if (evidence.length === 0) return "-";
  const precedence = [
    EvidenceStatus.CONTRADICTED,
    EvidenceStatus.MISSING,
    EvidenceStatus.UNKNOWN,
    EvidenceStatus.UNSUPPORTED,
    EvidenceStatus.PROVEN,
  ];
  const status = precedence.find((candidate) => evidence.some((entry) => entry.status === candidate));
  return enumLabel(EvidenceStatus, status);
}

function formatRange(entry: Evidence): string {
  const range = entry.range;
  if (!range || range.startLine === 0) return "-";
  if (range.endLine > 0 && range.endLine !== range.startLine) {
    return `${range.startLine}:${range.startColumn}-${range.endLine}:${range.endColumn}`;
  }
  return `${range.startLine}:${range.startColumn}`;
}

function attributesSummary(attributes: Record<string, string>): string {
  const entries = Object.entries(attributes);
  if (entries.length === 0) return "-";
  return entries.map(([key, value]) => `${key}=${value}`).join(", ");
}

function jsonReplacer(_key: string, value: unknown) {
  return typeof value === "bigint" ? value.toString() : value;
}
