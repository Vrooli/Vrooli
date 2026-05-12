/**
 * ScenarioDetailPage — single scenario view.
 *
 * Renders the scenario header (name, description, path) and a table of
 * its flows. Each flow row links to the existing /flows/:flowId page.
 * Per-row + bulk verify lives here too so users can act inside the
 * scenario context they navigated into.
 */
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

import { ApiError } from "../api/client";
import { verifyFlow, fetchRuns, type RunRow } from "../api/inventory";
import { fetchScenarioDetail } from "../api/scenarios";
import { errorMessage } from "../lib/errorMessage";
import { useTranslation } from "../i18n";

export function ScenarioDetailPage() {
  const { t } = useTranslation();
  const { scenarioId = "" } = useParams<{ scenarioId: string }>();
  const queryClient = useQueryClient();

  const detail = useQuery({
    queryKey: ["scenarios", scenarioId],
    queryFn: () => fetchScenarioDetail(scenarioId),
    enabled: scenarioId !== "",
    retry: (count, err) => {
      if (err instanceof ApiError && err.status === 404) return false;
      return count < 2;
    },
  });

  const runs = useQuery({
    queryKey: ["runs", "all"],
    queryFn: () => fetchRuns({ limit: 200 }),
  });

  const latestByFlow = useMemo(() => {
    const m = new Map<string, RunRow>();
    for (const r of runs.data ?? []) {
      const existing = m.get(r.flowId);
      if (!existing || r.finishedAt > existing.finishedAt) m.set(r.flowId, r);
    }
    return m;
  }, [runs.data]);

  const verifyOne = useMutation({
    mutationFn: async (flowId: string) => {
      // ?scenario= isn't directly accepted by /verifications yet; we
      // pass the scenario's filesystem path as root so the existing
      // verify pipeline finds the flow without changes to that domain.
      const root = detail.data?.path ?? "";
      await verifyFlow(root, flowId);
      await queryClient.invalidateQueries({ queryKey: ["runs", "all"] });
    },
  });
  const verifyAll = useMutation({
    mutationFn: async () => {
      const root = detail.data?.path ?? "";
      for (const f of detail.data?.flows ?? []) {
        await verifyFlow(root, f.flowId);
        await queryClient.invalidateQueries({ queryKey: ["runs", "all"] });
      }
    },
  });

  if (detail.isLoading) {
    return (
      <p data-testid="scenario-detail-loading" className="text-sm text-app-muted-foreground">
        {t("scenarioDetail.loading", { defaultValue: "Loading scenario…" })}
      </p>
    );
  }
  if (detail.error) {
    const is404 = detail.error instanceof ApiError && detail.error.status === 404;
    return (
      <div data-testid="scenario-detail-error" className="flex flex-col gap-2">
        <p className="text-sm text-app-danger">
          {is404
            ? t("scenarioDetail.notFound", {
                defaultValue: "No scenario with id “{{id}}”.",
                id: scenarioId,
              })
            : errorMessage(detail.error, t)}
        </p>
        <Link to="/scenarios" className="text-sm text-app-primary hover:underline">
          {t("scenarioDetail.backToList", { defaultValue: "← Back to scenarios" })}
        </Link>
      </div>
    );
  }
  if (!detail.data) return null;

  const { displayName, description, path, flows, discoveryError } = detail.data;

  return (
    <div data-testid="scenario-detail-page" className="flex flex-col gap-4">
      <nav className="text-xs text-app-muted-foreground">
        <Link
          to="/scenarios"
          data-testid="scenario-detail-breadcrumb"
          className="hover:underline"
        >
          {t("scenarioDetail.breadcrumb", { defaultValue: "Scenarios" })}
        </Link>
        <span aria-hidden> / </span>
        <span className="text-app-foreground">{scenarioId}</span>
      </nav>

      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1
            data-testid="scenario-detail-heading"
            className="text-2xl font-semibold text-app-foreground"
          >
            {displayName}
          </h1>
          <p className="mt-1 font-mono text-xs text-app-muted-foreground">{scenarioId}</p>
          {description && (
            <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{description}</p>
          )}
          <p className="mt-1 text-xs text-app-muted-foreground">
            <span className="opacity-70">{t("scenarioDetail.path", { defaultValue: "Path:" })}</span>{" "}
            <code>{path}</code>
          </p>
        </div>
        {flows.length > 0 && (
          <button
            type="button"
            data-testid="scenario-detail-verify-all"
            onClick={() => verifyAll.mutate()}
            disabled={verifyAll.isPending}
            className="inline-flex h-9 items-center rounded-control bg-app-primary px-4 text-sm font-medium text-app-primary-foreground hover:brightness-95 disabled:opacity-60"
          >
            {verifyAll.isPending
              ? t("scenarioDetail.verifyingAll", { defaultValue: "Verifying all…" })
              : t("scenarioDetail.verifyAll", { defaultValue: "Verify all" })}
          </button>
        )}
      </header>

      {discoveryError && (
        <p data-testid="scenario-detail-discovery-error" className="text-sm text-app-danger">
          {discoveryError}
        </p>
      )}

      {flows.length === 0 ? (
        <p data-testid="scenario-detail-empty" className="text-sm text-app-muted-foreground">
          {t("scenarioDetail.empty", {
            defaultValue:
              "This scenario has no flow.json files yet. Add one under a feature directory to see it here.",
          })}
        </p>
      ) : (
        <table
          data-testid="scenario-detail-flows"
          className="w-full border-collapse text-left text-sm"
        >
          <thead>
            <tr className="border-b border-app-border text-xs uppercase tracking-wide text-app-muted-foreground">
              <th className="py-2 pr-3 font-medium">
                {t("scenarioDetail.col.flow", { defaultValue: "Flow" })}
              </th>
              <th className="py-2 pr-3 font-medium">
                {t("scenarioDetail.col.language", { defaultValue: "Language" })}
              </th>
              <th className="py-2 pr-3 font-medium">
                {t("scenarioDetail.col.status", { defaultValue: "Last run" })}
              </th>
              <th className="py-2 font-medium" />
            </tr>
          </thead>
          <tbody>
            {flows.map((f) => {
              const last = latestByFlow.get(f.flowId);
              return (
                <tr
                  key={f.flowId}
                  data-testid={`scenario-detail-row-${f.flowId}`}
                  className="border-b border-app-border last:border-b-0"
                >
                  <td className="py-2 pr-3">
                    <Link
                      to={`/flows/${encodeURIComponent(f.flowId)}?scenario=${encodeURIComponent(scenarioId)}`}
                      className="font-mono text-app-primary hover:underline"
                    >
                      {f.flowId}
                    </Link>
                    <div className="font-mono text-xs text-app-muted-foreground">
                      {f.contractPath}
                    </div>
                  </td>
                  <td className="py-2 pr-3 text-xs text-app-muted-foreground">{f.language}</td>
                  <td className="py-2 pr-3 text-xs">
                    {last ? (
                      <Link
                        to={`/runs/${last.id}`}
                        className={statusClass(last.status)}
                      >
                        {last.status}
                      </Link>
                    ) : (
                      <span className="text-app-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="py-2 text-right">
                    <button
                      type="button"
                      data-testid={`scenario-detail-verify-${f.flowId}`}
                      onClick={() => verifyOne.mutate(f.flowId)}
                      disabled={verifyOne.isPending}
                      className="inline-flex h-7 items-center rounded-control border border-app-border bg-app-surface px-3 text-xs font-medium text-app-foreground hover:bg-app-surface-muted disabled:opacity-60"
                    >
                      {verifyOne.isPending && verifyOne.variables === f.flowId
                        ? t("scenarioDetail.verifying", { defaultValue: "…" })
                        : t("scenarioDetail.verify", { defaultValue: "Verify" })}
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}

      {verifyAll.error && (
        <p data-testid="scenario-detail-verifyall-error" className="text-sm text-app-danger">
          {errorMessage(verifyAll.error, t)}
        </p>
      )}
      {verifyOne.error && (
        <p data-testid="scenario-detail-verifyone-error" className="text-sm text-app-danger">
          {errorMessage(verifyOne.error, t)}
        </p>
      )}
    </div>
  );
}

function statusClass(status: RunRow["status"]): string {
  const base = "rounded-control px-2 py-0.5 font-medium hover:underline";
  switch (status) {
    case "passed":
      return `${base} bg-app-success/10 text-app-success`;
    case "failed":
      return `${base} bg-app-danger/10 text-app-danger`;
    default:
      return `${base} bg-app-warning/10 text-app-warning`;
  }
}
