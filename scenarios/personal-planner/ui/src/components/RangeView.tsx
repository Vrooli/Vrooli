import type { Allocation } from "../api/calendar";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";

function clock(minutes: number) { return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`; }
function readableDate(date: string) { return new Intl.DateTimeFormat(undefined, { weekday: "short", month: "short", day: "numeric" }).format(new Date(`${date}T12:00:00`)); }
function longDate(date: string) { return new Intl.DateTimeFormat(undefined, { weekday: "long", month: "long", day: "numeric" }).format(new Date(`${date}T12:00:00`)); }
function monthDays() {
  const today = new Date();
  const first = new Date(today.getFullYear(), today.getMonth(), 1);
  const last = new Date(today.getFullYear(), today.getMonth() + 1, 0);
  return Array.from({ length: last.getDate() }, (_, index) => {
    const date = new Date(first.getFullYear(), first.getMonth(), index + 1);
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  });
}

export function RangeView({ view, query, loading, error }: { view: "week" | "agenda" | "month" | "timeline"; query: readonly Allocation[]; loading: boolean; error: boolean }) {
  if (loading) return <p className="planner-range-status" role="status">Loading the accepted range…</p>;
  if (error) return <p className="planner-range-status" role="alert">The accepted range is unavailable right now.</p>;
  if (query.length === 0) return <EmptyState className="planner-empty-state" title="No accepted placements in this range yet. Day placement stays explicit." />;
  const entries = query.reduce<Record<string, Allocation[]>>((groups, entry) => { (groups[entry.localDate] ??= []).push(entry); return groups; }, {});
  if (view === "month") return <div className="plan-range plan-range-month" aria-label="Accepted month">{monthDays().map((date) => <section key={date} className={entries[date]?.length ? "has-allocations" : ""}><h2>{readableDate(date)}</h2>{entries[date]?.length ? entries[date].map((entry) => <article className="plan-range-item" key={entry.id}><span>{clock(Number(entry.startMinutes))} · {Number(entry.durationMinutes)} min</span><strong>{entry.title}</strong></article>) : <small>Clear</small>}</section>)}</div>;
  if (view === "timeline") return <div className="plan-range plan-range-timeline" aria-label="Accepted timeline">{Object.entries(entries).flatMap(([date, allocations]) => allocations.map((entry) => ({ date, entry }))).sort((left, right) => left.date.localeCompare(right.date) || Number(left.entry.startMinutes) - Number(right.entry.startMinutes)).map(({ date, entry }) => <article className="plan-range-item" key={entry.id}><span>{longDate(date)} · {clock(Number(entry.startMinutes))}</span><strong>{entry.title}</strong><small>{Number(entry.durationMinutes)} min · {entry.sourceLabel || "Personal Planner"}</small></article>)}</div>;
  return <div className={`plan-range plan-range-${view}`} aria-label={view === "week" ? "Accepted week" : "Accepted agenda"}>{(view === "week" ? Object.entries(entries) : [["agenda", Object.values(entries).flat()] as [string, Allocation[]]]).map(([date, allocations]) => <section key={date}><h2>{date === "agenda" ? "Upcoming accepted work" : readableDate(date)}</h2>{allocations.map((entry) => <article className="plan-range-item" key={entry.id}><span>{date !== "agenda" && `${clock(Number(entry.startMinutes))} · `}{Number(entry.durationMinutes)} min</span><strong>{entry.title}</strong><small>{entry.sourceLabel || "Personal Planner"}</small></article>)}</section>)}</div>;
}
