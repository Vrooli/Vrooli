/** @vrooliComponentSource data-display.data-table */
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Alert } from "@vrooli/react-component-library/Alert/1";
import { type CaptureCell } from "@vrooli/react-component-library/CaptureGrid/1";
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";
import { BulkActionBar } from "@vrooli/react-component-library/BulkActionBar/1";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1";
import { DiffViewer } from "@vrooli/react-component-library/DiffViewer/1";
import { FindingList, type Finding } from "@vrooli/react-component-library/FindingList/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { UndoableDestructiveAction } from "@vrooli/react-component-library/UndoableDestructiveAction/1";
import { VerdictSummary } from "@vrooli/react-component-library/VerdictSummary/1";
import { VirtualList } from "@vrooli/react-component-library/VirtualList/1";
import {
  VersionRow,
  type VersionAdopter,
  type VersionDiffSummary,
} from "@vrooli/react-component-library/VersionRow/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { versionsClient, type DiffVersionsResponse, type Version } from "../../api/versions";
import { versionLifecycleClient } from "../../api/versionLedger";
import type { Adoption } from "../../api/adoptions";
import type { ComponentTestArtifact, ComponentTestReport } from "../../api/componentTests";
import { listVersionHistory, type VersionHistoryRow } from "../../api/versionHistory";
import { errorMessage } from "../../lib/errorMessage";

interface VersionsCardProps {
  componentId: string;
  selectedVersion?: string;
  onSelectVersion?: (version: string | undefined) => void;
  /** Lets the enclosing code workspace own comparison rendering. */
  onCompare?: (diff: DiffVersionsResponse) => void;
}

const EMPTY_VERSIONS: Version[] = [];

function artifactIsImage(artifact: ComponentTestArtifact) {
  return ["bas-screenshot", "screenshot", "bas-story-sheet"].includes(artifact.kind);
}

function captureCells(report: ComponentTestReport | undefined, version: string): CaptureCell[] {
  return (report?.artifacts ?? []).filter(artifactIsImage).map((artifact, index) => {
    const label = artifact.storyId || artifact.label || `capture-${index + 1}`;
    const result = report?.results.find(
      (candidate) => candidate.version === version && candidate.subject === label,
    );
    return {
      id: `${version}-${artifact.kind}-${artifact.storyId || index}`,
      viewport: label.replace(/[:_-]?(screenshot|story-sheet)/gi, "").trim() || "default",
      theme: /dark/i.test(label) ? "dark" : "light",
      status: result?.verdict === "failed" ? "stale" : "pass",
      captureRef: artifact.reference,
    } satisfies CaptureCell;
  });
}

function reportFindings(report: ComponentTestReport | undefined, version: string): Finding[] {
  return (report?.results ?? [])
    .filter((result) => result.verdict !== "passed")
    .map((result, index) => ({
      id: `${version}-${result.stage}-${index}`,
      assetId: result.subject || result.stage,
      severity: result.verdict === "failed" ? "error" : "warning",
      message: result.message || `${result.stage} is ${result.verdict}.`,
      remediation: result.remediation,
    }));
}

function adopterRows(adopters: Adoption[]): VersionAdopter[] {
  return adopters.map((adopter) => ({
    scenario: adopter.scenario || "unknown scenario",
    adoptedVersion: adopter.adoptedVersion,
    forkStatus: adopter.forkStatus || undefined,
    statusDetail: adopter.statusDetail || undefined,
  }));
}

/**
 * VersionsCard renders the version-history list for one component and
 * a built-in diff viewer that picks two versions (or one version and
 * an `adoption:<id>` ref) and renders aligned left/right rows.
 *
 * Surface for req 11 (VR-001..003).
 */
export function VersionsCard({
  componentId,
  selectedVersion,
  onSelectVersion,
  onCompare,
}: VersionsCardProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [retireActionsVisible, setRetireActionsVisible] = useState(false);
  const [expandedVersion, setExpandedVersion] = useState<string>();
  const [materializeErrors, setMaterializeErrors] = useState<Record<string, string>>({});

  const historyQuery = useQuery<VersionHistoryRow[]>({
    queryKey: ["version-history", componentId],
    queryFn: () => listVersionHistory(componentId),
  });

  const diffMutation = useMutation({
    mutationFn: () => versionsClient.diffVersions({ componentId, from, to }),
    onSuccess: (diff) => onCompare?.(diff),
  });

  const historyRows = historyQuery.data ?? [];
  const versions: Version[] = historyRows.length
    ? historyRows.map((row) => row.version)
    : EMPTY_VERSIONS;
  const diff = diffMutation.data;
  const expandedIndex = versions.findIndex((version) => version.version === expandedVersion);
  const previousVersion = expandedIndex >= 0 ? versions[expandedIndex + 1]?.version : undefined;
  const expandedDiffQuery = useQuery({
    queryKey: ["version-history-diff", componentId, previousVersion, expandedVersion],
    queryFn: () =>
      versionsClient.diffVersions({
        componentId,
        from: previousVersion ?? "",
        to: expandedVersion ?? "",
      }),
    enabled: Boolean(previousVersion && expandedVersion),
  });
  const retireCandidatesQuery = useQuery({
    queryKey: ["retire-candidates", componentId],
    queryFn: () => versionLifecycleClient.listRetireCandidates({ componentId }),
    enabled: Boolean(componentId),
  });
  const retireMutation = useMutation({
    mutationFn: (input: { componentId: string; version: string }) =>
      versionLifecycleClient.retireVersion({ ...input, confirm: true }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["version-history", componentId] }),
        queryClient.invalidateQueries({ queryKey: ["retire-candidates", componentId] }),
      ]);
    },
  });
  const materializeMutation = useMutation({
    mutationFn: (version: string) =>
      versionLifecycleClient.materializeVersion({ componentId, version }),
    onSuccess: async (_, version) => {
      setMaterializeErrors((errors) => {
        const next = { ...errors };
        delete next[version];
        return next;
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["version-history", componentId] }),
        queryClient.invalidateQueries({ queryKey: ["retire-candidates", componentId] }),
      ]);
    },
    onError: (error, version) => {
      setMaterializeErrors((errors) => ({ ...errors, [version]: errorMessage(error, t) }));
    },
  });

  const versionOptions = useMemo(
    () => versions.map((v) => v.version).filter((v) => v.length > 0),
    [versions],
  );
  const historyByVersion = useMemo(
    () => new Map(historyRows.map((row) => [row.version.version, row] as const)),
    [historyRows],
  );
  const columns = useMemo<DataTableColumn<Version>[]>(
    () => [
      {
        id: "version",
        header: "Version history",
        accessor: (version) => {
          const history = historyByVersion.get(version.version);
          const ledger = history?.ledger;
          const report = history?.testReport;
          const diffSummary: VersionDiffSummary | undefined =
            expandedVersion === version.version && expandedDiffQuery.data
              ? {
                  fromVersion: previousVersion ?? "",
                  additions: expandedDiffQuery.data.additions,
                  removals: expandedDiffQuery.data.removals,
                  note: version.changelogMd || undefined,
                }
              : undefined;
          const thumbnailArtifact = (report?.artifacts ?? []).find(
            (candidate) =>
              artifactIsImage(candidate) && /^(https?:\/\/|\/embedded\/)/.test(candidate.reference),
          );
          const presence = version.presence || "materialized";
          return (
            <div className="space-y-space-2xs">
              <div
                data-testid={`${selectors.versions.presenceBadge}-${version.version}`}
                aria-label={`${version.version} presence: ${presence}`}
                className="flex items-center gap-space-2xs text-xs text-app-muted-foreground"
              >
                <span aria-hidden="true">{presence === "evicted" ? "⤓" : "●"}</span>
                <span>
                  {presence === "evicted"
                    ? "Evicted — bytes remain in the ledger"
                    : "Materialized — bytes are on disk"}
                </span>
                {presence === "evicted" ? (
                  <button
                    type="button"
                    data-testid={`${selectors.versions.materializeButton}-${version.version}`}
                    className="rounded-control border border-app-border px-space-xs py-space-2xs text-xs text-app-foreground"
                    disabled={materializeMutation.isPending}
                    onClick={() => materializeMutation.mutate(version.version)}
                  >
                    {materializeMutation.isPending
                      ? "Materializing…"
                      : `Materialize v${version.version}`}
                  </button>
                ) : null}
              </div>
              {materializeErrors[version.version] ? (
                <p role="alert" className="text-xs text-app-danger">
                  Could not materialize v{version.version}: {materializeErrors[version.version]}
                </p>
              ) : null}
              <VersionRow
                version={version.version}
                sha={version.contentSha256}
                status={version.status}
                createdAt={
                  version.createdAt
                    ? new Date(Number(version.createdAt.seconds) * 1000).toISOString()
                    : undefined
                }
                releasedAt={
                  version.releasedAt
                    ? new Date(Number(version.releasedAt.seconds) * 1000).toISOString()
                    : undefined
                }
                sourcePath={version.sourcePath}
                requiredTokens={version.requiredTokens}
                lifecycleState={ledger?.lifecycleState}
                gatePassCount={ledger?.gatePassCount}
                gateFailCount={ledger?.gateFailCount}
                testRuns={ledger?.testRuns}
                testPassRate={ledger?.testPassRate}
                fileCount={ledger?.fileCount}
                linesOfCode={ledger?.linesOfCode}
                dependencyCount={ledger?.dependencyCount}
                adoptionCurrent={ledger?.adoptionCurrent}
                adoptionPeak={ledger?.adoptionPeak}
                trend={ledger ? [ledger.adoptionCurrent, ledger.adoptionPeak] : undefined}
                captures={captureCells(report, version.version)}
                thumbnail={
                  thumbnailArtifact?.reference
                    ? {
                        src: thumbnailArtifact.reference,
                        alt: `${version.version} captured state`,
                      }
                    : undefined
                }
                findings={reportFindings(report, version.version)}
                adopters={adopterRows(history?.adopters ?? [])}
                previousVersion={previousVersion}
                diffSummary={diffSummary}
                selected={selectedVersion === version.version}
                onSelect={() => onSelectVersion?.(version.version)}
                onExpandedChange={(expanded) =>
                  setExpandedVersion(expanded ? version.version : undefined)
                }
              />
            </div>
          );
        },
        searchValue: (version) => `${version.version} ${version.status} ${version.contentSha256}`,
      },
    ],
    [
      expandedDiffQuery.data,
      expandedVersion,
      historyByVersion,
      onSelectVersion,
      previousVersion,
      selectedVersion,
    ],
  );

  const before = diff?.rows.map((row) => row.left?.text ?? "").join("\n") ?? "";
  const after = diff?.rows.map((row) => row.right?.text ?? "").join("\n") ?? "";
  const ledgerRows = historyRows.flatMap((row) => (row.ledger ? [row.ledger] : []));
  const retireCandidates = retireCandidatesQuery.data?.candidates ?? [];
  const gatePassCount = ledgerRows.reduce((sum, row) => sum + row.gatePassCount, 0);
  const gateFailCount = ledgerRows.reduce((sum, row) => sum + row.gateFailCount, 0);
  const testRuns = ledgerRows.reduce((sum, row) => sum + row.testRuns, 0);
  const measuredTests = ledgerRows.filter((row) => row.testRuns > 0);
  const testPassRate = measuredTests.length
    ? measuredTests.reduce((sum, row) => sum + row.testPassRate, 0) / measuredTests.length
    : 0;
  const tokenFindings: Finding[] = versions.flatMap((version) =>
    version.requiredTokens.map((token) => ({
      id: `${version.version}-${token}`,
      assetId: version.version,
      severity: "warning",
      message: `Requires shared token ${token}.`,
      remediation: "Provide the token in the active library theme before adoption.",
    })),
  );

  return (
    <section
      data-testid={selectors.versions.card}
      aria-label={t(strings.versions.title)}
      className="mt-space-sm rounded-xl border border-app-border bg-app-surface p-space-sm backdrop-blur-sm"
    >
      <header className="flex items-center justify-between gap-space-xs">
        <h2 className="text-sm font-medium text-app-foreground">{t(strings.versions.title)}</h2>
        {versions.length > 0 && (
          <span className="text-xs text-app-muted-foreground">
            {t(strings.versions.summary, { count: versions.length })}
          </span>
        )}
      </header>

      {historyQuery.isLoading && (
        <div data-testid={selectors.versions.loading} className="mt-space-xs">
          <Skeleton label={t(strings.versions.loading)} style={{ minBlockSize: "6rem" }} />
          <p className="sr-only">{t(strings.versions.loading)}</p>
        </div>
      )}
      {historyQuery.error && (
        <p data-testid={selectors.versions.error} className="mt-space-xs text-app-danger">
          {errorMessage(historyQuery.error, t)}
        </p>
      )}
      {!historyQuery.isLoading && versions.length === 0 && (
        <p data-testid={selectors.versions.empty}>{t(strings.versions.empty)}</p>
      )}

      {versions.length > 0 && (
        <>
          {ledgerRows.length > 0 && (
            <div className="mt-space-sm grid gap-space-xs md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
              <VerdictSummary
                pass={gatePassCount}
                fail={gateFailCount}
                unmeasured={ledgerRows.length - measuredTests.length}
              />
              <BoundedMeter
                label="Test health"
                value={testPassRate * 100}
                max={100}
                valueText={
                  measuredTests.length ? `${Math.round(testPassRate * 100)}% pass` : "unknown"
                }
                status={measuredTests.length ? `${testRuns} test runs` : "No test runs recorded"}
                tone={measuredTests.length && testPassRate >= 0.9 ? "success" : "warning"}
                testId="versions-test-health"
              />
            </div>
          )}
          {tokenFindings.length > 0 && (
            <div className="mt-space-sm">
              <FindingList findings={tokenFindings} />
            </div>
          )}
          <button
            type="button"
            className="mt-space-xs rounded-control border border-app-border px-space-xs py-space-2xs text-xs"
            onClick={() => onSelectVersion?.(undefined)}
          >
            {t(strings.versions.currentSource)}
          </button>
          <div data-testid={selectors.versions.list} className="mt-space-xs">
            {versions.length > 50 ? (
              <VirtualList
                items={versions}
                getItemKey={(version) => version.id}
                label={t(strings.versions.title)}
                title={t(strings.versions.title)}
                description={`${versions.length} versions`}
                renderItem={(version) => columns[0]?.accessor(version) ?? null}
              />
            ) : (
              <DataTable
                rows={versions}
                columns={columns}
                getRowKey={(version) => version.id}
                caption={t(strings.versions.title)}
                tableTestId={selectors.versions.table}
                hideQueryControls
                hideDensityControl
                emptyMessage={t(strings.versions.empty)}
              />
            )}
          </div>
        </>
      )}

      {retireCandidates.length > 0 && (
        <div className="mt-space-sm space-y-space-xs" data-testid="versions-retire-candidates">
          <Alert
            tone="warning"
            title="Versions are safe to retire"
            description={
              <span>
                These versions are not latest or draft and have no recorded adopters, dependency
                pins, or source references:{" "}
                {retireCandidates.map((candidate) => candidate.version).join(", ")}.
              </span>
            }
            actions={
              <button
                type="button"
                className="h-control-compact rounded-control border border-app-border px-space-xs text-xs"
                onClick={() => setRetireActionsVisible((visible) => !visible)}
              >
                {retireActionsVisible ? "Hide retire actions" : "Review retire actions"}
              </button>
            }
          />
          <BulkActionBar
            selectedCount={retireCandidates.length}
            totalCount={retireCandidates.length}
            actionLabel="Review retire actions"
            selectionLabel={`${retireCandidates.length} safe retire candidate${retireCandidates.length === 1 ? "" : "s"}`}
            scopeLabel="Each action keeps an undo path after the server confirms retirement."
            onAction={() => setRetireActionsVisible(true)}
            onClear={() => setRetireActionsVisible(false)}
          />
          {retireActionsVisible &&
            retireCandidates.map((candidate) => (
              <UndoableDestructiveAction
                key={`${candidate.componentId}-${candidate.version}`}
                itemLabel={`Version ${candidate.version}`}
                description={`Retire ${candidate.version} after confirming it has no surviving references.`}
                deleteLabel={`Retire ${candidate.version}`}
                onDelete={async () => {
                  await retireMutation.mutateAsync({
                    componentId: candidate.componentId,
                    version: candidate.version,
                  });
                }}
                onUndo={async () => {
                  await versionLifecycleClient.archiveVersion({
                    componentId: candidate.componentId,
                    version: candidate.version,
                    confirm: true,
                  });
                }}
              />
            ))}
        </div>
      )}

      <div
        data-testid={selectors.versions.diff.card}
        className="mt-space-sm rounded-lg border border-app-border bg-app-surface-muted p-space-xs"
      >
        <h3 className="text-xs font-medium text-app-muted-foreground">
          {t(strings.versions.diff.title)}
        </h3>
        <div className="mt-space-2xs flex flex-wrap items-center gap-space-2xs text-xs text-app-muted-foreground">
          <label className="flex items-center gap-space-2xs">
            <span>{t(strings.versions.diff.fromLabel)}</span>
            <Select
              data-testid={selectors.versions.diff.fromSelect}
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="w-field-short"
              options={versionOptions.map((version) => ({ value: version, label: version }))}
              placeholder="—"
            />
          </label>
          <label className="flex items-center gap-space-2xs">
            <span>{t(strings.versions.diff.toLabel)}</span>
            <Select
              data-testid={selectors.versions.diff.toSelect}
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="w-field-short"
              options={versionOptions.map((version) => ({ value: version, label: version }))}
              placeholder="—"
            />
          </label>
          <button
            data-testid={selectors.versions.diff.runButton}
            type="button"
            onClick={() => diffMutation.mutate()}
            disabled={!from || !to || diffMutation.isPending}
            className="h-control-compact px-space-xs text-xs"
          >
            {diffMutation.isPending
              ? t(strings.versions.diff.running)
              : t(strings.versions.diff.runAction)}
          </button>
        </div>

        {diffMutation.error && (
          <p
            data-testid={selectors.versions.diff.error}
            className="mt-space-2xs text-xs text-app-danger"
          >
            {errorMessage(diffMutation.error, t)}
          </p>
        )}

        {!diff && !diffMutation.isPending && !diffMutation.error && (
          <p
            data-testid={selectors.versions.diff.empty}
            className="mt-space-2xs text-xs text-app-muted-foreground"
          >
            {t(strings.versions.diff.empty)}
          </p>
        )}

        {diff && !onCompare && (
          <>
            <p
              data-testid={selectors.versions.diff.summary}
              className="mt-space-2xs text-xs text-app-muted-foreground"
            >
              {t(strings.versions.diff.summary, {
                from: diff.fromLabel,
                to: diff.toLabel,
                additions: diff.additions,
                removals: diff.removals,
                rows: diff.rows.length,
              })}
            </p>
            <div data-testid={selectors.versions.diff.table}>
              <DiffViewer before={before} after={after} />
            </div>
          </>
        )}
      </div>
    </section>
  );
}
