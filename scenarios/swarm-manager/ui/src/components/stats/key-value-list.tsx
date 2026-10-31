import { cn } from "../../lib/utils";

interface KeyValueListProps {
  entries: [string, number][];
  formatKey?: (value: string) => string;
  className?: string;
}

export function KeyValueList({ entries, formatKey = (value) => value, className }: KeyValueListProps) {
  return (
    <ul className={cn("space-y-1", className)}>
      {entries.map(([key, value]) => (
        <li key={key} className="flex items-center justify-between rounded px-2 py-1 text-sm hover:bg-slate-800/50">
          <span className="truncate text-slate-300">{formatKey(key)}</span>
          <span className="ml-2 shrink-0 rounded bg-slate-700/60 px-1.5 py-0.5 text-xs text-slate-400">
            {value.toLocaleString()}
          </span>
        </li>
      ))}
    </ul>
  );
}
