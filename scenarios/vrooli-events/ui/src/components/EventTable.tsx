// DOC: docs/internal/COHERENCE-NOTES.md
import type { EventEnvelope } from "../lib/api";
import { truncate } from "../lib/utils";

interface EventTableProps {
  events: EventEnvelope[];
  onSelect?: (event: EventEnvelope) => void;
}

export function EventTable({ events, onSelect }: EventTableProps) {
  if (events.length === 0) {
    return (
      <div className="rounded-xl border border-[var(--border-default)] bg-[var(--surface-elevated)] p-8 text-center text-sm text-[var(--text-muted)]">
        No events found. Events will appear here as they are ingested.
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-xl border border-[var(--border-default)]">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-[var(--border-default)] bg-[var(--surface-elevated)]">
            <th className="px-4 py-3 font-medium text-[var(--text-muted)]">Event Type</th>
            <th className="px-4 py-3 font-medium text-[var(--text-muted)]">Source</th>
            <th className="px-4 py-3 font-medium text-[var(--text-muted)]">Target</th>
            <th className="px-4 py-3 font-medium text-[var(--text-muted)]">Correlation</th>
            <th className="px-4 py-3 font-medium text-[var(--text-muted)]">Event ID</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[var(--border-subtle)]">
          {events.map((evt) => (
            <tr
              key={evt.eventId}
              onClick={() => onSelect?.(evt)}
              className="cursor-pointer transition-colors hover:bg-[var(--surface-elevated)]"
            >
              <td className="px-4 py-2.5 font-mono text-xs text-[var(--text-accent-light)]">
                {truncate(evt.eventType, 40)}
              </td>
              <td className="px-4 py-2.5 text-[var(--text-secondary)]">{evt.sourceScenario}</td>
              <td className="px-4 py-2.5 text-[var(--text-muted)]">{evt.targetScenario || "—"}</td>
              <td className="px-4 py-2.5 font-mono text-xs text-[var(--text-faint)]">
                {evt.correlationId ? truncate(evt.correlationId, 16) : "—"}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-[var(--text-faint)]">
                {truncate(evt.eventId, 16)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
