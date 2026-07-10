/**
 * RoundDetailDialog
 *
 * Read-only focus surface for a single operating-mode round. Renders the
 * round's existing client-side fields — summary, error, items, handoffs,
 * agent profile, runId — without fetching extra state. Live agent-log
 * streaming is deliberately deferred (see PROBLEMS.md); the runId is shown
 * as a copyable mono token so users can correlate with agent-manager runs.
 */

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { selectors } from "../../../consts/selectors";
import { formatRelativeTime } from "../../../lib";
import type { OperatingModePhaseResolutionRecord, OperatingModeRound } from "../../../types/operating-mode";
import { phaseLabel, resolutionSummary, statusClasses } from "./utils";

export interface RoundDetailDialogProps {
  round: OperatingModeRound;
  isOpen: boolean;
  onClose: () => void;
}

export function RoundDetailDialog({ round, isOpen, onClose }: RoundDetailDialogProps) {
  const [copied, setCopied] = useState(false);
  const errorTone = round.status === "needs_attention"
    ? "border-amber-500/30 bg-amber-500/10 text-amber-200"
    : "border-red-500/30 bg-red-500/10 text-red-200";
  const errorLabelTone = round.status === "needs_attention" ? "text-amber-300" : "text-red-300";
  const errorLabel = round.status === "needs_attention" ? "Needs attention" : "Error";

  const handleCopyRunId = async () => {
    if (!round.runId) return;
    try {
      await navigator.clipboard.writeText(round.runId);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard rejected (insecure context, no permission). The runId
      // text is still selectable on the page.
    }
  };

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={`Round ${round.round} — ${phaseLabel(round.phase)}`}
      maxWidth="max-w-3xl"
      testId={selectors.initiativeDetails.roundDetailDialog}
    >
      <div className="space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className={`rounded-full border px-2 py-0.5 text-[11px] ${statusClasses(round.status)}`}>
            {phaseLabel(round.status)}
          </span>
          {round.generatedAt && (
            <span className="text-[11px] text-slate-500">
              {formatRelativeTime(round.generatedAt)}
            </span>
          )}
        </div>

        <DetailRow label="Agent profile" value={round.agentProfileKey} />

        {round.runId && (
          <div className="flex flex-col gap-1">
            <p className="text-[11px] uppercase tracking-wide text-slate-500">Run ID</p>
            <div className="flex flex-wrap items-center gap-2">
              <code className="break-all rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-200">
                {round.runId}
              </code>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleCopyRunId}
                data-testid={selectors.initiativeDetails.roundDetailRunIdCopy}
              >
                {copied ? (
                  <>
                    <Check className="mr-1.5 h-3.5 w-3.5 text-emerald-400" />
                    Copied
                  </>
                ) : (
                  <>
                    <Copy className="mr-1.5 h-3.5 w-3.5" />
                    Copy
                  </>
                )}
              </Button>
            </div>
          </div>
        )}

        <ResolutionBlock resolution={round.resolution} />

        {round.resolvedEnvelope && Object.keys(round.resolvedEnvelope).length > 0 && (
          <details className="rounded-md border border-slate-800 bg-slate-950/40 p-3">
            <summary className="cursor-pointer text-[11px] font-semibold uppercase tracking-wide text-slate-400">
              Resolved envelope
            </summary>
            <pre className="mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-words text-xs text-slate-300">
              {JSON.stringify(round.resolvedEnvelope, null, 2)}
            </pre>
          </details>
        )}

        {round.error && (
          <div className={`rounded-md border p-3 text-sm ${errorTone}`}>
            <p className={`text-[11px] font-semibold uppercase tracking-wide ${errorLabelTone}`}>{errorLabel}</p>
            <p className="mt-1 whitespace-pre-wrap text-sm">{round.error}</p>
          </div>
        )}

        {round.items && round.items.length > 0 && (
          <div>
            <p className="mb-1 text-[11px] uppercase tracking-wide text-slate-500">Items operated on</p>
            <ul className="space-y-1">
              {round.items.map((item, idx) => (
                <li
                  key={`${item.ref}-${idx}`}
                  className="flex flex-wrap items-center gap-2 rounded-md border border-slate-800 bg-slate-950/40 px-2 py-1.5 text-xs text-slate-300"
                >
                  <code className="break-all rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-200">
                    {item.ref}
                  </code>
                  {item.title && <span className="text-slate-400">{item.title}</span>}
                  {item.status && (
                    <span className="rounded-full border border-slate-700/80 bg-slate-900/60 px-2 py-0.5 text-[10px] text-slate-400">
                      {item.status}
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        {round.handoffs && round.handoffs.length > 0 && (
          <div>
            <p className="mb-1 text-[11px] uppercase tracking-wide text-slate-500">Handoffs</p>
            <div className="space-y-2">
              {round.handoffs.map((handoff, index) => (
                <div
                  key={`handoff-${index}`}
                  className="rounded-md border border-slate-800 bg-slate-950/40 p-2.5 text-sm"
                >
                  <p className="text-[11px] font-medium text-slate-300">Handoff {index + 1}</p>
                  {handoff.summary && (
                    <p className="mt-1 whitespace-pre-wrap text-sm text-slate-300">{handoff.summary}</p>
                  )}
                  {handoff.nextStep && (
                    <p className="mt-1 text-xs text-cyan-300">Next: {handoff.nextStep}</p>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="flex justify-end pt-1">
          <Button type="button" variant="outline" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </Dialog>
  );
}

function ResolutionBlock({ resolution }: { resolution?: OperatingModePhaseResolutionRecord }) {
  const summary = resolutionSummary(resolution);
  if (!summary || !resolution) return null;
  return (
    <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-amber-300">Resolution ladder</p>
      <div className="mt-2 grid gap-2 sm:grid-cols-2">
        <ResolutionField label="Outcome" value={summary} />
        {typeof resolution.messagesScanned === "number" && resolution.messagesScanned > 0 && (
          <ResolutionField
            label="Messages"
            value={`${resolution.messagesScanned} scanned${typeof resolution.chosenMessageIndex === "number" && resolution.chosenMessageIndex >= 0 ? `, chose index ${resolution.chosenMessageIndex}` : ""}`}
          />
        )}
      </div>
      <ResolutionList label="Missing" values={resolution.missing} />
      <ResolutionList label="Violations" values={resolution.violations} />
      <ResolutionList label="Notes" values={resolution.notes} />
      {resolution.selectedMessage && (
        <div className="mt-3 border-t border-amber-500/20 pt-2">
          <p className="text-[11px] font-semibold uppercase tracking-wide text-amber-300">Selected assistant event</p>
          <div className="mt-2 grid gap-2 sm:grid-cols-2">
            {resolution.selectedMessage.eventId && (
              <ResolutionField label="Event ID" value={resolution.selectedMessage.eventId} />
            )}
            {typeof resolution.selectedMessage.sequence === "number" && (
              <ResolutionField label="Sequence" value={String(resolution.selectedMessage.sequence)} />
            )}
            <ResolutionField label="Content digest" value={resolution.selectedMessage.contentDigest} />
            <ResolutionField label="Selection version" value={resolution.selectedMessage.selectionAlgorithmVersion} />
            {resolution.selectedMessage.fallbackReason && (
              <ResolutionField label="Fallback reason" value={resolution.selectedMessage.fallbackReason} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function ResolutionField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[11px] uppercase tracking-wide text-amber-300/80">{label}</p>
      <p className="mt-0.5 text-sm text-amber-100">{value}</p>
    </div>
  );
}

function ResolutionList({ label, values }: { label: string; values?: string[] }) {
  const present = values?.filter((value) => value.trim() !== "") ?? [];
  if (present.length === 0) return null;
  return (
    <div className="mt-2">
      <p className="text-[11px] uppercase tracking-wide text-amber-300/80">{label}</p>
      <p className="mt-0.5 whitespace-pre-wrap text-sm text-amber-100">{present.join(", ")}</p>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div className="flex flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-3">
      <p className="shrink-0 text-[11px] uppercase tracking-wide text-slate-500 sm:w-32">{label}</p>
      <code className="break-all rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-200">
        {value}
      </code>
    </div>
  );
}
