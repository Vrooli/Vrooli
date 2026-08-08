import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, ExternalLink, RefreshCw, ShieldCheck, StopCircle } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { StatusBadge } from "../ui/status-badge";
import { Button } from "../ui/button";

type Target = {
  kind: string;
  os: string;
  architecture: string;
  mode: string;
  descriptor?: {
    target_id?: string;
    display_name?: string;
    available?: boolean;
    capabilities?: number[];
    reason?: string;
  };
};

type Inventory = { targets: Target[] };

type Journey = {
  journey_id: string;
  display_name: string;
  source_path?: string;
  execution_mode?: string;
  required: boolean;
  required_capabilities?: number[];
  category?: string;
  requirements?: string[];
  estimated_duration_seconds?: number;
  safety?: {
    mutating?: boolean;
    requires_isolation?: boolean;
    requires_confirmation?: boolean;
  };
};

type CatalogResponse = { journeys?: Journey[] };

type EnvironmentProfile = { value: number; label: string; description: string };

type Evidence = {
  kind?: number | string;
  evidence_id?: string;
  uri?: string;
  sha256?: string;
  media_type?: string;
  redacted?: boolean;
};

type Cell = {
  cell?: {
    cell_id?: string;
    target_id?: string;
    journey_id?: string;
    environment_profile?: number | string;
    disposition?: number | string;
    reason?: string;
    evidence?: Evidence[];
    required?: boolean;
    applicable?: boolean;
  };
  state?: string;
  attempts?: number;
  updated_at?: string;
};

type Gate = {
  passed?: boolean;
  disposition?: number | string;
  required_cell_count?: number;
  passing_cell_count?: number;
  missing_cell_ids?: string[];
  failed_cell_ids?: string[];
  reason?: string;
};

type MatrixRun = {
  run_id: string;
  state?: string;
  selection?: { scenario_name?: string; artifact_digest?: string; deployment_mode?: string; release_profile?: string; journeys?: Journey[]; targets?: TargetSelection[]; environment_profiles?: (number | string)[] };
  cells?: Cell[];
  gate?: Gate;
  parent_run_id?: string;
  created_at?: string;
};

type TargetSelection = { descriptor: Target["descriptor"]; kind: string };

const ENVIRONMENT_PROFILES: EnvironmentProfile[] = [
  { value: 1, label: "Normal", description: "Healthy network and credentials" },
  { value: 2, label: "Offline", description: "No external network access" },
  { value: 3, label: "Slow network", description: "Constrained network timing" },
  { value: 4, label: "Missing credential", description: "Required credential is absent" },
  { value: 5, label: "Provider failure", description: "Validation provider is unavailable" },
  { value: 6, label: "Interrupted update", description: "Update stops part way through" },
  { value: 7, label: "Crash recovery", description: "Application restarts during validation" },
];

const DISPOSITIONS: Record<string, { label: string; explanation: string; tone: "neutral" | "success" | "warning" | "danger" | "info" }> = {
  "0": { label: "unspecified", explanation: "The provider has not reported a disposition.", tone: "neutral" },
  "1": { label: "pass", explanation: "Required validation and evidence are present.", tone: "success" },
  "2": { label: "failed", explanation: "Validation produced a failing assertion.", tone: "danger" },
  "3": { label: "degraded", explanation: "Validation completed with reduced evidence or capability.", tone: "warning" },
  "4": { label: "unavailable", explanation: "The selected target or dependency was unavailable.", tone: "warning" },
  "5": { label: "unsupported", explanation: "This journey cannot run with the selected target capabilities.", tone: "warning" },
  "6": { label: "refused", explanation: "Execution was refused by a safety, lease, or policy check.", tone: "danger" },
  "7": { label: "not run", explanation: "This required cell was omitted or has not executed yet.", tone: "warning" },
};

const PROFILE_LABELS = Object.fromEntries(ENVIRONMENT_PROFILES.map((profile) => [profile.value, profile.label]));

const PLATFORM_JOURNEY: Journey = {
  journey_id: "platform-desktop-smoke",
  display_name: "Platform desktop smoke",
  source_path: "scenario-to-desktop platform suite",
  execution_mode: "platform",
  required: true,
  category: "platform",
};

function enumKey(value: number | string | undefined): string {
  if (typeof value === "number") return String(value);
  if (!value) return "0";
  const match = value.match(/(UNSPECIFIED|PASS|FAILED|DEGRADED|UNAVAILABLE|UNSUPPORTED|REFUSED|NOT_RUN|NORMAL|OFFLINE|SLOW_NETWORK|MISSING_CREDENTIAL|PROVIDER_FAILURE|UPDATE_INTERRUPTED|CRASH_RECOVERY)$/);
  if (!match) return value;
  const names: Record<string, string> = {
    UNSPECIFIED: "0", PASS: "1", FAILED: "2", DEGRADED: "3", UNAVAILABLE: "4", UNSUPPORTED: "5", REFUSED: "6", NOT_RUN: "7",
    NORMAL: "1", OFFLINE: "2", SLOW_NETWORK: "3", MISSING_CREDENTIAL: "4", PROVIDER_FAILURE: "5", UPDATE_INTERRUPTED: "6", CRASH_RECOVERY: "7",
  };
  const suffix = match[1];
  return suffix ? (names[suffix] ?? value) : value;
}

function disposition(value: number | string | undefined) {
  return DISPOSITIONS[enumKey(value)] ?? { label: "unknown", explanation: "The provider returned an unknown state; it cannot pass the gate.", tone: "warning" as const };
}

function profileLabel(value: number | string | undefined) {
  return PROFILE_LABELS[Number(enumKey(value))] ?? (typeof value === "string" ? value : "unspecified");
}

async function requestJSON<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, init);
  if (!response.ok) {
    let detail = "";
    try {
      const body = (await response.json()) as { error?: string };
      detail = body.error ? `: ${body.error}` : "";
    } catch {
      // Keep the HTTP status when the server did not return JSON.
    }
    throw new Error(`validation request failed (${String(response.status)})${detail}`);
  }
  return (await response.json()) as T;
}

export function ValidationWorkspace() {
  const [inventory, setInventory] = useState<Inventory | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [scenarioName, setScenarioName] = useState("scenario-to-desktop");
  const [artifactDigest, setArtifactDigest] = useState("");
  const [deploymentMode, setDeploymentMode] = useState("release-candidate");
  const [releaseProfile, setReleaseProfile] = useState("desktop-standard");
  const [catalogJourneys, setCatalogJourneys] = useState<Journey[]>([PLATFORM_JOURNEY]);
  const [selectedJourneyIds, setSelectedJourneyIds] = useState<string[]>([PLATFORM_JOURNEY.journey_id]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogError, setCatalogError] = useState<string | null>(null);
  const [selectedTargetIds, setSelectedTargetIds] = useState<string[]>([]);
  const [selectedProfiles, setSelectedProfiles] = useState<number[]>([1]);
  const [run, setRun] = useState<MatrixRun | null>(null);
  const [runID, setRunID] = useState("");
  const [runLoading, setRunLoading] = useState(false);
  const [matrixError, setMatrixError] = useState<string | null>(null);
  const [selectedCell, setSelectedCell] = useState<Cell | null>(null);
  const [selectedCellIds, setSelectedCellIds] = useState<string[]>([]);
  const [priorRunID, setPriorRunID] = useState("");
  const [comparison, setComparison] = useState<{ changed?: boolean; cells?: { changed?: boolean }[] } | null>(null);
  const [comparisonError, setComparisonError] = useState<string | null>(null);

  const loadCatalog = async () => {
    setCatalogLoading(true);
    setCatalogError(null);
    try {
      const response = await requestJSON<CatalogResponse>(`/api/v1/validation/catalog?scenario=${encodeURIComponent(scenarioName.trim())}`);
      const seen = new Set<string>();
      const merged = [PLATFORM_JOURNEY, ...(response.journeys ?? [])].filter((item) => {
        if (!item.journey_id || seen.has(item.journey_id)) return false;
        seen.add(item.journey_id);
        return true;
      });
      setCatalogJourneys(merged);
      setSelectedJourneyIds((current) => current.filter((id) => merged.some((item) => item.journey_id === id)));
    } catch (cause) {
      setCatalogError(cause instanceof Error ? cause.message : "journey catalog unavailable");
    } finally {
      setCatalogLoading(false);
    }
  };

  const loadInventory = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch("/api/v1/validation/targets");
      if (!response.ok) throw new Error(`target inventory failed (${String(response.status)})`);
      const nextInventory = (await response.json()) as Inventory;
      setInventory(nextInventory);
      setSelectedTargetIds((current) => current.length > 0 ? current : nextInventory.targets.filter((target) => target.descriptor?.available).map((target) => target.descriptor?.target_id).filter((id): id is string => Boolean(id)));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "target inventory unavailable");
    } finally {
      setLoading(false);
    }
  }, []);

  const availableTargets = useMemo(() => inventory?.targets ?? [], [inventory]);
  const selectedTargets = availableTargets.filter((target) => selectedTargetIds.includes(target.descriptor?.target_id ?? ""));
  const selectedJourneys = catalogJourneys.filter((item) => selectedJourneyIds.includes(item.journey_id));
  const catalogGroups = useMemo(() => {
    const groups = new Map<string, Journey[]>();
    for (const journey of catalogJourneys) {
      const category = journey.category ?? "provider journey";
      groups.set(category, [...(groups.get(category) ?? []), journey]);
    }
    return Array.from(groups.entries());
  }, [catalogJourneys]);

  const createMatrix = async () => {
    setRunLoading(true);
    setMatrixError(null);
    try {
      const created = await requestJSON<MatrixRun>("/api/v1/validation/matrices", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          scenario_name: scenarioName.trim(),
          artifact_digest: artifactDigest.trim(),
          deployment_mode: deploymentMode,
          release_profile: releaseProfile,
          idempotency_key: `workspace-${scenarioName.trim()}-${artifactDigest.trim()}-${selectedJourneyIds.join(".")}-${selectedTargetIds.join(".")}-${selectedProfiles.join(".")}`,
          journeys: selectedJourneys,
          targets: selectedTargets.map((target) => ({ descriptor: target.descriptor, kind: target.kind })),
          environment_profiles: selectedProfiles,
          max_concurrency: 1,
        }),
      });
      setRun(created);
      setRunID(created.run_id);
    } catch (cause) {
      setMatrixError(cause instanceof Error ? cause.message : "matrix creation failed");
    } finally {
      setRunLoading(false);
    }
  };

  const loadRun = async (path: string, method = "GET", body?: unknown) => {
    if (!runID.trim()) return;
    setRunLoading(true);
    setMatrixError(null);
    try {
      const loaded = await requestJSON<MatrixRun>(`/api/v1/validation/matrices/${encodeURIComponent(runID.trim())}${path}`, {
        method,
        headers: body ? { "Content-Type": "application/json" } : undefined,
        body: body ? JSON.stringify(body) : undefined,
      });
      setRun(loaded);
    } catch (cause) {
      setMatrixError(cause instanceof Error ? cause.message : "matrix run unavailable");
    } finally {
      setRunLoading(false);
    }
  };

  const startRun = () => void loadRun("/start", "POST");
  const waitForRun = () => void loadRun("/wait");
  const abortRun = () => void loadRun("/abort", "POST");
  const rerunFailed = () => void loadRun("/rerun", "POST", { kind: "failed" });

  const rerunSelected = async () => {
    if (!runID.trim() || selectedCellIds.length === 0) return;
    setRunLoading(true);
    setMatrixError(null);
    try {
      let latest: MatrixRun | null = null;
      for (const cellID of selectedCellIds) {
        latest = await requestJSON<MatrixRun>(`/api/v1/validation/matrices/${encodeURIComponent(runID.trim())}/rerun`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ kind: "cell", cell_id: cellID }),
        });
      }
      if (latest) setRun(latest);
      setSelectedCellIds([]);
    } catch (cause) {
      setMatrixError(cause instanceof Error ? cause.message : "selected cell rerun failed");
    } finally {
      setRunLoading(false);
    }
  };

  const compareRuns = async () => {
    if (!runID.trim() || !priorRunID.trim()) return;
    setComparisonError(null);
    try {
      setComparison(await requestJSON<{ changed?: boolean; cells?: { changed?: boolean }[] }>(`/api/v1/validation/matrices/${encodeURIComponent(runID.trim())}/compare/${encodeURIComponent(priorRunID.trim())}`));
    } catch (cause) {
      setComparisonError(cause instanceof Error ? cause.message : "comparison unavailable");
    }
  };

  const runState = run?.state ?? "not created";
  const gate = run?.gate;
  const swimlanes = (run?.selection?.targets ?? []).map((target) => {
    const targetID = target.descriptor?.target_id ?? "unknown";
    const cells = (run?.cells ?? []).filter((record) => record.cell?.target_id === targetID);
    const completed = cells.filter((record) => record.state === "completed").length;
    const evidence = cells.reduce((total, record) => total + (record.cell?.evidence?.length ?? 0), 0);
    const active = cells.find((record) => record.state === "running" || record.state === "retrying");
    return { target, total: cells.length, completed, evidence, active: active?.cell?.journey_id ?? "idle" };
  });

  useEffect(() => {
    void loadInventory();
  }, [loadInventory]);

  return (
    <section
      aria-labelledby="validation-workspace-heading"
      data-testid={selectors.validation.root}
      className="space-y-5"
    >
      <div className="flex flex-wrap items-start justify-between gap-3 rounded-2xl border border-blue-500/30 bg-blue-950/20 p-5">
        <div>
          <div className="mb-2 flex items-center gap-2 text-blue-300">
            <ShieldCheck className="h-5 w-5" aria-hidden="true" />
            <span className="text-xs font-semibold uppercase tracking-wider">Validation workspace</span>
          </div>
          <h2 id="validation-workspace-heading" className="text-2xl font-semibold text-white">
            Desktop evidence matrix
          </h2>
          <p className="mt-1 max-w-2xl text-sm text-slate-300">
            Targets report only capabilities they can prove. Workflow providers own semantic validation; this workspace binds their evidence to desktop cells and release gates.
          </p>
        </div>
        <Button
          type="button"
          variant="secondary"
          size="sm"
          data-testid={selectors.validation.refreshTargets}
          onClick={() => void loadInventory()}
          disabled={loading}
          icon={<RefreshCw className={loading ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden="true" />}
        >
          Refresh targets
        </Button>
      </div>

      {error && (
        <div role="alert" data-testid={selectors.validation.error} className="rounded-xl border border-red-500/40 bg-red-950/30 p-4 text-sm text-red-200">
          {error}
        </div>
      )}

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <div className="rounded-2xl border border-slate-800 bg-slate-900/60 p-5">
          <div className="mb-4">
            <h3 className="font-semibold text-white">1. Select validation scope</h3>
            <p className="text-sm text-slate-400">The selection is snapshotted when the matrix is created. Empty required selections remain visible as gate blockers.</p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="text-sm text-slate-300">Scenario
              <input data-testid={selectors.validation.scenarioName} value={scenarioName} onChange={(event) => { setScenarioName(event.target.value); }} className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-white" />
            </label>
            <label className="text-sm text-slate-300">Artifact digest
              <input data-testid={selectors.validation.artifactDigest} value={artifactDigest} onChange={(event) => { setArtifactDigest(event.target.value); }} placeholder="sha256:…" className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-white" />
            </label>
          </div>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="text-sm text-slate-300">Deployment mode
              <select data-testid={selectors.validation.deploymentMode} value={deploymentMode} onChange={(event) => { setDeploymentMode(event.target.value); }} className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-white">
                <option value="release-candidate">Release candidate</option>
                <option value="bundled-private">Bundled private</option>
                <option value="tier-1-thin-client">Tier 1 thin client</option>
                <option value="tier-2-shared-provider">Tier 2 shared provider</option>
              </select>
            </label>
            <label className="text-sm text-slate-300">Release profile
              <select data-testid={selectors.validation.releaseProfile} value={releaseProfile} onChange={(event) => { setReleaseProfile(event.target.value); }} className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-white">
                <option value="desktop-standard">Desktop standard</option>
                <option value="desktop-offline">Desktop offline</option>
                <option value="desktop-update">Desktop update</option>
                <option value="desktop-recovery">Desktop recovery</option>
              </select>
            </label>
          </div>
          <div className="mt-4 rounded-xl border border-slate-700 bg-slate-950/50 p-3">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h4 className="font-medium text-white">Journey catalog</h4>
                <p className="text-xs text-slate-400">Platform journeys and provider-owned existing BAS cases share one typed selection contract.</p>
              </div>
              <Button type="button" size="sm" variant="secondary" data-testid={selectors.validation.discoverCatalog} onClick={() => { void loadCatalog(); }} disabled={catalogLoading || !scenarioName.trim()}> {catalogLoading ? "Discovering…" : "Discover provider cases"}</Button>
            </div>
            {catalogError && <p role="alert" className="mt-2 text-xs text-amber-200">{catalogError}</p>}
            <div className="mt-3 space-y-3">
              {catalogGroups.map(([category, journeys]) => <div key={category} className="space-y-2" aria-label={`${category} journeys`}><h5 className="text-xs font-semibold uppercase tracking-wide text-slate-500">{category}</h5>{journeys.map((item) => (
                <label key={item.journey_id} data-testid={selectors.validation.catalogJourney} className="flex cursor-pointer items-start gap-2 rounded-lg border border-slate-800 bg-slate-900/60 p-2 text-xs text-slate-300">
                  <input type="checkbox" checked={selectedJourneyIds.includes(item.journey_id)} onChange={(event) => { setSelectedJourneyIds((current) => event.target.checked ? [...current, item.journey_id] : current.filter((id) => id !== item.journey_id)); }} aria-label={`Select journey ${item.display_name}`} />
                  <span className="min-w-0 flex-1"><span className="flex flex-wrap items-center gap-2 font-medium text-white">{item.display_name}<StatusBadge tone={item.required ? "success" : "info"}>{item.required ? "required" : "optional"}</StatusBadge></span><span className="mt-1 block text-slate-400">{item.execution_mode ?? "unspecified"}{item.estimated_duration_seconds ? ` · ~${String(item.estimated_duration_seconds)}s` : ""}</span><span className="mt-1 block break-all text-slate-500">source: {item.source_path ?? "not reported"}</span>{item.requirements?.length ? <span className="mt-1 block text-slate-500">requirements: {item.requirements.join(", ")}</span> : null}{item.required_capabilities?.length ? <span className="mt-1 block text-slate-500">target capabilities: {item.required_capabilities.join(", ")}</span> : null}{item.safety?.mutating ? <span className="mt-1 block text-amber-200">mutating · isolation {item.safety.requires_isolation ? "required" : "not declared"}</span> : null}</span>
                </label>
              ))}</div>)}
            </div>
          </div>
          <fieldset className="mt-4">
            <legend className="text-sm font-medium text-white">Environment profiles</legend>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              {ENVIRONMENT_PROFILES.map((profile) => (
                <label key={profile.value} className="flex cursor-pointer gap-2 rounded-lg border border-slate-800 p-2 text-xs text-slate-300">
                  <input type="checkbox" checked={selectedProfiles.includes(profile.value)} onChange={(event) => { setSelectedProfiles((current) => event.target.checked ? [...current, profile.value] : current.filter((value) => value !== profile.value)); }} />
                  <span><span className="font-medium text-white">{profile.label}</span><span className="block text-slate-500">{profile.description}</span></span>
                </label>
              ))}
            </div>
          </fieldset>
          <Button type="button" className="mt-4" data-testid={selectors.validation.createMatrix} onClick={() => void createMatrix()} disabled={runLoading || !scenarioName.trim() || !artifactDigest.trim() || selectedJourneys.length === 0 || selectedTargets.length === 0 || selectedProfiles.length === 0} icon={<Check className="h-4 w-4" aria-hidden="true" />}>Create immutable matrix</Button>
          {(selectedJourneys.length === 0 || selectedTargets.length === 0 || selectedProfiles.length === 0) && <p className="mt-2 text-xs text-amber-200">Select at least one journey, target, and environment profile before creating a matrix.</p>}
        </div>

        <div className="rounded-2xl border border-slate-800 bg-slate-900/60 p-5">
          <div className="mb-4 flex items-start justify-between gap-3">
            <div><h3 className="font-semibold text-white">2. Targets and reattach</h3><p className="text-sm text-slate-400">Unavailable targets remain selectable only when explicitly included, and cannot produce a passing gate.</p></div>
            <StatusBadge tone={selectedTargets.length > 0 ? "success" : "warning"}>{selectedTargets.length} selected</StatusBadge>
          </div>
          <div className="grid gap-2">
            {availableTargets.map((target) => {
              const targetID = target.descriptor?.target_id ?? `${target.kind}-${target.os}-${target.architecture}`;
              return <label key={targetID} className="flex cursor-pointer items-start gap-3 rounded-xl border border-slate-700 bg-slate-950/50 p-3">
                <input type="checkbox" checked={selectedTargetIds.includes(targetID)} onChange={(event) => { setSelectedTargetIds((current) => event.target.checked ? [...current, targetID] : current.filter((value) => value !== targetID)); }} aria-label={`Select ${target.descriptor?.display_name ?? targetID}`} />
                <span className="min-w-0 flex-1"><span className="font-medium text-white">Include target</span><span className="mt-1 block text-xs text-slate-400">{target.kind} · {target.os} · {target.architecture} · {target.mode}</span></span>
              </label>;
            })}
          </div>
          <div className="mt-4 flex gap-2">
            <input value={runID} onChange={(event) => { setRunID(event.target.value); }} placeholder="matrix run ID" aria-label="Matrix run ID" className="min-w-0 flex-1 rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-white" />
            <Button type="button" data-testid={selectors.validation.reattachRun} onClick={() => void loadRun("")} disabled={runLoading || !runID.trim()} size="sm">Reattach</Button>
          </div>
        </div>
      </div>

      {matrixError && <div role="alert" data-testid={selectors.validation.matrixError} className="rounded-xl border border-red-500/40 bg-red-950/30 p-4 text-sm text-red-200">{matrixError}</div>}

      {run && <section aria-labelledby="validation-run-heading" className="space-y-4 rounded-2xl border border-slate-800 bg-slate-900/60 p-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div><h3 id="validation-run-heading" className="font-semibold text-white">Matrix review</h3><p className="text-xs text-slate-400">Run <code>{run.run_id}</code> · {run.selection?.scenario_name ?? scenarioName} · {run.selection?.artifact_digest ?? artifactDigest}</p></div>
          <div className="flex flex-wrap gap-2"><StatusBadge tone={runState === "completed" ? "success" : runState === "failed" || runState === "cancelled" ? "danger" : "info"}>{runState}</StatusBadge><Button type="button" size="sm" variant="secondary" onClick={() => void loadRun("")} disabled={runLoading}>Refresh run</Button><Button type="button" size="sm" data-testid={selectors.validation.startRun} onClick={startRun} disabled={runLoading || runState === "running"}>Run matrix</Button><Button type="button" size="sm" data-testid={selectors.validation.waitRun} onClick={waitForRun} disabled={runLoading || runState === "completed" || runState === "failed" || runState === "cancelled"}>Wait / reattach</Button><Button type="button" size="sm" data-testid={selectors.validation.abortRun} onClick={abortRun} disabled={runLoading || runState === "completed" || runState === "cancelled"} icon={<StopCircle className="h-4 w-4" aria-hidden="true" />}>Abort</Button><Button type="button" size="sm" variant="secondary" data-testid={selectors.validation.rerunFailed} onClick={rerunFailed} disabled={runLoading || !runState}>Rerun failed</Button><Button type="button" size="sm" variant="secondary" data-testid={selectors.validation.rerunSelected} onClick={() => void rerunSelected()} disabled={runLoading || selectedCellIds.length === 0}>Rerun selected ({selectedCellIds.length})</Button></div>
        </div>

        {swimlanes.length > 0 && <div className="rounded-xl border border-slate-700 bg-slate-950/50 p-4">
          <div className="flex flex-wrap items-center justify-between gap-2"><div><h4 className="font-medium text-white">Live target swimlanes</h4><p className="text-xs text-slate-400">Server-owned progress grouped by target; refresh or wait to reattach after leaving the page.</p></div><StatusBadge tone="info">{swimlanes.length} target(s)</StatusBadge></div>
          <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-3">{swimlanes.map(({ target, total, completed, evidence, active }) => { const progress = total > 0 ? Math.round((completed / total) * 100) : 0; return <article key={target.descriptor?.target_id ?? target.kind} className="rounded-lg border border-slate-800 bg-slate-900/70 p-3"><div className="flex items-start justify-between gap-2"><div><h5 className="font-medium text-white">{target.descriptor?.display_name ?? target.descriptor?.target_id ?? "Unknown target"}</h5><p className="text-xs text-slate-500">{target.kind} · selected target</p></div><StatusBadge tone={completed === total && total > 0 ? "success" : "info"}>{completed}/{total}</StatusBadge></div><div className="mt-3 h-2 overflow-hidden rounded-full bg-slate-800" aria-label={`${String(progress)} percent complete`}><div className="h-full rounded-full bg-blue-400 transition-all" style={{ width: `${String(progress)}%` }} /></div><p className="mt-2 text-xs text-slate-400">{active === "idle" ? "No active journey" : `Active: ${active}`} · {evidence} evidence item(s)</p></article>; })}</div>
        </div>}

        <div className="rounded-xl border border-slate-700 bg-slate-950/50 p-4">
          <div className="flex flex-wrap items-end gap-2"><label className="min-w-[16rem] flex-1 text-xs text-slate-400">Prior run for comparison<input data-testid={selectors.validation.comparePriorRun} value={priorRunID} onChange={(event) => { setPriorRunID(event.target.value); }} placeholder="prior matrix run ID" className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white" /></label><Button type="button" size="sm" variant="secondary" data-testid={selectors.validation.compareRun} onClick={() => void compareRuns()} disabled={!priorRunID.trim() || !runID.trim()}>Compare runs</Button></div>
          {comparisonError && <p role="alert" className="mt-2 text-xs text-amber-200">{comparisonError}</p>}
          {comparison && <p className="mt-2 text-xs text-slate-300">Comparison: {comparison.changed ? "changed" : "no changes reported"} · {comparison.cells?.filter((cell) => cell.changed).length ?? 0} changed cell(s).</p>}
        </div>

        <div className={`rounded-xl border p-4 ${gate?.passed ? "border-emerald-500/40 bg-emerald-950/20" : "border-amber-500/40 bg-amber-950/20"}`} aria-live="polite">
          <div className="flex flex-wrap items-center gap-2"><h4 className="font-medium text-white">Release gate</h4><StatusBadge tone={gate?.passed ? "success" : "warning"}>{gate?.passed ? "pass" : disposition(gate?.disposition).label}</StatusBadge></div>
          <p className="mt-1 text-sm text-slate-300">{gate?.reason ?? (gate ? "Required coverage is not yet complete." : "The gate will be computed after the run starts.")}</p>
          <p className="mt-2 text-xs text-slate-400">{gate?.passing_cell_count ?? 0} / {gate?.required_cell_count ?? 0} required cells passing · missing: {gate?.missing_cell_ids?.length ?? 0} · failed: {gate?.failed_cell_ids?.length ?? 0}</p>
          {(gate?.missing_cell_ids?.length || gate?.failed_cell_ids?.length) ? <p className="mt-2 text-xs text-amber-200">Blockers: {[...(gate.missing_cell_ids ?? []), ...(gate.failed_cell_ids ?? [])].join(", ")}</p> : null}
        </div>

        <div className="overflow-x-auto rounded-xl border border-slate-800"><table className="w-full min-w-[900px] text-left text-sm"><caption className="sr-only">Validation matrix cells by journey, target, and environment profile</caption><thead className="bg-slate-950 text-xs uppercase tracking-wide text-slate-400"><tr><th className="p-3">Select</th><th className="p-3">Journey</th><th className="p-3">Target</th><th className="p-3">Profile</th><th className="p-3">State</th><th className="p-3">Disposition and reason</th><th className="p-3">Evidence</th></tr></thead><tbody>{(run.cells ?? []).map((record, index) => { const cell = record.cell; const cellID = cell?.cell_id ?? `cell-${String(index)}`; const status = disposition(cell?.disposition); const target = run.selection?.targets?.find((item) => item.descriptor?.target_id === cell?.target_id); const selected = selectedCell === record; const selectedForRerun = selectedCellIds.includes(cellID); return <tr key={cellID} data-testid={selectors.validation.matrixCell} className={`border-t border-slate-800 align-top ${selected ? "bg-blue-950/20" : ""}`}><td className="p-3"><input type="checkbox" checked={selectedForRerun} onChange={(event) => { setSelectedCellIds((current) => event.target.checked ? [...current, cellID] : current.filter((id) => id !== cellID)); }} aria-label={`Select cell ${cellID} for rerun`} /></td><td className="p-3 text-white">{run.selection?.journeys?.find((item) => item.journey_id === cell?.journey_id)?.display_name ?? cell?.journey_id ?? "Unknown journey"}<span className="block text-xs text-slate-500">{cell?.required ? "required" : "optional"}</span></td><td className="p-3 text-slate-300">{target?.descriptor?.display_name ?? cell?.target_id ?? "Unknown target"}</td><td className="p-3 text-slate-300">{profileLabel(cell?.environment_profile)}</td><td className="p-3"><StatusBadge tone={record.state === "completed" && status.label === "pass" ? "success" : status.tone}>{record.state ?? "queued"}</StatusBadge></td><td className="max-w-xs p-3"><StatusBadge tone={status.tone}>{status.label}</StatusBadge><p className="mt-1 text-xs text-slate-400">{cell?.reason ?? status.explanation}</p></td><td className="p-3"><button type="button" className="text-left text-xs text-blue-300 underline" onClick={() => { setSelectedCell(record); }} aria-label={`Inspect evidence for ${cellID}`}>{cell?.evidence?.length ?? 0} item(s) · inspect</button></td></tr>; })}</tbody></table>{(run.cells ?? []).length === 0 && <p className="p-4 text-sm text-amber-200">No cells were returned. Required coverage is omitted and the release gate cannot pass.</p>}</div>

        {selectedCell && <div className="rounded-xl border border-blue-500/30 bg-blue-950/20 p-4" aria-live="polite"><div className="flex items-center justify-between gap-3"><h4 className="font-medium text-white">Evidence review</h4><button type="button" className="text-xs text-slate-400 underline" onClick={() => { setSelectedCell(null); }}>Close</button></div>{selectedCell.cell?.evidence?.length ? <div className="mt-3 grid gap-2 md:grid-cols-2">{selectedCell.cell.evidence.map((item, index) => <article key={item.evidence_id ?? index} className="rounded-lg border border-slate-700 bg-slate-950/60 p-3 text-xs"><p className="font-medium text-white">{item.media_type ?? `evidence kind ${String(item.kind ?? "unknown")}`}</p><p className="mt-1 break-all text-slate-400">{item.uri ?? "No URI reported"}</p><p className="mt-1 break-all text-slate-500">checksum: {item.sha256 ?? "not reported"}</p><p className="mt-1 text-slate-500">{item.redacted ? "redacted" : "not redacted"}</p>{item.uri?.startsWith("http") && <a className="mt-2 inline-flex items-center gap-1 text-blue-300 underline" href={item.uri} target="_blank" rel="noreferrer">Open evidence <ExternalLink className="h-3 w-3" aria-hidden="true" /></a>}</article>)}</div> : <p className="mt-2 text-sm text-amber-200">No evidence was returned for this cell. This is a visible gate blocker, not an empty success state.</p>}</div>}
      </section>}

      <div className="rounded-2xl border border-slate-800 bg-slate-900/60 p-5">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div>
            <h3 className="font-semibold text-white">Available targets</h3>
            <p className="text-sm text-slate-400">Unavailable and unsupported states remain visible and cannot pass the release gate.</p>
          </div>
          {inventory && <StatusBadge tone="info">{inventory.targets.length} discovered</StatusBadge>}
        </div>
        {!inventory && !error ? (
          <p className="text-sm text-slate-400" aria-live="polite">Discovering target capabilities…</p>
        ) : inventory?.targets.length === 0 ? (
          <p className="text-sm text-amber-200">No targets are currently registered.</p>
        ) : (
          <div className="grid gap-3 md:grid-cols-2">
            {inventory?.targets.map((target) => {
              const descriptor = target.descriptor;
              const available = Boolean(descriptor?.available);
              return (
                <article
                  key={descriptor?.target_id ?? `${target.kind}-${target.os}`}
                  data-testid={selectors.validation.targetCard}
                  className="rounded-xl border border-slate-700 bg-slate-950/50 p-4"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h4 className="font-medium text-white">{descriptor?.display_name ?? descriptor?.target_id ?? "Unnamed target"}</h4>
                      <p className="mt-1 text-xs text-slate-400">{target.os} · {target.architecture} · {target.mode}</p>
                    </div>
                    <StatusBadge tone={available ? "success" : "warning"}>{available ? "available" : "unavailable"}</StatusBadge>
                  </div>
                  <p className="mt-3 break-words text-xs text-slate-400">{descriptor?.reason ?? "No limitation reported."}</p>
                  <div className="mt-3 flex flex-wrap gap-1.5" aria-label="Target capabilities">
                    {(descriptor?.capabilities ?? []).map((capability) => (
                      <span key={capability} className="rounded bg-slate-800 px-2 py-1 text-[11px] text-slate-300">capability:{capability}</span>
                    ))}
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}
