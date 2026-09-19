import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

import { fetchRun, type RunRow } from "../../api/inventory";
import { ArtifactStatusPill } from "../artifacts/ArtifactStatusPill";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { TabList, TabPanel } from "../../components/ui/Tabs";
import { ROUTES } from "../../routes.generated";

const RUN_KEY = (runId: string) => ["run", runId] as const;

type TabKey = "result" | "counterexample" | "raw";

type ParsedCe = { ok: true; value: unknown } | { ok: false; error: string };

function parseCounterexample(raw: string | undefined): ParsedCe | null {
  if (!raw) return null;
  try {
    return { ok: true, value: JSON.parse(raw) };
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : String(err) };
  }
}

export function RunDetailPage() {
  const { t } = useTranslation();
  const { runId } = useParams<{ runId: string }>();

  if (!runId) {
    return (
      <section
        data-testid="run-detail-missing"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-danger"
      >
        {t("runDetail.missingId", { defaultValue: "No runId in route." })}
      </section>
    );
  }

  return <RunDetailBody runId={runId} />;
}

function RunDetailBody({ runId }: { runId: string }) {
  const { t } = useTranslation();
  const runQuery = useQuery({
    queryKey: RUN_KEY(runId),
    queryFn: () => fetchRun(runId),
  });

  if (runQuery.isLoading) {
    return (
      <section
        data-testid="run-detail-loading"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground"
      >
        {t("runDetail.loading", { defaultValue: "Loading run…" })}
      </section>
    );
  }

  if (runQuery.error || !runQuery.data) {
    return (
      <section
        data-testid="run-detail-error"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-danger"
      >
        {errorMessage(runQuery.error, t)}
        <div className="mt-3">
          <Link
            data-testid="run-detail-back"
            to={ROUTES.flowsInventory}
            className="text-sm text-app-primary underline"
          >
            {t("runDetail.back", { defaultValue: "Back to inventory" })}
          </Link>
        </div>
      </section>
    );
  }

  return <RunDetailLoaded run={runQuery.data} />;
}

function tabLabel(key: TabKey, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (key) {
    case "result":
      return t("runDetail.tabResult", { defaultValue: "Result" });
    case "counterexample":
      return t("runDetail.tabCounterexample", { defaultValue: "Counterexample" });
    case "raw":
      return t("runDetail.tabRaw", { defaultValue: "Raw output" });
  }
}

function RunDetailLoaded({ run }: { run: RunRow }) {
  const { t } = useTranslation();
  const parsed = useMemo(
    () => parseCounterexample(run.counterexample),
    [run.counterexample],
  );
  const [tab, setTab] = useState<TabKey>("result");

  const statusClass =
    run.status === "passed"
      ? "text-app-success"
      : run.status === "failed"
        ? "text-app-danger"
        : "text-app-warning";

  return (
    <section
      data-testid="run-detail-page"
      aria-label={t("runDetail.title", { defaultValue: "Run detail" })}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <header className="flex flex-wrap items-baseline justify-between gap-3">
        <div>
          <h2 className="text-sm font-medium text-app-muted-foreground">
            {t("runDetail.title", { defaultValue: "Run detail" })}
          </h2>
          <p
            data-testid="run-detail-id"
            className="mt-1 font-mono text-base text-app-foreground"
          >
            {run.id}
          </p>
          <p className="mt-1 text-xs text-app-muted-foreground">
            <Link
              data-testid="run-detail-flow-link"
              to={ROUTES.flowDetail(encodeURIComponent(run.flowId))}
              className="font-mono text-app-primary underline"
            >
              {run.flowId}
            </Link>
            {" · "}
            <span data-testid="run-detail-mode">{run.mode}</span>
            {" · "}
            <span data-testid="run-detail-status" className={statusClass}>
              {run.status}
            </span>
            {" · "}
            <span data-testid="run-detail-duration">
              {t("runDetail.durationMs", {
                defaultValue: "{{ms}} ms",
                ms: run.durationMs,
              })}
            </span>
          </p>
        </div>
        <Link
          data-testid="run-detail-back"
          to={ROUTES.flowsInventory}
          className="text-sm text-app-primary underline"
        >
          {t("runDetail.back", { defaultValue: "Back to inventory" })}
        </Link>
      </header>

      <TabList
        idPrefix="run-detail"
        value={tab}
        onChange={setTab}
        aria-label={t("runDetail.tabsAria", { defaultValue: "Run detail sections" })}
        className="mt-4"
        items={[
          { value: "result", label: tabLabel("result", t) },
          { value: "counterexample", label: tabLabel("counterexample", t) },
          { value: "raw", label: tabLabel("raw", t) },
        ]}
      />

      <div className="mt-4">
        <TabPanel idPrefix="run-detail" value="result" active={tab}>
          <ResultTab run={run} />
        </TabPanel>
        <TabPanel idPrefix="run-detail" value="counterexample" active={tab}>
          <CounterexampleTab parsed={parsed} />
        </TabPanel>
        <TabPanel idPrefix="run-detail" value="raw" active={tab}>
          <RawOutputTab output={run.output ?? ""} />
        </TabPanel>
      </div>
    </section>
  );
}

function ResultTab({ run }: { run: RunRow }) {
  const { t } = useTranslation();
  return (
    <dl
      data-testid="run-detail-result"
      className="grid gap-x-4 gap-y-2 text-sm text-app-foreground sm:grid-cols-2"
    >
      <Row label={t("runDetail.colStarted", { defaultValue: "Started" })}>
        <span data-testid="run-detail-started">
          {new Date(run.startedAt).toLocaleString()}
        </span>
      </Row>
      <Row label={t("runDetail.colFinished", { defaultValue: "Finished" })}>
        <span data-testid="run-detail-finished">
          {new Date(run.finishedAt).toLocaleString()}
        </span>
      </Row>
      <Row label={t("runDetail.colRoot", { defaultValue: "Root" })}>
        <span data-testid="run-detail-root" className="font-mono text-xs">
          {run.root}
        </span>
      </Row>
      <Row label={t("runDetail.colFlowPath", { defaultValue: "Flow path" })}>
        <span data-testid="run-detail-flow-path" className="font-mono text-xs">
          {run.flowPath}
        </span>
      </Row>
      {run.failureReason === "missing_artifacts" || run.failureReason === "stale_artifacts" ? (
        <Row label={t("runDetail.colArtifacts", { defaultValue: "Artifacts" })}>
          <span className="flex flex-col gap-1">
            <ArtifactStatusPill
              status="needs_generate"
              testId="run-detail-needs-generate"
            />
            {run.missingArtifacts && run.missingArtifacts.length > 0 && (
              <ul
                data-testid="run-detail-missing-artifacts"
                className="mt-1 list-disc pl-5 font-mono text-xs text-app-muted-foreground"
              >
                {run.missingArtifacts.map((path) => (
                  <li key={path}>{path}</li>
                ))}
              </ul>
            )}
          </span>
        </Row>
      ) : null}
      {run.errorMessage && (
        <Row label={t("runDetail.colError", { defaultValue: "Error" })}>
          <span data-testid="run-detail-error-message" className="text-app-danger">
            {run.errorMessage}
          </span>
        </Row>
      )}
    </dl>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col">
      <dt className="text-xs uppercase text-app-muted-foreground">{label}</dt>
      <dd className="text-app-foreground">{children}</dd>
    </div>
  );
}

function CounterexampleTab({ parsed }: { parsed: ParsedCe | null }) {
  const { t } = useTranslation();

  if (parsed === null) {
    return (
      <p
        data-testid="run-detail-no-counterexample"
        className="text-sm text-app-muted-foreground"
      >
        {t("runDetail.noCounterexample", {
          defaultValue: "No counterexample for this run.",
        })}
      </p>
    );
  }

  return (
    <details
      data-testid="run-detail-counterexample"
      open
      className="rounded-panel border border-app-border bg-app-surface-muted p-3"
    >
      <summary
        data-testid="run-detail-counterexample-toggle"
        className="cursor-pointer text-sm text-app-foreground"
      >
        {t("runDetail.counterexampleHeader", { defaultValue: "Counterexample" })}
      </summary>
      <div className="mt-3">
        {parsed.ok ? (
          <JsonNode value={parsed.value} depth={0} pathKey="root" />
        ) : (
          <p
            data-testid="run-detail-counterexample-parse-error"
            className="text-sm text-app-danger"
          >
            {t("runDetail.counterexampleParseError", {
              defaultValue: `Could not parse counterexample JSON: ${parsed.error}`,
            })}
          </p>
        )}
      </div>
    </details>
  );
}

function RawOutputTab({ output }: { output: string }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  if (!output) {
    return (
      <p
        data-testid="run-detail-raw-empty"
        className="text-sm text-app-muted-foreground"
      >
        {t("runDetail.rawEmpty", { defaultValue: "No raw output captured." })}
      </p>
    );
  }

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(output);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div data-testid="run-detail-raw" className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs uppercase text-app-muted-foreground">
          {t("runDetail.rawLabel", { defaultValue: "Quint output" })}
        </span>
        <button
          type="button"
          data-testid="run-detail-raw-copy"
          onClick={() => void onCopy()}
          className="rounded-control border border-app-border bg-app-surface px-2 py-1 text-xs text-app-foreground hover:bg-app-surface-muted"
        >
          {copied
            ? t("runDetail.rawCopied", { defaultValue: "Copied" })
            : t("runDetail.rawCopy", { defaultValue: "Copy" })}
        </button>
      </div>
      <pre
        data-testid="run-detail-raw-pre"
        className="max-h-96 overflow-auto rounded-panel border border-app-border bg-app-surface-muted p-3 font-mono text-xs text-app-foreground"
      >
        {output}
      </pre>
    </div>
  );
}

const LAZY_DEPTH = 2;

function JsonNode({
  value,
  depth,
  pathKey,
}: {
  value: unknown;
  depth: number;
  pathKey: string;
}) {
  const { t } = useTranslation();
  if (value === null) {
    return (
      <span className="text-app-muted-foreground">
        {t("runDetail.jsonNull", { defaultValue: "null" })}
      </span>
    );
  }
  if (typeof value !== "object") {
    return <span className="text-app-success">{JSON.stringify(value)}</span>;
  }

  const isArray = Array.isArray(value);
  const entries = isArray
    ? (value as unknown[]).map((v, i) => [String(i), v] as const)
    : Object.entries(value as Record<string, unknown>);

  if (entries.length === 0) {
    return <span className="text-app-muted-foreground">{isArray ? "[]" : "{}"}</span>;
  }

  const defaultOpen = depth < LAZY_DEPTH;

  return (
    <details
      open={defaultOpen}
      data-testid={`run-detail-json-${pathKey}`}
      className="ml-2 border-l border-app-border pl-2"
    >
      <summary className="cursor-pointer text-xs text-app-muted-foreground">
        {isArray ? `Array(${entries.length})` : `Object{${entries.length}}`}
      </summary>
      <ul className="ml-2 mt-1 space-y-1 text-xs">
        {entries.map(([k, v]) => (
          <li key={k}>
            <span className="text-app-muted-foreground">{isArray ? `[${k}]` : `${k}: `}</span>
            <JsonNode value={v} depth={depth + 1} pathKey={`${pathKey}.${k}`} />
          </li>
        ))}
      </ul>
    </details>
  );
}
