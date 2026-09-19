import type { Allocation } from "../api/calendar";

function clock(minutes: number) { return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`; }
function readableDate(date: string) { return new Intl.DateTimeFormat(undefined, { weekday: "short", month: "short", day: "numeric" }).format(new Date(`${date}T12:00:00`)); }

export function RangeView({ view, query, loading, error }: { view: "week" | "agenda"; query: readonly Allocation[]; loading: boolean; error: boolean }) {
  if (loading) return <p className="planner-range-status" role="status">Loading the accepted range…</p>;
  if (error) return <p className="planner-range-status" role="alert">The accepted range is unavailable right now.</p>;
  if (query.length === 0) return <p className="planner-empty">No accepted placements in this range yet. Day placement stays explicit.</p>;
  const entries = view === "week" ? query.reduce<Record<string, Allocation[]>>((groups, entry) => { (groups[entry.localDate] ??= []).push(entry); return groups; }, {}) : { agenda: [...query] };
  return <div className={`plan-range plan-range-${view}`} aria-label={view === "week" ? "Accepted week" : "Accepted agenda"}>{Object.entries(entries).map(([date, allocations]) => <section key={date}><h2>{date === "agenda" ? "Upcoming accepted work" : readableDate(date)}</h2>{allocations.map((entry) => <article className="plan-range-item" key={entry.id}><span>{date !== "agenda" && `${clock(Number(entry.startMinutes))} · `}{Number(entry.durationMinutes)} min</span><strong>{entry.title}</strong><small>{entry.sourceLabel || "Personal Planner"}</small></article>)}</section>)}</div>;
}
