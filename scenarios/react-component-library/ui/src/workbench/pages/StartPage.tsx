import { useMemo } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  ArrowUpRight,
  Blocks,
  Boxes,
  CircleAlert,
  Compass,
  FileText,
  FormInput,
  LayoutTemplate,
  Layers,
  Sparkles,
  Table2,
  Workflow,
} from "lucide-react";

import { getReviewSummary } from "../../api/review";
import { WorkspaceProvenance } from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { appRoutes, libraryPath, statsPath, workspacePath } from "../../routes";
import {
  assetDomain,
  formatNumber,
  formatPercent,
  timestampToDate,
  useCatalog,
  useCoverage,
} from "../data";
import {
  Banner,
  Dot,
  Empty,
  Panel,
  PanelHead,
  Skeleton,
} from "../primitives";
import { SearchSurface } from "../search/SearchSurface";
import { useWorkspaces } from "../workspaces/useWorkspaces";

/** The intent a domain usually serves. This is presentation, not data. */
const DOMAIN_ICON: Record<string, typeof Boxes> = {
  controls: Blocks,
  data: Table2,
  forms: FormInput,
  navigation: Compass,
  layout: LayoutTemplate,
  feedback: Sparkles,
  content: FileText,
  motion: Workflow,
};

const SHORTCUTS = [
  { key: "workbench.start.shortcutSearch", label: "Search everything", keys: "⌘K" },
  { key: "workbench.start.shortcutNew", label: "New workspace", keys: "N" },
  { key: "workbench.start.shortcutDesign", label: "Go to Design", keys: "G D" },
  { key: "workbench.start.shortcutStats", label: "Go to Stats", keys: "G S" },
] as const;

/** A domain rendered as the intent it serves. The filter still uses the real
 * domain string, so the label is presentation on top of the catalog, not a
 * second taxonomy. */
const DOMAIN_LABEL: Record<string, string> = {
  controls: "Actions & controls",
  "data-display": "Collections & data",
  forms: "Forms & auth",
  hooks: "Hooks & runtime",
  legacy: "Legacy surface",
  navigation: "Navigation & shell",
  primitives: "Foundations",
  uncategorized: "Uncategorised",
};

function labelForDomain(domain: string): string {
  if (DOMAIN_LABEL[domain]) return DOMAIN_LABEL[domain];
  return domain.replace(/[-_]/g, " ").replace(/^\w/, (letter) => letter.toUpperCase());
}

function shortAgo(value: string): string {
  const date = timestampToDate(value);
  if (!date) return "";
  const seconds = Math.max(0, (Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  if (seconds < 86_400) return `${Math.round(seconds / 3600)}h ago`;
  return `${Math.round(seconds / 86_400)}d ago`;
}

function pageCount(count: number): string {
  return count === 1 ? "1 page" : `${count} pages`;
}

function iconForDomain(domain: string): typeof Boxes {
  const lower = domain.toLowerCase();
  for (const [key, icon] of Object.entries(DOMAIN_ICON)) {
    if (lower.includes(key)) return icon;
  }
  return Boxes;
}

/**
 * Start is a way in, not a dashboard. It answers “what can I make or find?”
 * with one search, a browse-by-intent grid, the workspaces this browser is
 * already in, and a compact pulse rail. Counts live in Stats; the pulse only
 * keeps one number in reach.
 */
export function StartPage() {
  const { t } = useTranslation();
  const catalog = useCatalog();
  const coverage = useCoverage();
  const workspaces = useWorkspaces();
  const review = useQuery({
    queryKey: ["review", "summary"],
    queryFn: getReviewSummary,
    retry: 0,
  });

  const assets = useMemo(() => catalog.data ?? [], [catalog.data]);
  const summaries = workspaces.data ?? [];
  const curated = summaries.filter(
    (workspace) => workspace.provenance === WorkspaceProvenance.CURATED,
  );
  const recent = summaries
    .filter((workspace) => workspace.provenance !== WorkspaceProvenance.CURATED)
    .slice(0, 3);
  const openFindings = review.data?.open;
  const report = coverage.data;
  const production = report?.maturity?.productionReadyCoverage;

  const intents = useMemo(() => {
    const byDomain = new Map<string, { count: number; examples: string[] }>();
    for (const asset of assets) {
      const domain = assetDomain(asset);
      const entry = byDomain.get(domain) ?? { count: 0, examples: [] };
      entry.count += 1;
      if (entry.examples.length < 3) entry.examples.push(asset.displayName);
      byDomain.set(domain, entry);
    }
    return [...byDomain.entries()]
      .sort(([leftName, left], [rightName, right]) =>
        right.count === left.count ? leftName.localeCompare(rightName) : right.count - left.count,
      )
      .slice(0, 8)
      .map(([domain, entry]) => ({ domain, ...entry }));
  }, [assets]);

  const continuation = recent[0];

  return (
    <div className="wb-page wb-start" data-experience-surface="start" data-testid="start-page">
      <header className="wb-start-head">
        <p className="wb-start-kicker">
          {catalog.isLoading
            ? t("workbench.start.loadingKicker", { defaultValue: "The shared UI layer · loading…" })
            : t("workbench.start.kicker", {
                defaultValue: "The shared UI layer · {{count}} assets · {{workspaces}} workspaces",
                count: assets.length,
                workspaces: summaries.length,
              })}
        </p>
        <h1 className="wb-start-h1">
          {t("workbench.start.titleLead", { defaultValue: "Find the" })}{" "}
          <em>{t("workbench.start.titleEmphasis", { defaultValue: "surface." })}</em>
        </h1>
        <p className="wb-start-lede">
          {t("workbench.start.lede", {
            defaultValue:
              "Start from an intent, a workspace, an asset or a command. Nothing here requires a query.",
          })}
        </p>
        <div className="wb-start-search">
          <SearchSurface compact />
        </div>
      </header>

      <div className="wb-start-layout">
        <div className="wb-start-main">
          <section aria-labelledby="start-intent-title">
            <div className="wb-start-section-head">
              <b id="start-intent-title">
                {t("workbench.start.browseTitle", { defaultValue: "Browse by intent" })}
              </b>
              <span>
                {t("workbench.start.browseMeta", {
                  defaultValue: "From the catalog · {{count}}",
                  count: assets.length,
                })}
              </span>
            </div>
            {catalog.isLoading ? (
              <div className="wb-start-catgrid" data-testid="start-categories-loading">
                {Array.from({ length: 4 }).map((_, index) => (
                  <Skeleton key={index} height={104} />
                ))}
              </div>
            ) : catalog.isError ? (
              <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
                <b>
                  {t("workbench.start.catalogUnavailableTitle", {
                    defaultValue: "The catalog did not answer.",
                  })}
                </b>{" "}
                {t("workbench.start.catalogUnavailableBody", {
                  defaultValue:
                    "Domains cannot be derived without it. The library is not shown as empty.",
                })}
              </Banner>
            ) : intents.length === 0 ? (
              <Empty
                title={t("workbench.start.emptyCategoriesTitle", {
                  defaultValue: "No domains indexed yet",
                })}
                action={
                  <Link to={appRoutes.create} className="wb-btn ghost sm">
                    {t("workbench.start.emptyCategoriesAction", {
                      defaultValue: "Create the first asset",
                    })}
                  </Link>
                }
              >
                {t("workbench.start.emptyCategoriesBody", {
                  defaultValue:
                    "Category cards come from the catalog's own domains; add an asset or reindex to populate them.",
                })}
              </Empty>
            ) : (
              <div className="wb-start-catgrid">
                {intents.map((intent) => {
                  const Icon = iconForDomain(intent.domain);
                  return (
                    <Link
                      key={intent.domain}
                      to={libraryPath(intent.domain)}
                      className="wb-start-cat"
                      data-testid="start-category-card"
                    >
                      <span className="wb-start-cat-icon" aria-hidden>
                        <Icon size={15} />
                      </span>
                      <b>{labelForDomain(intent.domain)}</b>
                      <span className="wb-start-cat-count">
                        {t("workbench.start.categoryCount", {
                          defaultValue: "{{count}} assets · {{domain}}",
                          count: intent.count,
                          domain: intent.domain,
                        })}
                      </span>
                      <span className="wb-start-cat-examples">{intent.examples.join(" · ")}</span>
                    </Link>
                  );
                })}
              </div>
            )}
          </section>

          <section aria-labelledby="start-recent-title">
            <div className="wb-start-section-head">
              <b id="start-recent-title">
                {t("workbench.start.continue", { defaultValue: "Recently opened" })}
              </b>
              <span>
                {t("workbench.start.recentMeta", { defaultValue: "This browser" })}
              </span>
            </div>
            {workspaces.isLoading ? (
              <div className="wb-start-strip" data-testid="start-designs-loading">
                {Array.from({ length: 3 }).map((_, index) => (
                  <Skeleton key={index} height={64} />
                ))}
              </div>
            ) : workspaces.isError ? (
              <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                {t("workbench.start.designsUnavailable", {
                  defaultValue:
                    "The workspace list is unavailable. Open Design to retry the design service.",
                })}
              </Banner>
            ) : recent.length === 0 ? (
              <Empty
                title={t("workbench.start.noDesigns", { defaultValue: "No workspaces yet" })}
                action={
                  <Link to={appRoutes.design} className="wb-btn ghost sm">
                    {t("workbench.start.openDesignArchive", {
                      defaultValue: "Open the design index",
                    })}
                  </Link>
                }
              >
                {t("workbench.start.noDesignsBody", {
                  defaultValue:
                    "The design index lists workspaces and templates once the studio has created one.",
                })}
              </Empty>
            ) : (
              <div className="wb-start-strip">
                {recent.map((workspace, index) => (
                  <Link
                    key={workspace.id}
                    to={workspacePath(workspace.id)}
                    className="wb-start-recent"
                    aria-label={t("workbench.start.openWorkspace", {
                      defaultValue: "Open workspace {{name}}",
                      name: workspace.name,
                    })}
                  >
                    <span
                      className="wb-start-recent-art"
                      data-tone={index % 3}
                      aria-hidden
                    >
                      <Layers size={14} />
                    </span>
                    <span className="wb-start-recent-copy">
                      <b>{workspace.name}</b>
                      <small>
                        {t("workbench.start.recentMetaLine", {
                          defaultValue: "{{pages}} · {{updated}}",
                          pages: pageCount(workspace.pageCount),
                          updated: shortAgo(workspace.updatedAt),
                        })}
                      </small>
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </section>
        </div>

        <aside className="wb-start-rail" aria-label={t("workbench.start.rail", { defaultValue: "Start sidebar" })}>
          {continuation ? (
            <div className="wb-start-continue">
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
                <b>{catalog.isError ? "—" : catalog.isLoading ? "…" : formatNumber(assets.length)}</b>
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
                <b>{workspaces.isError ? "—" : workspaces.isLoading ? "…" : formatNumber(summaries.length)}</b>
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
                          displays: workspace.displayCount === 1 ? "1 display" : `${workspace.displayCount} displays`,
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
      </div>

      <p className="wb-start-foot">
        <Dot tone="green" />
        {t("workbench.start.footNote", {
          defaultValue:
            "Search is server-ranked across workspaces, pages, displays, states, page types, assets, commands and evidence.",
        })}
        <Link to={appRoutes.library}>
          {t("workbench.start.footLibrary", { defaultValue: "Browse the whole library" })}
          <ArrowRight aria-hidden size={12} />
        </Link>
      </p>
    </div>
  );
}
