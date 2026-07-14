import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { Select } from "../../components/ui/select";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  versionsClient,
  DiffOp,
  type DiffRow,
  type Version,
} from "../../api/versions";
import { errorMessage } from "../../lib/errorMessage";

interface VersionsCardProps {
  componentId: string;
  selectedVersion?: string;
  onSelectVersion?: (version: string | undefined) => void;
}

const EMPTY_VERSIONS: Version[] = [];

/**
 * VersionsCard renders the version-history list for one component and
 * a built-in diff viewer that picks two versions (or one version and
 * an `adoption:<id>` ref) and renders aligned left/right rows.
 *
 * Surface for req 11 (VR-001..003).
 */
export function VersionsCard({ componentId, selectedVersion, onSelectVersion }: VersionsCardProps) {
  const { t } = useTranslation();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const versionsQuery = useQuery({
    queryKey: ["versions", componentId],
    queryFn: () => versionsClient.listVersions({ componentId, limit: 0 }),
  });

  const diffMutation = useMutation({
    mutationFn: () => versionsClient.diffVersions({ componentId, from, to }),
  });

  const versions: Version[] = versionsQuery.data?.versions ?? EMPTY_VERSIONS;
  const diff = diffMutation.data;

  const versionOptions = useMemo(
    () => versions.map((v) => v.version).filter((v) => v.length > 0),
    [versions],
  );

  return (
    <section
      data-testid={selectors.versions.card}
      aria-label={t(strings.versions.title)}
      className="mt-4 rounded-xl border border-app-border bg-app-surface p-4 backdrop-blur-sm"
    >
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-medium text-app-foreground">{t(strings.versions.title)}</h2>
        {versions.length > 0 && (
          <span className="text-xs text-app-muted-foreground">
            {t(strings.versions.summary, { count: versions.length })}
          </span>
        )}
      </header>

      {versionsQuery.isLoading && (
        <p data-testid={selectors.versions.loading} className="mt-3 text-app-foreground">
          {t(strings.versions.loading)}
        </p>
      )}
      {versionsQuery.error && (
        <p data-testid={selectors.versions.error} className="mt-3 text-app-danger">
          {errorMessage(versionsQuery.error, t)}
        </p>
      )}
      {!versionsQuery.isLoading && versions.length === 0 && (
        <div data-testid={selectors.versions.empty}>
          <EmptyState
            title={t(strings.versions.empty)}
            className="mt-3 border-app-border bg-app-surface text-app-foreground"
          />
        </div>
      )}

      {versions.length > 0 && (
        <>
          <Button
            type="button"
            variant={selectedVersion ? "secondary" : "primary"}
            className="mt-3 h-8 px-3 text-xs"
            onClick={() => onSelectVersion?.(undefined)}
          >
            {t(strings.versions.currentSource)}
          </Button>
          <ul data-testid={selectors.versions.list} className="mt-3 space-y-2 text-sm text-app-foreground">
          {versions.map((v) => (
            <li
              key={v.id}
              data-testid={selectors.versions.item}
              className="rounded-lg border border-app-border p-3"
            >
              <div className="flex items-baseline justify-between gap-3">
                <span data-testid={selectors.versions.itemVersion} className="font-medium">
                  {v.version
                    ? t(strings.versions.versionLabel, { version: v.version })
                    : "(no @version)"}
                </span>
                <span
                  data-testid={selectors.versions.itemSha}
                  className="font-mono text-xs text-app-muted-foreground"
                >
                  {t(strings.versions.shaLabel, { sha: v.contentSha256.slice(0, 12) })}
                </span>
              </div>
              <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-app-muted-foreground">
                <span data-testid={selectors.versions.itemRecordedAt}>
                  {t(strings.versions.recordedAt, {
                    when: v.recordedAt?.seconds
                      ? new Date(Number(v.recordedAt.seconds) * 1000).toLocaleString()
                      : "",
                  })}
                </span>
                {v.changelogMd && (
                  <span data-testid={selectors.versions.itemChangelog}>{v.changelogMd}</span>
                )}
              </div>
              <span data-testid={selectors.versions.itemId} className="sr-only">
                {v.id}
              </span>
              <Button
                type="button"
                variant={selectedVersion === v.version ? "primary" : "secondary"}
                className="mt-2 h-7 px-2 text-xs"
                onClick={() => onSelectVersion?.(v.version)}
              >
                {selectedVersion === v.version
                  ? t(strings.versions.viewingVersion)
                  : t(strings.versions.viewVersion)}
              </Button>
            </li>
          ))}
        </ul>
        </>
      )}

      <div
        data-testid={selectors.versions.diff.card}
        className="mt-4 rounded-lg border border-app-border bg-app-surface-muted p-3"
      >
        <h3 className="text-xs font-medium text-app-muted-foreground">{t(strings.versions.diff.title)}</h3>
        <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-app-muted-foreground">
          <label className="flex items-center gap-1.5">
            <span>{t(strings.versions.diff.fromLabel)}</span>
            <Select
              data-testid={selectors.versions.diff.fromSelect}
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="min-h-8 w-32 border-app-border bg-app-surface px-2 py-1 text-sm text-app-foreground"
              options={versionOptions.map((version) => ({ value: version, label: version }))}
              placeholder="—"
            />
          </label>
          <label className="flex items-center gap-1.5">
            <span>{t(strings.versions.diff.toLabel)}</span>
            <Select
              data-testid={selectors.versions.diff.toSelect}
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="min-h-8 w-32 border-app-border bg-app-surface px-2 py-1 text-sm text-app-foreground"
              options={versionOptions.map((version) => ({ value: version, label: version }))}
              placeholder="—"
            />
          </label>
          <Button
            data-testid={selectors.versions.diff.runButton}
            onClick={() => diffMutation.mutate()}
            disabled={!from || !to || diffMutation.isPending}
            className="h-7 px-3 text-xs"
          >
            {diffMutation.isPending
              ? t(strings.versions.diff.running)
              : t(strings.versions.diff.runAction)}
          </Button>
        </div>

        {diffMutation.error && (
          <p data-testid={selectors.versions.diff.error} className="mt-2 text-xs text-app-danger">
            {errorMessage(diffMutation.error, t)}
          </p>
        )}

        {!diff && !diffMutation.isPending && !diffMutation.error && (
          <p data-testid={selectors.versions.diff.empty} className="mt-2 text-xs text-app-muted-foreground">
            {t(strings.versions.diff.empty)}
          </p>
        )}

        {diff && (
          <>
            <p
              data-testid={selectors.versions.diff.summary}
              className="mt-2 text-xs text-app-muted-foreground"
            >
              {t(strings.versions.diff.summary, {
                from: diff.fromLabel,
                to: diff.toLabel,
                additions: diff.additions,
                removals: diff.removals,
                rows: diff.rows.length,
              })}
            </p>
            <div className="mt-2 overflow-auto">
              <div
                data-testid={selectors.versions.diff.table}
                role="table"
                className="w-full min-w-max font-mono text-[0.7rem]"
              >
                {diff.rows.map((r, i) => (
                  <DiffRowView key={i} row={r} />
                ))}
              </div>
            </div>
          </>
        )}
      </div>
    </section>
  );
}

function DiffRowView({ row }: { row: DiffRow }) {
  return (
    <div
      data-testid={selectors.versions.diff.row}
      role="row"
      className="grid grid-cols-2 align-top"
    >
      <CellView cell={row.left} side="left" />
      <CellView cell={row.right} side="right" />
    </div>
  );
}

function CellView({ cell, side }: { cell: { lineNumber: number; text: string; op: DiffOp } | undefined; side: "left" | "right" }) {
  if (!cell) return <div role="cell" />;
  const cls =
    cell.op === DiffOp.ADD
      ? "bg-app-success/10 text-app-success"
      : cell.op === DiffOp.REMOVE
      ? "bg-app-danger/10 text-app-danger"
      : cell.op === DiffOp.EMPTY
      ? "bg-app-surface-muted text-app-muted-foreground"
      : "text-app-muted-foreground";
  const marker = cell.op === DiffOp.ADD ? "+" : cell.op === DiffOp.REMOVE ? "-" : " ";
  return (
    <div role="cell" className={`whitespace-pre px-2 py-0.5 ${cls}`} data-side={side}>
      <span className="mr-2 inline-block w-5 select-none text-app-muted-foreground">
        {cell.op === DiffOp.EMPTY ? "" : cell.lineNumber || ""}
      </span>
      <span className="mr-1 inline-block w-3 select-none">{marker}</span>
      <HighlightedSource source={cell.text} />
    </div>
  );
}

function HighlightedSource({ source }: { source: string }) {
  const [html, setHTML] = useState<string>();

  useEffect(() => {
    let active = true;
    void import("shiki")
      .then(({ codeToHtml }) => codeToHtml(source, { lang: "tsx", theme: "github-dark" }))
      .then((rendered) => {
        if (active) setHTML(rendered);
      })
      .catch(() => {
        // The source remains visible if a language bundle cannot load.
        if (active) setHTML(undefined);
      });
    return () => { active = false; };
  }, [source]);

  if (!html) return <>{source}</>;
  // Shiki escapes source text before producing this markup; the component
  // never accepts HTML from an API response as executable markup.
  return <span className="inline [&_pre]:m-0 [&_pre]:inline [&_pre]:bg-transparent! [&_code]:bg-transparent!" dangerouslySetInnerHTML={{ __html: html }} />;
}
