import { useMemo, useState } from "react";
import { FilePath } from "@vrooli/react-component-library/FilePath/1.0.6";

export type IdentityState = "known" | "partial" | "stale" | "unknown";
type IdentityReport = {
  schema: "rcl-page-component-identity/v1";
  page: { source: string; kind: string; route: string | null; state: IdentityState };
  build: { stampPath?: string; digest?: string | null; pageInputMatches?: boolean; emittedChunkCount?: number; claimLimit?: string };
  components: Array<{
    component: string;
    kind?: string;
    state: IdentityState;
    assetId?: string | null;
    version?: string | null;
    sourceAssetPath?: string | null;
    builtArtifactPath?: string | null;
    reason?: string;
    ownerComponent?: string | null;
    matchingChunks?: unknown[];
    matchingSourceMaps?: unknown[];
    outputMarkers?: string[];
  }>;
  totals: Record<IdentityState, number>;
};

const identityStates = new Set<IdentityState>(["known", "partial", "stale", "unknown"]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isIdentityState(value: unknown): value is IdentityState {
  return typeof value === "string" && identityStates.has(value as IdentityState);
}

export function parsePageIdentityReport(value: unknown): IdentityReport {
  if (!isRecord(value) || value.schema !== "rcl-page-component-identity/v1") {
    throw new Error("Choose a page identity report produced by the RCL identity CLI.");
  }
  if (!isRecord(value.page) || typeof value.page.source !== "string" || typeof value.page.kind !== "string" || !isIdentityState(value.page.state)) {
    throw new Error("The report page section is incomplete or has an unsupported state.");
  }
  const totalsValue = value.totals;
  if (!isRecord(value.build) || !Array.isArray(value.components) || !isRecord(totalsValue)) {
    throw new Error("The report needs build evidence, component rows, and totals.");
  }
  const components = value.components.map((component) => {
    if (!isRecord(component) || typeof component.component !== "string" || !isIdentityState(component.state)) {
      throw new Error("A component row is missing its name or has an unsupported state.");
    }
    return {
      component: component.component,
      kind: typeof component.kind === "string" ? component.kind : undefined,
      state: component.state,
      assetId: typeof component.assetId === "string" ? component.assetId : null,
      version: typeof component.version === "string" ? component.version : null,
      sourceAssetPath: typeof component.sourceAssetPath === "string" ? component.sourceAssetPath : null,
      builtArtifactPath: typeof component.builtArtifactPath === "string" ? component.builtArtifactPath : null,
      reason: typeof component.reason === "string" ? component.reason : "No reason was supplied.",
      ownerComponent: typeof component.ownerComponent === "string" ? component.ownerComponent : null,
      matchingChunks: Array.isArray(component.matchingChunks) ? component.matchingChunks : [],
      matchingSourceMaps: Array.isArray(component.matchingSourceMaps) ? component.matchingSourceMaps : [],
      outputMarkers: Array.isArray(component.outputMarkers) ? component.outputMarkers.filter((marker): marker is string => typeof marker === "string") : [],
    };
  });
  const totals = Object.fromEntries([...identityStates].map((state) => {
    const count = totalsValue[state];
    if (typeof count !== "number" || !Number.isInteger(count) || count < 0) {
      throw new Error(`The report total for ${state} must be a non-negative integer.`);
    }
    return [state, count];
  })) as Record<IdentityState, number>;
  return {
    schema: "rcl-page-component-identity/v1",
    page: {
      source: value.page.source,
      kind: value.page.kind,
      route: typeof value.page.route === "string" ? value.page.route : null,
      state: value.page.state,
    },
    build: {
      stampPath: typeof value.build.stampPath === "string" ? value.build.stampPath : undefined,
      digest: typeof value.build.digest === "string" ? value.build.digest : null,
      pageInputMatches: typeof value.build.pageInputMatches === "boolean" ? value.build.pageInputMatches : undefined,
      emittedChunkCount: typeof value.build.emittedChunkCount === "number" ? value.build.emittedChunkCount : undefined,
      claimLimit: typeof value.build.claimLimit === "string" ? value.build.claimLimit : undefined,
    },
    components,
    totals,
  };
}

const stateClasses: Record<IdentityState, string> = {
  known: "bg-wc-success/12 text-wc-success",
  partial: "bg-wc-warning/10 text-wc-warning",
  stale: "bg-wc-error-surface/50 text-wc-error-text",
  unknown: "bg-wc-surface-input text-wc-text-muted",
};

function StateTag({ state }: { state: IdentityState }) {
  return (
    <span data-identity-state={state} className={`inline-flex min-h-6 items-center rounded-full px-2.5 py-1 text-xs font-semibold ${stateClasses[state]}`}>
      {state}
    </span>
  );
}

function scenarioName(source: string): string {
  return source.match(/^scenarios\/([^/]+)\//)?.[1]?.replace(/-/g, " ") ?? "Scenario page";
}

function quoteShell(value: string): string {
  return `'${value.replace(/'/g, "'\\''")}'`;
}

function identityCommand(report: IdentityReport): string {
  const route = report.page.route ?? "";
  return [
    "node packages/react-component-library/tooling/page-component-identity-cli.mjs .",
    quoteShell(report.page.source),
    quoteShell(report.build.stampPath ?? "<scenario-ui>/dist/index.html.freshness.json"),
    "scenarios/react-component-library/library",
    "--kind", quoteShell(report.page.kind),
    "--route", quoteShell(route),
  ].join(" ");
}

export function PageIdentityTool({ onClose }: { onClose: () => void }) {
  const [report, setReport] = useState<IdentityReport | null>(null);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState<IdentityState | "all">("all");
  const [loadedFile, setLoadedFile] = useState("");
  const visibleComponents = useMemo(
    () => report?.components.filter((component) => filter === "all" || component.state === filter) ?? [],
    [filter, report],
  );

  const loadReport = async (file: File | undefined) => {
    if (!file) return;
    setError("");
    setReport(null);
    if (file.size > 2_000_000) {
      setError("Choose a report smaller than 2 MB.");
      return;
    }
    try {
      const parsed = parsePageIdentityReport(JSON.parse(await file.text()));
      setReport(parsed);
      setLoadedFile(file.name);
      setFilter("all");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The selected file could not be read.");
      setLoadedFile("");
    }
  };

  return (
    <main className="min-h-screen overflow-y-auto bg-wc-canvas px-4 py-5 text-wc-text-primary sm:px-8 sm:py-8" data-testid="page-identity-tool">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-5">
        <header className="flex flex-wrap items-start justify-between gap-4 rounded-xl border border-wc-default bg-wc-surface-raised p-5 shadow-lg sm:p-7">
          <div className="min-w-0 flex-1">
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-wc-accent">Web Console · UX evidence</p>
            <h1 className="mt-1 text-2xl font-semibold text-wc-text-primary">Page component identity</h1>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-wc-text-secondary">
              See which page components have source and build evidence, who owns each one, and where the identity link is incomplete.
            </p>
          </div>
          <button type="button" onClick={onClose} className="min-h-11 shrink-0 rounded-lg border border-wc-default px-4 text-sm font-medium text-wc-text-primary hover:bg-wc-surface-input">
            Return to workspace
          </button>
        </header>

        <section className="rounded-xl border border-wc-default bg-wc-surface-raised p-5 shadow-lg sm:p-7" aria-labelledby="identity-report-input-heading">
          <h2 id="identity-report-input-heading" className="text-lg font-semibold">Load an identity report</h2>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-wc-text-secondary">
            Choose the JSON output from `page-component-identity-cli`. The report is read in this browser session; it is not sent to a service.
          </p>
          <label htmlFor="page-identity-report-file" className="mt-4 flex min-h-14 cursor-pointer flex-col justify-center rounded-lg border border-dashed border-wc-accent/60 bg-wc-surface-input/60 px-4 py-3 text-sm hover:border-wc-accent">
            <span className="font-semibold">Identity report (.json)</span>
            <span className="mt-0.5 text-xs text-wc-text-muted">Select one report up to 2 MB</span>
            <input
              id="page-identity-report-file"
              className="mt-3 block min-h-11 w-full text-sm text-wc-text-secondary file:me-3 file:min-h-11 file:rounded-md file:border-0 file:bg-wc-accent file:px-4 file:font-semibold file:text-wc-accent-fg"
              type="file"
              accept="application/json,.json"
              onChange={(event) => { void loadReport(event.currentTarget.files?.[0]); event.currentTarget.value = ""; }}
            />
          </label>
          {error && <p role="alert" className="mt-3 rounded-lg border border-wc-error-border bg-wc-error-surface/40 px-3 py-2 text-sm text-wc-error-text">{error}</p>}
          {report && <p role="status" className="mt-3 text-sm text-wc-success">Loaded {loadedFile} · {report.components.length} component rows</p>}
        </section>

        {report && (
          <>
            <section className="rounded-xl border border-wc-default bg-wc-surface-raised p-5 shadow-lg sm:p-7" aria-labelledby="identity-page-heading">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="text-xs font-semibold uppercase tracking-[0.1em] text-wc-text-muted">{scenarioName(report.page.source)} · {report.page.kind} page</p>
                  <h2 id="identity-page-heading" className="mt-1 break-words text-xl font-semibold">{report.page.route ?? "Route not recorded"}</h2>
                  <FilePath path={report.page.source} presentation="inline" showCopyButton testId="page-identity-page-source" copyButtonTestId="page-identity-page-source-copy" copyLabel="Copy page path" copiedLabel="Copied" />
                </div>
                <StateTag state={report.page.state} />
              </div>
              <dl className="mt-5 grid gap-3 border-t border-wc-default pt-4 text-sm sm:grid-cols-3">
                <div><dt className="text-xs font-semibold uppercase text-wc-text-muted">Build stamp</dt><dd className="mt-1 break-all text-wc-text-secondary">{report.build.stampPath ?? "Not available"}</dd></div>
                <div><dt className="text-xs font-semibold uppercase text-wc-text-muted">Page input hash</dt><dd className="mt-1 font-medium">{report.build.pageInputMatches === true ? "Matches build stamp" : report.build.pageInputMatches === false ? "Does not match" : "Not recorded"}</dd></div>
                <div><dt className="text-xs font-semibold uppercase text-wc-text-muted">Emitted chunks</dt><dd className="mt-1 font-medium">{report.build.emittedChunkCount ?? "Not recorded"}</dd></div>
              </dl>
              <p className="mt-4 rounded-lg border border-wc-warning/30 bg-wc-warning/5 px-3 py-2 text-xs leading-5 text-wc-warning">
                Source/build evidence only. This report does not prove that a page was served, deployed, or visually rendered.
              </p>
            </section>

            <section className="rounded-xl border border-wc-default bg-wc-surface-raised p-5 shadow-lg sm:p-7" aria-labelledby="identity-components-heading">
              <div className="flex flex-wrap items-end justify-between gap-3">
                <div>
                  <h2 id="identity-components-heading" className="text-lg font-semibold">Components and owners</h2>
                  <p className="mt-1 text-sm text-wc-text-secondary">Each row shows the owner chain and the strongest source/build link available.</p>
                </div>
                <label className="flex flex-col gap-1 text-xs font-semibold text-wc-text-muted">
                  Filter identity state
                  <select value={filter} onChange={(event) => setFilter(event.currentTarget.value as IdentityState | "all")} className="min-h-11 min-w-44 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm font-normal text-wc-text-primary">
                    <option value="all">All states ({report.components.length})</option>
                    {([...identityStates] as IdentityState[]).map((state) => <option key={state} value={state}>{state} ({report.totals[state]})</option>)}
                  </select>
                </label>
              </div>
              <ul className="mt-4 space-y-3" data-testid="page-identity-components">
                {visibleComponents.map((component, index) => (
                  <li key={`${component.ownerComponent ?? "page"}:${component.component}:${index}`} className="rounded-lg border border-wc-default bg-wc-surface-input/45 p-4" data-testid="page-identity-component-row">
                    <div className="flex flex-wrap items-start justify-between gap-2">
                      <div className="min-w-0">
                        <h3 className="break-words font-semibold">{component.component}</h3>
                        <p className="mt-0.5 text-xs text-wc-text-muted">{component.ownerComponent ? `Owned by ${component.ownerComponent}` : "Placed by the page"}</p>
                      </div>
                      <StateTag state={component.state} />
                    </div>
                    <div className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
                      <div>
                        <p className="text-xs font-semibold uppercase text-wc-text-muted">Source owner</p>
                        {component.sourceAssetPath ? (
                          <FilePath path={component.sourceAssetPath} presentation="inline" showCopyButton testId={`page-identity-source-${index}`} copyButtonTestId={`page-identity-source-copy-${index}`} copyLabel="Copy source path" copiedLabel="Copied" />
                        ) : <p className="mt-1 text-wc-text-secondary">No source file resolved</p>}
                      </div>
                      <div>
                        <p className="text-xs font-semibold uppercase text-wc-text-muted">Shared asset</p>
                        <p className="mt-1 break-words text-wc-text-secondary">{component.assetId ? `${component.assetId}${component.version ? ` · ${component.version}` : ""}` : component.kind ?? "Unknown"}</p>
                      </div>
                    </div>
                    <p className="mt-3 text-sm leading-5 text-wc-text-secondary">{component.reason}</p>
                    {(component.matchingSourceMaps?.length || component.outputMarkers?.length) ? (
                      <p className="mt-2 text-xs text-wc-text-muted">Linked evidence: {component.matchingSourceMaps?.length ? `${component.matchingSourceMaps.length} source map` : ""}{component.matchingSourceMaps?.length && component.outputMarkers?.length ? " · " : ""}{component.outputMarkers?.length ? `${component.outputMarkers.length} emitted marker${component.outputMarkers.length === 1 ? "" : "s"}` : ""}</p>
                    ) : null}
                  </li>
                ))}
                {visibleComponents.length === 0 && <li className="rounded-lg border border-wc-default px-4 py-6 text-center text-sm text-wc-text-muted">No components match this state.</li>}
              </ul>
              <details className="mt-4 rounded-lg border border-wc-default p-3">
                <summary className="cursor-pointer text-sm font-semibold">Repeat this identity check</summary>
                <p className="mt-2 text-sm text-wc-text-secondary">Run the same lens against another routed page, then load its report here.</p>
                <pre className="mt-3 overflow-x-auto rounded-lg bg-wc-surface-input p-3 text-xs leading-5 text-wc-text-secondary"><code>{identityCommand(report)}</code></pre>
              </details>
            </section>
          </>
        )}
      </div>
    </main>
  );
}
