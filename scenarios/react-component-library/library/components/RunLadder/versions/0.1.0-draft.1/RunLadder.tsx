/**
 * @libraryId react-component-library:RunLadder
 * @displayName Run Ladder
 * @description A phased, resumable long-running operation surface with item outcomes.
 * @version 0.1.0-draft.1
 * @tags ["data-display","operations"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { ProgressLadder } from "@vrooli/react-component-library/ProgressLadder/1";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";

export interface RunPhase {
  id: string;
  label: string;
  state: "pending" | "active" | "complete" | "failed";
  detail?: string;
  durationMs?: number;
}
export interface RunItem {
  name: string;
  outcome: "pending" | "running" | "succeeded" | "failed" | "skipped";
  error?: string;
}
export interface RunLadderProps {
  runId?: string;
  phases: RunPhase[];
  items?: RunItem[];
  reconnecting?: boolean;
  resumeNote?: string;
}

const tones: Record<RunItem["outcome"], StatusTone> = {
  pending: "neutral",
  running: "info",
  succeeded: "success",
  failed: "danger",
  skipped: "warning",
};

export function RunLadder({
  runId,
  phases,
  items = [],
  reconnecting = false,
  resumeNote,
}: RunLadderProps) {
  const busy = phases.some((phase) => phase.state === "active");
  return (
    <section aria-live="polite" aria-busy={busy} data-testid="run-ladder">
      {reconnecting && (
        <p role="status" data-testid="run-reconnecting">
          Connection dropped. Retrying; current progress is preserved.
        </p>
      )}
      <div className="run-ladder-heading">
        <strong>Run</strong>
        {runId && <code data-testid="run-id">{runId}</code>}
        {resumeNote && <span>{resumeNote}</span>}
      </div>
      <ProgressLadder
        rungs={phases.map((phase) => ({
          ...phase,
          complete: phase.state === "complete",
          current: phase.state === "active",
        }))}
      />
      {items.length > 0 && (
        <ul data-testid="run-items" className="run-ladder-items">
          {items.map((item) => (
            <li key={item.name}>
              <code>{item.name}</code>
              <StatusBadge tone={tones[item.outcome]}>{item.outcome}</StatusBadge>
              {item.error && <p>{item.error}</p>}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
