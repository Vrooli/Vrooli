import "../activity.css";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, ArrowUpRight, CircleAlert, Filter, Search } from "lucide-react";
import { NotificationBadge } from "@vrooli/react-component-library/NotificationBadge/1.0.5";
import { Timeline } from "@vrooli/react-component-library/Timeline/1.0.10";

import { getActivitySummary, listActivity, type ActivityEvent } from "../../api/activity";
import { useTranslation } from "../../i18n";
import { formatDay, formatNumber, formatTime } from "../data";
import { Banner, Empty, Kpi, Panel, PanelBody, PanelHead, Skeleton } from "../primitives";

/**
 * Activity is the append-only event log owned by ActivityService. It no longer
 * derives "who changed what" from catalog timestamps; an empty log is a
 * measured zero and an unreachable log is unavailable.
 */
export function ActivityPage({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useTranslation();
  const [action, setAction] = useState("all");
  const [term, setTerm] = useState("");

  const events = useQuery({
    queryKey: ["activity", "events"],
    queryFn: () => listActivity({ limit: 200 }),
    retry: 0,
  });
  const summary = useQuery({
    queryKey: ["activity", "summary", 7],
    queryFn: () => getActivitySummary(7),
    retry: 0,
  });

  const rows = useMemo(() => events.data ?? [], [events.data]);
  const actions = useMemo(() => [...new Set(rows.map((event) => event.action))].sort(), [rows]);
  const filtered = useMemo(() => {
    const needle = term.trim().toLowerCase();
    return rows.filter((event) => {
      if (action !== "all" && event.action !== action) return false;
      if (!needle) return true;
      return [event.actor, event.action, event.subjectId, event.summary]
        .join(" ")
        .toLowerCase()
        .includes(needle);
    });
  }, [rows, action, term]);

  const groups = useMemo(() => {
    const map = new Map<string, ActivityEvent[]>();
    for (const event of filtered) {
      const at = new Date(event.at);
      const day = Number.isNaN(at.getTime()) ? "Unknown day" : formatDay(at);
      map.set(day, [...(map.get(day) ?? []), event]);
    }
    return [...map.entries()];
  }, [filtered]);

  const week = summary.data;
  const actors = week?.byActor.length ?? 0;

  if (events.isLoading) {
    return (
      <div className={embedded ? "wb-page is-embedded" : "wb-page"} data-testid="activity-page">
        <Skeleton height={40} width={340} />
        <div style={{ height: 18 }} />
        <Skeleton height={90} />
        <div style={{ height: 18 }} />
        <Skeleton height={420} />
      </div>
    );
  }

  return (
    <div className={embedded ? "wb-page is-embedded" : "wb-page"} data-testid="activity-page">
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
                "Durable events written with the mutation they describe. Attribution is recorded, not inferred from a timestamp.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <label className="wb-search" style={{ minWidth: 190 }}>
            <Search aria-hidden size={14} />
            <input
              value={term}
              onChange={(event) => setTerm(event.target.value)}
              aria-label={t("workbench.activity.search", { defaultValue: "Search activity" })}
              placeholder={t("workbench.activity.searchPlaceholder", {
                defaultValue: "Actor, subject or action",
              })}
            />
          </label>
          <select
            className="wb-select"
            aria-label={t("workbench.activity.filter", { defaultValue: "Filter activity" })}
            value={action}
            onChange={(event) => setAction(event.target.value)}
            style={{ width: "auto", minWidth: 150 }}
          >
            <option value="all">
              {t("workbench.activity.allActions", { defaultValue: "All actions" })}
            </option>
            {actions.map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </select>
        </div>
      </div>

      {events.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
          <b>
            {t("workbench.activity.errorTitle", {
              defaultValue: "The activity service did not answer.",
            })}
          </b>{" "}
          {t("workbench.activity.errorBody", {
            defaultValue:
              "Activity shows unavailable rather than “no changes”. Retry when the API is reachable.",
          })}
        </Banner>
      ) : null}

      <div className="wb-kpis">
        <Kpi
          label={t("workbench.activity.kpiEvents", { defaultValue: "Last 7 days" })}
          value={week ? formatNumber(week.total) : "—"}
          detail={t("workbench.activity.kpiEventsDetail", {
            defaultValue: "Recorded events",
          })}
        />
        <Kpi
          label={t("workbench.activity.kpiActors", { defaultValue: "Actors" })}
          value={week ? formatNumber(actors) : "—"}
          tone="signal"
          detail={t("workbench.activity.kpiActorsDetail", {
            defaultValue: "Durable attribution",
          })}
        />
        <Kpi
          label={t("workbench.activity.kpiActions", { defaultValue: "Action kinds" })}
          value={week ? formatNumber(week.byAction.length) : "—"}
          tone="green"
          detail={t("workbench.activity.kpiActionsDetail", {
            defaultValue: "Distinct mutations",
          })}
        />
        <Kpi
          label={t("workbench.activity.kpiShown", { defaultValue: "Shown" })}
          value={formatNumber(filtered.length)}
          detail={t("workbench.activity.kpiShownDetail", {
            defaultValue: "Filtered from the latest {{count}}",
            count: rows.length,
          })}
        />
      </div>

      <div className="wb-activity-layout">
        <Panel>
          <PanelHead
            title={
              <NotificationBadge
                value={filtered.length}
                tone={filtered.length ? "info" : "neutral"}
                badgeLabel={`${filtered.length} events in this view`}
              >
                {t("workbench.activity.trail", { defaultValue: "Change trail" })}
              </NotificationBadge>
            }
            meta={t("workbench.activity.trailMeta", { count: filtered.length })}
            actions={
              <span className="wb-muted-2" aria-hidden>
                <Filter size={13} />
              </span>
            }
          />
          <PanelBody flush>
            {filtered.length === 0 ? (
              <Empty
                title={t("workbench.activity.emptyTitle", {
                  defaultValue: "No events in this view",
                })}
              >
                {events.isError
                  ? t("workbench.activity.emptyUnavailable", {
                      defaultValue: "The log is unreachable, so this is not an empty trail.",
                    })
                  : t("workbench.activity.emptyBody", {
                      defaultValue:
                        "The log answered with zero events for this filter. Workspace saves, forks and deletes write here.",
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
                        <span className="wb-event-icon is-published" aria-hidden>
                          <Activity size={13} />
                        </span>
                        <div className="wb-event-body">
                          <b>
                            {event.action} · {event.subjectId}
                          </b>
                          <small>
                            {event.summary || event.subjectKind} · {event.actor}
                          </small>
                          {event.beforeJson || event.afterJson ? (
                            <details>
                              <summary className="wb-muted-2 wb-small">
                                {t("workbench.activity.payload", { defaultValue: "Payload" })}
                              </summary>
                              <pre style={{ whiteSpace: "pre-wrap", margin: "6px 0" }}>
                                {event.afterJson || event.beforeJson}
                              </pre>
                            </details>
                          ) : null}
                        </div>
                        <span className="wb-event-time">{formatTime(new Date(event.at))}</span>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            )}
          </PanelBody>
        </Panel>

        <Panel>
          <PanelHead
            title={t("workbench.activity.recent", { defaultValue: "Recent events" })}
            meta={t("workbench.activity.latestCount", { defaultValue: "Latest 6" })}
          />
          <PanelBody>
            <Timeline
              state={events.isError ? "error" : "ready"}
              events={filtered.slice(0, 6).map((event) => ({
                label: `${event.action} · ${event.subjectId}`,
                detail: event.actor,
                date: formatTime(new Date(event.at)),
              }))}
            />
            <details style={{ marginTop: 16 }}>
              <summary>
                {t("workbench.activity.actors", { defaultValue: "Actors and actions" })}
              </summary>
              {week ? (
                <div className="wb-stack tight" style={{ marginTop: 10 }}>
                  {week.byActor.length === 0 ? (
                    <p className="wb-muted wb-small">
                      {t("workbench.activity.noActors", {
                        defaultValue: "No actors in this window.",
                      })}
                    </p>
                  ) : (
                    week.byActor.map((entry) => (
                      <div key={entry.actor} className="wb-row-between">
                        <span>{entry.actor}</span>
                        <span className="wb-tag">{entry.count}</span>
                      </div>
                    ))
                  )}
                  {week.byAction.map((entry) => (
                    <div key={entry.action} className="wb-row-between">
                      <span className="wb-mono wb-small">{entry.action}</span>
                      <span className="wb-tag">{entry.count}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="wb-muted wb-small" style={{ marginTop: 10 }}>
                  <ArrowUpRight aria-hidden size={12} />{" "}
                  {t("workbench.activity.summaryUnavailable", {
                    defaultValue: "The summary is unavailable.",
                  })}
                </p>
              )}
            </details>
          </PanelBody>
        </Panel>
      </div>
    </div>
  );
}
