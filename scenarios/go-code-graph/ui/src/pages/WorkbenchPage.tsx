import * as React from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { Tabs, TabsList, TabsTrigger, TabsPanel } from "../components/ui/tabs";
import { useExtract } from "../features/explorer/controllers/useExtract";
import { summarizeGraph } from "../features/explorer/lib/graphAdapter";
import { ExplorerTab } from "../features/explorer/ExplorerTab";
import { WarningsTab } from "../features/warnings/WarningsTab";
import { RewriteTab } from "../features/rewrite/RewriteTab";
import { FixturesTab } from "../features/fixtures/FixturesTab";

const TAB_GRAPH = "graph";
const TAB_WARNINGS = "warnings";
const TAB_REWRITE = "rewrite";
const TAB_FIXTURES = "fixtures";

/** Join an optional base directory with the target into a single module path. */
function resolveModulePath(target: string, projectDir: string): string {
  const t = target.trim();
  const base = projectDir.trim().replace(/\/+$/, "");
  if (base.length === 0) return t;
  if (t.length === 0) return base;
  return `${base}/${t}`;
}

interface StatProps {
  testId: string;
  label: string;
  value: React.ReactNode;
}

function Stat({ testId, label, value }: StatProps) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-3">
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <p data-testid={testId} className="mt-1 text-xl font-semibold">
        {value}
      </p>
    </div>
  );
}

/**
 * Single-page operational workbench for go-code-graph. The extract bar + stats
 * header persist; Graph / Warnings / Rewrite / Fixtures live as tabs.
 */
export function WorkbenchPage() {
  const { t } = useTranslation();

  const [target, setTarget] = React.useState("");
  const [projectDir, setProjectDir] = React.useState("");
  const [includeVendor, setIncludeVendor] = React.useState(false);
  const [submittedPath, setSubmittedPath] = React.useState<string | null>(null);
  const [activeTab, setActiveTab] = React.useState<string>(TAB_GRAPH);

  const params = submittedPath
    ? { modulePath: submittedPath, includeVendor }
    : null;
  const extract = useExtract(params);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const resolved = resolveModulePath(target, projectDir);
    if (resolved.length === 0) return;
    setSubmittedPath(resolved);
  };

  // Toggling vendor re-runs extraction for the same target (params key folds in
  // the flag, so React Query refetches automatically).
  const handleToggleVendor = (next: boolean) => {
    setIncludeVendor(next);
  };

  const graph = extract.data?.graph;
  const summary = React.useMemo(() => summarizeGraph(graph), [graph]);
  const warnings = extract.data?.warnings ?? [];

  return (
    <section
      data-testid={selectors.pages.workbench}
      aria-labelledby="workbench-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="workbench-heading" className="text-2xl font-semibold">
          {t(strings.pages.workbench.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.workbench.description)}</p>
      </header>

      {/* Extract bar */}
      <form
        data-testid={selectors.workbench.extractBar.root}
        onSubmit={handleSubmit}
        className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4 backdrop-blur-sm md:flex-row md:items-end"
      >
        <label className="flex flex-1 flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.workbench.extract.targetLabel)}</span>
          <Input
            data-testid={selectors.workbench.extractBar.target}
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder={t(strings.workbench.extract.targetPlaceholder)}
          />
        </label>
        <label className="flex flex-1 flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.workbench.extract.projectDirLabel)}</span>
          <Input
            data-testid={selectors.workbench.extractBar.projectDir}
            value={projectDir}
            onChange={(e) => setProjectDir(e.target.value)}
            placeholder={t(strings.workbench.extract.projectDirPlaceholder)}
          />
        </label>
        <label className="flex items-center gap-2 text-sm md:pb-3">
          <input
            type="checkbox"
            data-testid={selectors.workbench.extractBar.includeVendor}
            checked={includeVendor}
            onChange={(e) => setIncludeVendor(e.target.checked)}
            className="h-4 w-4 rounded border-app-border"
          />
          <span>{t(strings.workbench.extract.includeVendor)}</span>
        </label>
        <Button
          type="submit"
          data-testid={selectors.workbench.extractBar.submit}
          disabled={resolveModulePath(target, projectDir).length === 0 || extract.isFetching}
        >
          {extract.isFetching
            ? t(strings.workbench.extract.submitting)
            : t(strings.workbench.extract.submit)}
        </Button>
      </form>

      {/* Status / stats / tabs */}
      {submittedPath === null ? (
        <div data-testid={selectors.workbench.status.empty}>
          <EmptyState
            title={t(strings.workbench.status.idleTitle)}
            description={t(strings.workbench.status.idleDescription)}
          />
        </div>
      ) : extract.isPending ? (
        <div data-testid={selectors.workbench.status.loading}>
          <LoadingState label={t(strings.shared.loading.label)} />
        </div>
      ) : extract.isError ? (
        <div data-testid={selectors.workbench.status.error}>
          <ErrorState
            title={t(strings.workbench.status.errorTitle)}
            message={extract.error instanceof Error ? extract.error.message : String(extract.error)}
            retryLabel={t(strings.shared.error.retry)}
            onRetry={() => void extract.refetch()}
          />
        </div>
      ) : (
        <>
          <div
            data-testid={selectors.workbench.stats.root}
            className="grid grid-cols-2 gap-3 md:grid-cols-5"
          >
            <Stat testId={selectors.workbench.stats.files} label={t(strings.workbench.stats.files)} value={summary.files} />
            <Stat testId={selectors.workbench.stats.packages} label={t(strings.workbench.stats.packages)} value={summary.packages} />
            <Stat testId={selectors.workbench.stats.symbols} label={t(strings.workbench.stats.symbols)} value={summary.symbols} />
            <Stat testId={selectors.workbench.stats.imports} label={t(strings.workbench.stats.imports)} value={summary.imports} />
            <Stat testId={selectors.workbench.stats.warnings} label={t(strings.workbench.stats.warnings)} value={warnings.length} />
          </div>
          <p className="text-xs text-app-muted-foreground">
            {t(strings.workbench.stats.durationLabel, { count: Number(extract.data.extractionMs) })}
            {" · "}
            <span data-testid={selectors.workbench.stats.hash} className="font-mono">
              {t(strings.workbench.stats.hash)} {extract.data.graphHash.slice(0, 12)}…
            </span>
          </p>

          <Tabs
            value={activeTab}
            onValueChange={setActiveTab}
            ariaLabel={t(strings.workbench.tabs.label)}
          >
            <TabsList>
              <TabsTrigger value={TAB_GRAPH}>{t(strings.workbench.tabs.graph)}</TabsTrigger>
              <TabsTrigger value={TAB_WARNINGS}>{t(strings.workbench.tabs.warnings)}</TabsTrigger>
              <TabsTrigger value={TAB_REWRITE}>{t(strings.workbench.tabs.rewrite)}</TabsTrigger>
              <TabsTrigger value={TAB_FIXTURES}>{t(strings.workbench.tabs.fixtures)}</TabsTrigger>
            </TabsList>

            <TabsPanel value={TAB_GRAPH}>
              <ExplorerTab graph={graph} target={submittedPath} />
            </TabsPanel>
            <TabsPanel value={TAB_WARNINGS}>
              <WarningsTab
                warnings={warnings}
                includeVendor={includeVendor}
                onToggleVendor={handleToggleVendor}
              />
            </TabsPanel>
            <TabsPanel value={TAB_REWRITE}>
              <RewriteTab modulePath={submittedPath} />
            </TabsPanel>
            <TabsPanel value={TAB_FIXTURES}>
              <FixturesTab />
            </TabsPanel>
          </Tabs>
        </>
      )}
    </section>
  );
}
