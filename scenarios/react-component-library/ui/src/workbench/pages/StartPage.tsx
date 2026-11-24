import "../start.css";
import { useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight, CircleAlert } from "lucide-react";

import { getReviewSummary } from "../../api/review";
import { SearchKind } from "../../api/search";
import { useTranslation } from "../../i18n";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { appRoutes, workspacePath } from "../../routes";
import { useCatalog, useCoverage } from "../data";
import { Banner, Dot, Empty, Skeleton } from "../primitives";
import { StartPageRail } from "./StartPageRail";
import { SearchSurface } from "../search/SearchSurface";
import { useWorkbenchSearch } from "../search/useWorkbenchSearch";
import { useWorkspaces } from "../workspaces/useWorkspaces";
import { useStartPageEntries } from "./startPageEntries";

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

const MOBILE_COMMAND_FILTER = [SearchKind.COMMAND];
const NO_SEARCH_KINDS: SearchKind[] = [];
const LANDING_PAGE_TYPE_FILTER = [SearchKind.PAGE_TYPE];

/**
 * Start is a way in, not a dashboard. It answers “what can I make or find?”
 * with one search, a browse-by-intent grid, the workspaces this browser is
 * already in, and a compact pulse rail. Counts live in Stats; the pulse only
 * keeps one number in reach.
 */
export function StartPage() {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [searchParams, setSearchParams] = useSearchParams();
  const catalog = useCatalog();
  const coverage = useCoverage();
  const workspaces = useWorkspaces();
  const review = useQuery({
    queryKey: ["review", "summary"],
    queryFn: getReviewSummary,
    retry: 0,
  });
  const commands = useWorkbenchSearch("", {
    kinds: MOBILE_COMMAND_FILTER,
    limit: 100,
    enabled: isMobile,
  });
  // Page archetypes are indexed separately from the publishable asset catalog.
  // Use the unified search owner so Start does not report a false empty state.
  const landingTypes = useWorkbenchSearch("landing page", {
    kinds: LANDING_PAGE_TYPE_FILTER,
    limit: 20,
  });

  const assets = useMemo(() => catalog.data ?? [], [catalog.data]);
  const summaries = useMemo(() => workspaces.data ?? [], [workspaces.data]);
  const translate = (key: string, options?: Record<string, string | number>) => {
    const translated = t(key as never, options as never);
    return typeof translated === "string" ? translated : translated.res;
  };
  const { curated, recent, browseEntries, mobilePrimaryIntents } = useStartPageEntries(
    assets,
    summaries,
    commands,
    landingTypes,
    translate,
  );

  const continuation = recent[0];
  // On phones the continuation card is already the primary recent workspace.
  // Keep the later list useful by showing only the remaining workspaces.
  const recentlyOpened = isMobile && continuation ? recent.slice(1) : recent;
  const primaryIntents = isMobile ? mobilePrimaryIntents : browseEntries;
  const primaryIDs = new Set(mobilePrimaryIntents.map((entry) => entry.id));
  const remainingIntents = isMobile
    ? browseEntries.filter((entry) => !primaryIDs.has(entry.id))
    : [];

  const categoryCards = (items: typeof browseEntries) =>
    items.map((intent) => {
      const Icon = intent.Icon;
      return (
        <Link
          key={intent.id}
          to={intent.href}
          className="wb-start-cat"
          data-testid="start-category-card"
          data-start-entry={intent.id}
          data-entry-count={intent.countValue}
          data-entry-pages={intent.pagesValue}
        >
          <span className="wb-start-cat-icon" aria-hidden>
            <Icon size={15} />
          </span>
          <b>{intent.title}</b>
          <span className="wb-start-cat-count">{intent.count}</span>
          <span className="wb-start-cat-examples">
            {Array.isArray(intent.examples) ? intent.examples.join(" · ") : intent.examples}
          </span>
        </Link>
      );
    });

  const clearCommandFilter = () => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete("commands");
        return next;
      },
      { replace: true },
    );
  };

  const continueCard = continuation ? (
    <div className="wb-start-continue" data-testid="start-continue">
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
  ) : null;

  return (
    <div className="wb-page wb-start" data-experience-surface="start" data-testid="start-page">
      <div className="wb-start-layout">
        <div className="wb-start-main">
          <header className="wb-start-head">
            <p className="wb-start-kicker">
              {catalog.isLoading
                ? t("workbench.start.loadingKicker", {
                    defaultValue: "The shared UI layer · loading…",
                  })
                : t("workbench.start.kicker", {
                    defaultValue:
                      "The shared UI layer · {{count}} assets · {{workspaces}} workspaces",
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
              <SearchSurface
                compact
                initialKinds={
                  searchParams.get("commands") === "1" ? MOBILE_COMMAND_FILTER : NO_SEARCH_KINDS
                }
                onClearKinds={clearCommandFilter}
              />
            </div>
          </header>
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
            ) : browseEntries.length === 0 && !isMobile ? (
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
              <div className="wb-start-catgrid" data-testid="start-intents-primary">
                {categoryCards(primaryIntents)}
              </div>
            )}
          </section>

          {isMobile ? continueCard : null}

          {isMobile ? (
            <>
              {remainingIntents.length > 0 ? (
                <section
                  aria-labelledby="start-more-intents-title"
                  data-testid="start-intents-more"
                >
                  <div className="wb-start-section-head">
                    <b id="start-more-intents-title">
                      {t("workbench.start.moreBrowseTitle", { defaultValue: "More to explore" })}
                    </b>
                  </div>
                  <div className="wb-start-catgrid">{categoryCards(remainingIntents)}</div>
                </section>
              ) : null}
            </>
          ) : null}

          {recentlyOpened.length > 0 || !isMobile ? (
            <section aria-labelledby="start-recent-title">
              <div className="wb-start-section-head">
                <b id="start-recent-title">
                  {t("workbench.start.continue", { defaultValue: "Recently opened" })}
                </b>
                <span>{t("workbench.start.recentMeta", { defaultValue: "This browser" })}</span>
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
              ) : recentlyOpened.length === 0 ? (
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
                  {recentlyOpened.map((workspace, index) => (
                    <Link
                      key={workspace.id}
                      to={workspacePath(workspace.id)}
                      className="wb-start-recent"
                      aria-label={t("workbench.start.openWorkspace", {
                        defaultValue: "Open workspace {{name}}",
                        name: workspace.name,
                      })}
                    >
                      <span className="wb-start-recent-art" data-tone={index % 3} aria-hidden>
                        <span className="wb-start-recent-preview">
                          <i />
                          <i />
                          <i />
                        </span>
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
          ) : null}
        </div>

        <StartPageRail
          catalog={catalog}
          coverage={coverage}
          review={review}
          workspaces={workspaces}
          summaries={summaries}
          curated={curated}
          continuation={isMobile ? undefined : continuation}
        />
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
