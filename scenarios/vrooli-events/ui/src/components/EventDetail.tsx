// DOC: docs/internal/EXPERIENCE-AUDIT.md
import { X, ExternalLink } from "lucide-react";
import { useNavigate } from "react-router-dom";
import type { EventEnvelope } from "../lib/api";
import { formatTimestamp, safeStringify } from "../lib/utils";
import { Panel } from "./Panel";

interface EventDetailProps {
  event: EventEnvelope;
  onClose: () => void;
}

export function EventDetail({ event, onClose }: EventDetailProps) {
  const navigate = useNavigate();

  return (
    <Panel variant="overlay" className="p-5">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-semibold text-white">Event Detail</h3>
        <button onClick={onClose} className="text-[var(--text-muted)] hover:text-white">
          <X className="h-4 w-4" />
        </button>
      </div>

      <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
        <div>
          <dt className="text-xs text-[var(--text-faint)]">Event ID</dt>
          <dd className="mt-0.5 font-mono text-xs text-[var(--text-secondary)]">{event.eventId}</dd>
        </div>
        <div>
          <dt className="text-xs text-[var(--text-faint)]">Type</dt>
          <dd className="mt-0.5 font-mono text-xs text-[var(--text-accent-light)]">{event.eventType}</dd>
        </div>
        <div>
          <dt className="text-xs text-[var(--text-faint)]">Source</dt>
          <dd className="mt-0.5 text-[var(--text-secondary)]">{event.sourceScenario}</dd>
        </div>
        <div>
          <dt className="text-xs text-[var(--text-faint)]">Target</dt>
          <dd className="mt-0.5 text-[var(--text-secondary)]">{event.targetScenario || "—"}</dd>
        </div>
        <div>
          <dt className="text-xs text-[var(--text-faint)]">Correlation ID</dt>
          <dd className="mt-0.5 font-mono text-xs text-[var(--text-secondary)]">
            {event.correlationId ? (
              <button
                onClick={() => navigate(`/traces?cid=${encodeURIComponent(event.correlationId!)}`)}
                className="inline-flex items-center gap-1 text-[var(--text-accent)] hover:underline"
              >
                {event.correlationId}
                <ExternalLink className="h-3 w-3" />
              </button>
            ) : (
              "—"
            )}
          </dd>
        </div>
        <div>
          <dt className="text-xs text-[var(--text-faint)]">Created</dt>
          <dd className="mt-0.5 text-[var(--text-secondary)]">
            {event.createdAt ? formatTimestamp(event.createdAt) : "—"}
          </dd>
        </div>
      </dl>

      {event.metadata && Object.keys(event.metadata).length > 0 && (
        <div className="mt-4">
          <p className="mb-2 text-xs font-medium text-[var(--text-muted)]">Metadata</p>
          <pre className="overflow-x-auto rounded-lg bg-[var(--surface-code)] p-3 font-mono text-xs text-[var(--text-secondary)]">
            {safeStringify(event.metadata)}
          </pre>
        </div>
      )}

      {event.payload !== undefined && event.payload !== null && (
        <div className="mt-4">
          <p className="mb-2 text-xs font-medium text-[var(--text-muted)]">Payload</p>
          <pre className="overflow-x-auto rounded-lg bg-[var(--surface-code)] p-3 font-mono text-xs text-[var(--text-secondary)]">
            {typeof event.payload === "string" ? event.payload : safeStringify(event.payload)}
          </pre>
        </div>
      )}
    </Panel>
  );
}
