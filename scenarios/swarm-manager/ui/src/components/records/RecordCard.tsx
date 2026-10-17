/**
 * RecordCard — compact representation of a record in a list or chain view.
 */

import { Link } from "react-router-dom";
import type { RecordItem } from "../../types";

interface RecordCardProps {
  record: RecordItem;
  score?: number;
  highlight?: boolean;
}

const KIND_COLORS: Record<string, string> = {
  fix: "bg-red-900/40 text-red-200",
  execute: "bg-blue-900/40 text-blue-200",
  research: "bg-purple-900/40 text-purple-200",
  chore: "bg-slate-700 text-slate-200",
  idea: "bg-amber-900/40 text-amber-200",
};

export function RecordCard({ record, score, highlight }: RecordCardProps) {
  const kindClass = KIND_COLORS[record.kind] ?? "bg-slate-700 text-slate-200";
  return (
    <Link
      to={`/records/${record.id}`}
      className={`block rounded border bg-slate-900/60 p-3 transition hover:bg-slate-900 ${
        highlight ? "border-emerald-500" : "border-slate-700"
      }`}
      data-testid={`record-card-${record.id}`}
    >
      <div className="flex items-center gap-2">
        <span className={`rounded px-2 py-0.5 text-xs ${kindClass}`}>{record.kind}</span>
        <span className="text-xs text-slate-400">{record.scenario}</span>
        {record.stub ? (
          <span className="rounded bg-amber-900/60 px-2 py-0.5 text-xs text-amber-200">stub</span>
        ) : null}
        {record.supersededBy ? (
          <span className="rounded bg-slate-800 px-2 py-0.5 text-xs text-slate-400">superseded</span>
        ) : null}
        {typeof score === "number" ? (
          <span className="ml-auto text-xs text-slate-500">score {score.toFixed(2)}</span>
        ) : null}
      </div>
      <div className="mt-2 text-sm text-slate-100">
        {record.trigger || <span className="italic text-slate-500">no trigger</span>}
      </div>
      {record.backlogRef ? (
        <div className="mt-1 text-xs text-slate-400">backlog: {record.backlogRef}</div>
      ) : null}
    </Link>
  );
}
