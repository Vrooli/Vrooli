import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
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
}

const EMPTY_VERSIONS: Version[] = [];

/**
 * VersionsCard renders the version-history list for one component and
 * a built-in diff viewer that picks two versions (or one version and
 * an `adoption:<id>` ref) and renders aligned left/right rows.
 *
 * Surface for req 11 (VR-001..003).
 */
export function VersionsCard({ componentId }: VersionsCardProps) {
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
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4 backdrop-blur-sm"
    >
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-medium text-slate-200">{t(strings.versions.title)}</h2>
        {versions.length > 0 && (
          <span className="text-xs text-slate-400">
            {t(strings.versions.summary, { count: versions.length })}
          </span>
        )}
      </header>

      {versionsQuery.isLoading && (
        <p data-testid={selectors.versions.loading} className="mt-3 text-slate-200">
          {t(strings.versions.loading)}
        </p>
      )}
      {versionsQuery.error && (
        <p data-testid={selectors.versions.error} className="mt-3 text-red-400">
          {errorMessage(versionsQuery.error, t)}
        </p>
      )}
      {!versionsQuery.isLoading && versions.length === 0 && (
        <p data-testid={selectors.versions.empty} className="mt-3 text-slate-200">
          {t(strings.versions.empty)}
        </p>
      )}

      {versions.length > 0 && (
        <ul data-testid={selectors.versions.list} className="mt-3 space-y-2 text-sm text-slate-200">
          {versions.map((v) => (
            <li
              key={v.id}
              data-testid={selectors.versions.item}
              className="rounded-lg border border-white/10 p-3"
            >
              <div className="flex items-baseline justify-between gap-3">
                <span data-testid={selectors.versions.itemVersion} className="font-medium">
                  {v.version
                    ? t(strings.versions.versionLabel, { version: v.version })
                    : "(no @version)"}
                </span>
                <span
                  data-testid={selectors.versions.itemSha}
                  className="font-mono text-xs text-slate-500"
                >
                  {t(strings.versions.shaLabel, { sha: v.contentSha256.slice(0, 12) })}
                </span>
              </div>
              <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
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
            </li>
          ))}
        </ul>
      )}

      <div
        data-testid={selectors.versions.diff.card}
        className="mt-4 rounded-lg border border-white/10 bg-black/30 p-3"
      >
        <h3 className="text-xs font-medium text-slate-400">{t(strings.versions.diff.title)}</h3>
        <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-slate-300">
          <label className="flex items-center gap-1.5">
            <span>{t(strings.versions.diff.fromLabel)}</span>
            <select
              data-testid={selectors.versions.diff.fromSelect}
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="rounded border border-white/10 bg-black/40 px-2 py-1 text-slate-100"
            >
              <option value="">—</option>
              {versionOptions.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          </label>
          <label className="flex items-center gap-1.5">
            <span>{t(strings.versions.diff.toLabel)}</span>
            <select
              data-testid={selectors.versions.diff.toSelect}
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="rounded border border-white/10 bg-black/40 px-2 py-1 text-slate-100"
            >
              <option value="">—</option>
              {versionOptions.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
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
          <p data-testid={selectors.versions.diff.error} className="mt-2 text-red-400 text-xs">
            {errorMessage(diffMutation.error, t)}
          </p>
        )}

        {!diff && !diffMutation.isPending && !diffMutation.error && (
          <p data-testid={selectors.versions.diff.empty} className="mt-2 text-xs text-slate-500">
            {t(strings.versions.diff.empty)}
          </p>
        )}

        {diff && (
          <>
            <p
              data-testid={selectors.versions.diff.summary}
              className="mt-2 text-xs text-slate-400"
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
              <table
                data-testid={selectors.versions.diff.table}
                className="w-full font-mono text-[0.7rem]"
              >
                <tbody>
                  {diff.rows.map((r, i) => (
                    <DiffRowView key={i} row={r} />
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </section>
  );
}

function DiffRowView({ row }: { row: DiffRow }) {
  return (
    <tr
      data-testid={selectors.versions.diff.row}
      className="align-top"
    >
      <CellView cell={row.left} side="left" />
      <CellView cell={row.right} side="right" />
    </tr>
  );
}

function CellView({ cell, side }: { cell: { lineNumber: number; text: string; op: DiffOp } | undefined; side: "left" | "right" }) {
  if (!cell) return <td />;
  const cls =
    cell.op === DiffOp.ADD
      ? "bg-emerald-900/40 text-emerald-200"
      : cell.op === DiffOp.REMOVE
      ? "bg-red-900/40 text-red-200"
      : cell.op === DiffOp.EMPTY
      ? "bg-black/30 text-slate-700"
      : "text-slate-300";
  const marker = cell.op === DiffOp.ADD ? "+" : cell.op === DiffOp.REMOVE ? "-" : " ";
  return (
    <td className={`whitespace-pre px-2 py-0.5 ${cls}`} data-side={side}>
      <span className="mr-2 inline-block w-5 select-none text-slate-500">
        {cell.op === DiffOp.EMPTY ? "" : cell.lineNumber || ""}
      </span>
      <span className="mr-1 inline-block w-3 select-none">{marker}</span>
      {cell.text}
    </td>
  );
}
