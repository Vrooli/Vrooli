import { Link } from "react-router-dom";
import { ArrowUpRight } from "lucide-react";

import type { WorkspaceSummary } from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { appRoutes, statsPath, workspacePath } from "../../routes";
import { formatNumber, formatPercent, type CoverageReport } from "../data";
import { Panel, PanelHead, Skeleton } from "../primitives";
import type { UseQueryResult } from "@tanstack/react-query";
import type { CatalogAsset } from "../../api/components";
import type { useWorkspaces } from "../workspaces/useWorkspaces";

const SHORTCUTS = [
  { key: "workbench.start.shortcutSearch", label: "Search everything", keys: "⌘K" },
  { key: "workbench.start.shortcutNew", label: "New workspace", keys: "N" },
  { key: "workbench.start.shortcutDesign", label: "Go to Design", keys: "G D" },
  { key: "workbench.start.shortcutStats", label: "Go to Stats", keys: "G S" },
] as const;

function pageCount(count: number): string {
  return count === 1 ? "1 page" : `${count} pages`;
}

function shortAgo(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const seconds = Math.max(0, (Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  if (seconds < 86_400) return `${Math.round(seconds / 3600)}h ago`;
  return `${Math.round(seconds / 86_400)}d ago`;
}

type WorkspaceQuery = ReturnType<typeof useWorkspaces>;

export function StartPageRail({
  catalog,
  coverage,
  review,
  workspaces,
  summaries,
  curated,
  continuation,
}: {
  catalog: UseQueryResult<CatalogAsset[]>;
  coverage: UseQueryResult<CoverageReport>;
  review: { data?: { open?: number } | null; isError: boolean };
  workspaces: WorkspaceQuery;
  summaries: WorkspaceSummary[];
  curated: WorkspaceSummary[];
  continuation?: WorkspaceSummary;
}) {
  const { t } = useTranslation();
  const production = coverage.data?.maturity?.productionReadyCoverage;
  const openFindings = review.data?.open;

  return (
    <aside
      className="wb-start-rail"
      aria-label={t("workbench.start.rail", { defaultValue: "Start sidebar" })}
    >
      {continuation ? (
        <div className="wb-start-continue wb-start-continue-rail" data-testid="start-continue-rail">
          <span className="wb-start-rail-kicker">
            {t("workbench.start.continueLabel", { defaultValue: "Continue" })}
          </span>
          <b>{continuation.name}</b>
          <small>
            {t("workbench.start.recentMetaLine", {
              defaultValue: "{{pages}} · {{updated}}",
              pages: pageCount(continuation.pageCount),
              updated: shortAgo(continuation.updatedAt),
            })}
          </small>
          <div className="wb-start-continue-actions">
            <Link to={workspacePath(continuation.id)} className="wb-btn primary sm">
              {t("workbench.start.openWorkspaceShort", { defaultValue: "Open workspace" })}
            </Link>
            <Link to={appRoutes.design} className="wb-btn quiet sm">
              {t("workbench.start.allWorkspaces", { defaultValue: "All" })}
            </Link>
          </div>
        </div>
      ) : null}
      <Panel className="wb-start-pulse">
        <PanelHead
          title={t("workbench.start.pulse", { defaultValue: "Library pulse" })}
          meta={t("workbench.start.pulseMeta", { defaultValue: "Live" })}
        />
        <div className="wb-start-pulse-rows">
          <div className="wb-start-pulse-row">
            <span>{t("workbench.start.kpiAssets", { defaultValue: "Assets" })}</span>
            <b>
              {catalog.isError
                ? "—"
                : catalog.isLoading
                  ? "…"
                  : formatNumber(catalog.data?.length ?? 0)}
            </b>
          </div>
          <div className="wb-start-pulse-row">
            <span>{t("workbench.start.kpiProduction", { defaultValue: "Production ready" })}</span>
            <b data-tone={production && production.ratio < 0.8 ? "warn" : undefined}>
              {coverage.isError ? "—" : production ? formatPercent(production.ratio) : "…"}
            </b>
          </div>
          <div className="wb-start-pulse-row">
            <span>{t("workbench.start.openFindings", { defaultValue: "Open findings" })}</span>
            <b data-tone={openFindings && openFindings > 0 ? "warn" : undefined}>
              {review.isError ? "—" : openFindings === undefined ? "…" : formatNumber(openFindings)}
            </b>
          </div>
          <div className="wb-start-pulse-row">
            <span>{t("workbench.start.kpiWorkspaces", { defaultValue: "Workspaces" })}</span>
            <b>
              {workspaces.isError
                ? "—"
                : workspaces.isLoading
                  ? "…"
                  : formatNumber(summaries.length)}
            </b>
          </div>
        </div>
        <Link to={statsPath("overview")} className="wb-btn quiet sm wb-start-rail-button">
          {t("workbench.start.openStats", { defaultValue: "Open Stats" })}
          <ArrowUpRight aria-hidden size={12} />
        </Link>
      </Panel>

      <Panel className="wb-start-pulse">
        <PanelHead
          title={t("workbench.start.templates", { defaultValue: "Curated templates" })}
          meta={formatNumber(curated.length)}
        />
        <div className="wb-start-template-list">
          {workspaces.isLoading ? (
            <Skeleton height={48} />
          ) : curated.length === 0 ? (
            <p className="wb-start-rail-note">
              {t("workbench.start.noTemplatesBody", {
                defaultValue:
                  "A curated template is a reviewed workspace snapshot. It forks on use and never changes the original.",
              })}
            </p>
          ) : (
            curated.slice(0, 3).map((workspace, index) => (
              <Link
                key={workspace.id}
                to={workspacePath(workspace.id)}
                className="wb-start-template"
              >
                <span className="wb-start-template-thumb" data-tone={index % 3} aria-hidden>
                  <span />
                </span>
                <span className="wb-start-template-copy">
                  <b>{workspace.name}</b>
                  <small>
                    {t("workbench.start.templateMeta", {
                      defaultValue: "{{pages}} · {{displays}}",
                      pages: pageCount(workspace.pageCount),
                      displays:
                        workspace.displayCount === 1
                          ? "1 display"
                          : `${workspace.displayCount} displays`,
                    })}
                  </small>
                </span>
              </Link>
            ))
          )}
        </div>
      </Panel>

      <Panel className="wb-start-pulse">
        <PanelHead
          title={t("workbench.start.shortcuts", { defaultValue: "Shortcuts" })}
          meta={t("workbench.start.keyboard", { defaultValue: "Keyboard" })}
        />
        <div className="wb-start-shortcuts">
          {SHORTCUTS.map((shortcut) => (
            <div key={shortcut.key} className="wb-start-shortcut">
              <span>{t(shortcut.key, { defaultValue: shortcut.label })}</span>
              <kbd>{shortcut.keys}</kbd>
            </div>
          ))}
        </div>
      </Panel>
    </aside>
  );
}
