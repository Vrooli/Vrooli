// ArtifactsPanel renders the per-flow file table on the Flow Detail
// page's Artifacts tab and exposes the Generate / Regenerate / Clear
// actions. The status pill matches the colour the row's status pill
// shows in the inventory list — same component, same colour mapping.
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  clearArtifacts,
  fetchArtifactsStatus,
  generateArtifacts,
  type ArtifactReport,
} from "../../api/artifacts";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

import { ArtifactStatusPill } from "./ArtifactStatusPill";

interface Props {
  flowId: string;
  scenarioId?: string;
}

const ARTIFACTS_KEY = (flowId: string) => ["artifacts", flowId] as const;
const FLOW_RUNS_KEY = (flowId: string) => ["runs", "flow", flowId] as const;
const ALL_RUNS_KEY = ["runs", "all"];

export function ArtifactsPanel({ flowId, scenarioId }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [confirmingClear, setConfirmingClear] = useState(false);

  const statusQuery = useQuery({
    queryKey: ARTIFACTS_KEY(flowId),
    queryFn: () => fetchArtifactsStatus(flowId, { scenarioId }),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ARTIFACTS_KEY(flowId) });
    void queryClient.invalidateQueries({ queryKey: FLOW_RUNS_KEY(flowId) });
    void queryClient.invalidateQueries({ queryKey: ALL_RUNS_KEY });
  };

  const generate = useMutation({
    mutationFn: () => generateArtifacts(flowId, { scenarioId }),
    onSuccess: invalidate,
  });

  const clear = useMutation({
    mutationFn: () => clearArtifacts(flowId, { scenarioId }),
    onSuccess: () => {
      setConfirmingClear(false);
      invalidate();
    },
  });

  const report = useMemo<ArtifactReport | undefined>(() => statusQuery.data, [statusQuery.data]);
  const isPending = generate.isPending || clear.isPending;

  if (statusQuery.isLoading) {
    return (
      <p data-testid="artifacts-loading" className="text-sm text-app-muted-foreground">
        {t("artifacts.loading", { defaultValue: "Loading artifact status…" })}
      </p>
    );
  }
  if (statusQuery.error || !report) {
    return (
      <p data-testid="artifacts-error" className="text-sm text-app-danger">
        {errorMessage(statusQuery.error, t)}
      </p>
    );
  }

  return (
    <section data-testid="artifacts-panel" className="flex flex-col gap-3 text-sm text-app-foreground">
      <header className="flex flex-wrap items-center gap-2">
        <ArtifactStatusPill status={report.status} testId="artifacts-status" />
        <span data-testid="artifacts-generated-dir" className="font-mono text-xs text-app-muted-foreground">
          {report.generatedDir}
        </span>
      </header>

      <table data-testid="artifacts-files" className="w-full text-left">
        <thead className="text-xs uppercase text-app-muted-foreground">
          <tr>
            <th className="py-1 pr-3">{t("artifacts.colPath", { defaultValue: "Path" })}</th>
            <th className="py-1 pr-3">{t("artifacts.colExists", { defaultValue: "Exists" })}</th>
            <th className="py-1 pr-3">{t("artifacts.colMtime", { defaultValue: "Modified" })}</th>
          </tr>
        </thead>
        <tbody>
          {report.files.map((file) => (
            <tr
              key={file.path}
              data-testid={`artifacts-row-${file.path}`}
              className="border-t border-app-border"
            >
              <td className="py-1 pr-3 font-mono text-xs">{file.path}</td>
              <td className="py-1 pr-3">{file.exists ? "✓" : "—"}</td>
              <td className="py-1 pr-3 text-xs text-app-muted-foreground">
                {file.mtime ? new Date(file.mtime).toLocaleString() : "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          data-testid="artifacts-generate"
          disabled={isPending}
          onClick={() => generate.mutate()}
          className="inline-flex h-8 items-center rounded-control border border-app-border bg-app-primary px-3 text-xs text-app-primary-foreground hover:opacity-90 disabled:opacity-60"
        >
          {generate.isPending
            ? t("artifacts.generating", { defaultValue: "Generating…" })
            : report.status === "fresh"
              ? t("artifacts.regenerate", { defaultValue: "Regenerate" })
              : t("artifacts.generate", { defaultValue: "Generate" })}
        </button>
        {!confirmingClear ? (
          <button
            type="button"
            data-testid="artifacts-clear"
            disabled={isPending || report.files.every((f) => !f.exists)}
            onClick={() => setConfirmingClear(true)}
            className="inline-flex h-8 items-center rounded-control border border-app-border bg-app-surface px-3 text-xs text-app-foreground hover:bg-app-surface-muted disabled:opacity-60"
          >
            {t("artifacts.clear", { defaultValue: "Clear" })}
          </button>
        ) : (
          <div data-testid="artifacts-clear-confirm" className="flex items-center gap-2">
            <span className="text-xs text-app-warning">
              {t("artifacts.clearConfirm", { defaultValue: "Remove every generated file?" })}
            </span>
            <button
              type="button"
              data-testid="artifacts-clear-yes"
              disabled={isPending}
              onClick={() => clear.mutate()}
              className="inline-flex h-7 items-center rounded-control border border-app-border bg-app-danger px-2 text-xs text-app-primary-foreground"
            >
              {t("artifacts.clearYes", { defaultValue: "Yes, remove" })}
            </button>
            <button
              type="button"
              data-testid="artifacts-clear-cancel"
              onClick={() => setConfirmingClear(false)}
              className="inline-flex h-7 items-center rounded-control border border-app-border bg-app-surface px-2 text-xs"
            >
              {t("artifacts.clearCancel", { defaultValue: "Cancel" })}
            </button>
          </div>
        )}
      </div>

      {generate.error && (
        <p data-testid="artifacts-generate-error" className="text-xs text-app-danger">
          {errorMessage(generate.error, t)}
        </p>
      )}
      {clear.error && (
        <p data-testid="artifacts-clear-error" className="text-xs text-app-danger">
          {errorMessage(clear.error, t)}
        </p>
      )}
    </section>
  );
}
