import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Activity, ArrowUpRight, Boxes, CircleAlert, Filter } from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import {
  activityFromCatalog,
  formatDay,
  formatTime,
  useCatalog,
  type ActivityEvent,
  type ActivityKind,
} from "../data";
import { Banner, Empty, Kpi, Panel, PanelBody, PanelHead, Select, Skeleton } from "../primitives";

const KINDS: Array<{ id: ActivityKind | "all"; label: string }> = [
  { id: "all", label: "All activity" },
  { id: "draft", label: "Drafts" },
  { id: "indexed", label: "Index" },
];

export function ActivityPage() {
  const { t } = useTranslation();
  const catalog = useCatalog();
  const [kind, setKind] = useState<ActivityKind | "all">("all");

  const events = useMemo(() => activityFromCatalog(catalog.data ?? []), [catalog.data]);
  const filtered = useMemo(
    () => (kind === "all" ? events : events.filter((event) => event.kind === kind)),
    [events, kind],
  );

  const visible = useMemo(() => filtered.slice(0, 200), [filtered]);

  const groups = useMemo(() => {
    const map = new Map<string, ActivityEvent[]>();
    for (const event of visible) {
      const day = formatDay(event.at);
      map.set(day, [...(map.get(day) ?? []), event]);
    }
    return [...map.entries()];
  }, [visible]);

  const week = useMemo(() => {
    const cutoff = Date.now() - 7 * 86_400_000;
    const recent = events.filter((event) => event.at.getTime() >= cutoff);
    return {
      total: recent.length,
      drafts: recent.filter((event) => event.kind === "draft").length,
      index: recent.filter((event) => event.kind === "indexed").length,
    };
  }, [events]);

  const actors = useMemo(() => {
    const map = new Map<string, number>();
    for (const event of filtered) map.set(event.actor, (map.get(event.actor) ?? 0) + 1);
    return [...map.entries()].sort(([, left], [, right]) => right - left);
  }, [filtered]);

  return (
    <div className="wb-page" data-testid="activity-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">
            {t("workbench.activity.kicker", { defaultValue: "Library · Activity" })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.activity.title", { defaultValue: "What changed, and who changed it" })}
          </h1>
          <p className="wb-lede">
            {t("workbench.activity.lede", {
              defaultValue:
                "Releases, drafts and index events behind the current state. This is catalog index activity, not a durable event store — the distinction is shown, not hidden.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <Select
            aria-label={t("workbench.activity.filter", { defaultValue: "Filter activity" })}
            value={kind}
            onChange={(event) => setKind(event.target.value as ActivityKind | "all")}
            style={{ width: "auto", minWidth: 150 }}
          >
            {KINDS.map((entry) => (
              <option key={entry.id} value={entry.id}>
                {entry.label}
              </option>
            ))}
          </Select>
        </div>
      </div>

      {catalog.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
          <b>
            {t("workbench.activity.errorTitle", { defaultValue: "The catalog did not answer." })}
          </b>{" "}
          {t("workbench.activity.errorBody", {
            defaultValue:
              "Activity cannot be derived without the catalog. Nothing is shown as “no changes”.",
          })}
        </Banner>
      ) : null}

      <div className="wb-kpis">
        <Kpi
          label={t("workbench.activity.kpiEvents", { defaultValue: "This week" })}
          value={week.total}
          detail={t("workbench.activity.kpiEventsDetail", { defaultValue: "Index-derived events" })}
        />
        <Kpi
          label={t("workbench.activity.kpiDrafts", { defaultValue: "Drafts" })}
          value={week.drafts}
          tone="amber"
          detail={t("workbench.activity.kpiDraftsDetail", { defaultValue: "Open drafts" })}
        />
        <Kpi
          label={t("workbench.activity.kpiIndex", { defaultValue: "Index events" })}
          value={week.index}
          tone="green"
          detail={t("workbench.activity.kpiIndexDetail", {
            defaultValue: "Source updated or re-indexed",
          })}
        />
        <Kpi
          label={t("workbench.activity.kpiEventsTotal", { defaultValue: "Events" })}
          value={week.total}
          tone="signal"
          detail={t("workbench.activity.kpiEventsTotalDetail", {
            defaultValue: "In the last 7 days",
          })}
        />
      </div>

      <div className="wb-activity-layout">
        <Panel>
          <PanelHead
            title={t("workbench.activity.trail", { defaultValue: "Change trail" })}
            meta={t("workbench.activity.trailMeta", {
              count: filtered.length,
            })}
            actions={
              <span className="wb-muted-2" aria-hidden>
                <Filter size={13} />
              </span>
            }
          />
          <PanelBody flush>
            {catalog.isLoading ? (
              <div style={{ padding: 16 }}>
                <Skeleton height={180} />
              </div>
            ) : filtered.length === 0 ? (
              <Empty
                title={t("workbench.activity.emptyTitle", {
                  defaultValue: "No events in this view",
                })}
              >
                {t("workbench.activity.emptyBody", {
                  defaultValue:
                    "Change the filter. The catalog answered; this view is simply empty.",
                })}
              </Empty>
            ) : (
              <div className="wb-trail">
                {groups.map(([day, dayEvents]) => (
                  <div key={day}>
                    <div className="wb-day">
                      {day} · {dayEvents.length}
                    </div>
                    {dayEvents.map((event) => (
                      <div key={event.id} className="wb-event">
                        <span className={`wb-event-icon is-${event.kind}`} aria-hidden>
                          {event.kind === "published" ? (
                            <Activity size={13} />
                          ) : (
                            <Boxes size={13} />
                          )}
                        </span>
                        <div className="wb-event-body">
                          <b>
                            {event.title} · {event.assetName}{" "}
                            <span className="wb-muted-2">{event.version}</span>
                          </b>
                          <small>
                            {event.detail} · {event.actor}
                          </small>
                        </div>
                        <span className="wb-event-time">{formatTime(event.at)}</span>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            )}
            {filtered.length > visible.length ? (
              <p className="wb-muted-2 wb-small" style={{ padding: "0 16px 12px" }}>
                {t("workbench.activity.truncated", {
                  defaultValue: "Showing the 200 most recent events of {{count}}.",
                  count: filtered.length,
                })}
              </p>
            ) : null}
          </PanelBody>
        </Panel>

        <aside
          className="wb-stack"
          aria-label={t("workbench.activity.details", { defaultValue: "Activity details" })}
        >
          <Panel>
            <PanelHead
              title={t("workbench.activity.week", { defaultValue: "This week" })}
              meta="7 days"
            />
            <PanelBody>
              <div className="wb-week-grid">
                <div className="wb-week-cell">
                  <b>{week.drafts}</b>
                  <span>{t("workbench.activity.drafts", { defaultValue: "Drafts" })}</span>
                </div>
                <div className="wb-week-cell">
                  <b>{week.index}</b>
                  <span>{t("workbench.activity.indexed", { defaultValue: "Indexed" })}</span>
                </div>
                <div className="wb-week-cell">
                  <b>{week.total}</b>
                  <span>{t("workbench.activity.events", { defaultValue: "Events" })}</span>
                </div>
                <div className="wb-week-cell">
                  <b>{week.total}</b>
                  <span>{t("workbench.activity.window", { defaultValue: "7 days" })}</span>
                </div>
              </div>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead
              title={t("workbench.activity.actors", { defaultValue: "Actors" })}
              meta={`${actors.length}`}
            />
            <PanelBody>
              {actors.map(([actor, count]) => (
                <div key={actor} className="wb-actor">
                  <span
                    className="wb-identity-mark"
                    style={{ width: 24, height: 24, fontSize: 9 }}
                    aria-hidden
                  >
                    {actor.slice(0, 2).toUpperCase()}
                  </span>
                  <div className="wb-actor-bar">
                    <div className="wb-row-between">
                      <span className="wb-small">{actor}</span>
                      <span className="wb-mono wb-small wb-muted-2">{count}</span>
                    </div>
                    <span style={{ width: `${(count / Math.max(1, filtered.length)) * 100}%` }} />
                  </div>
                </div>
              ))}
              <p className="wb-muted-2 wb-small" style={{ marginTop: 10 }}>
                {t("workbench.activity.actorNote", {
                  defaultValue:
                    "Actor is the catalog process today. Human attribution arrives with the durable event projection.",
                })}
              </p>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead title={t("workbench.activity.related", { defaultValue: "Related" })} />
            <PanelBody>
              <div className="wb-stack tight">
                <Link to={appRoutes.review} className="wb-row-between wb-small">
                  {t("workbench.activity.openReview", { defaultValue: "Review queue" })}
                  <ArrowUpRight aria-hidden size={13} />
                </Link>
                <Link to={appRoutes.coverage} className="wb-row-between wb-small">
                  {t("workbench.activity.openCoverage", { defaultValue: "Coverage" })}
                  <ArrowUpRight aria-hidden size={13} />
                </Link>
                <Link to={appRoutes.components} className="wb-row-between wb-small">
                  {t("workbench.activity.openComponents", { defaultValue: "Components" })}
                  <ArrowUpRight aria-hidden size={13} />
                </Link>
              </div>
            </PanelBody>
          </Panel>
        </aside>
      </div>
    </div>
  );
}
