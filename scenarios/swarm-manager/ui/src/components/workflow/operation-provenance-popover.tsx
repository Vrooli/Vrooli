/**
 * OperationProvenancePopover
 *
 * Operator inspectability affordance for one canonical operation record: a
 * small fingerprint trigger that opens an anchored popover listing the exact
 * provenance the runtime pinned — operation contract id@version, mode id +
 * exact revision, binding source (layer + owner), run/execution ids,
 * attempt/retry linkage, and (for executions) the verified-evidence
 * (reproducible) flag. All data comes from the workflow projection or
 * execution-history queries — nothing is computed client-side.
 */

import { useRef, useState } from "react";
import { Fingerprint, ShieldAlert, ShieldCheck } from "lucide-react";
import { Popover } from "../ui/popover";
import { cn } from "../../lib/utils";
import { bindingSourceLabel, shortDigest } from "../../lib/agent-ops-utils";
import type { OperationProvenanceData } from "../../lib/agent-ops-utils";
import { formatRelativeTime } from "../../lib";

export interface OperationProvenancePopoverProps {
  data: OperationProvenanceData;
  className?: string;
  /** Accessible label for the trigger; defaults to "Operation provenance". */
  label?: string;
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <span className="shrink-0 text-[11px] font-medium uppercase tracking-wide text-slate-500">
        {label}
      </span>
      <span className="min-w-0 break-all text-right font-mono text-[11px] text-slate-300">
        {children}
      </span>
    </div>
  );
}

export function OperationProvenancePopover({
  data,
  className,
  label = "Operation provenance",
}: OperationProvenancePopoverProps) {
  const [isOpen, setIsOpen] = useState(false);
  // A span[role=button] rather than <button>: the affordance is embedded in
  // row headers that are themselves buttons, and nested buttons are invalid.
  const triggerRef = useRef<HTMLSpanElement>(null);

  const contractIdentity = data.operationVersion
    ? `${data.operation}@${data.operationVersion}`
    : data.operation;
  const modeIdentity = data.mode
    ? `${data.mode}${data.modeRevision ? ` @ ${shortDigest(data.modeRevision)}` : ""}`
    : "";

  return (
    <>
      <span
        ref={triggerRef}
        role="button"
        tabIndex={0}
        onClick={(event) => {
          event.stopPropagation();
          event.preventDefault();
          setIsOpen((open) => !open);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.stopPropagation();
            event.preventDefault();
            setIsOpen((open) => !open);
          }
        }}
        className={cn(
          "inline-flex cursor-pointer items-center gap-1 rounded border border-slate-700/80 bg-slate-800/60 px-1.5 py-0.5 text-[10px] font-medium text-slate-400 transition-colors hover:border-cyan-500/50 hover:text-cyan-300",
          className,
        )}
        title={label}
        aria-label={label}
        data-testid="operation-provenance-trigger"
      >
        <Fingerprint className="h-3 w-3" aria-hidden />
        {data.source}
      </span>
      <Popover
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        triggerRef={triggerRef}
        className="w-80 p-3"
        testId="operation-provenance-popover"
      >
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-2">
            <p className="text-xs font-semibold text-slate-200">Operation provenance</p>
            {data.reproducible !== undefined && (
              <span
                className={cn(
                  "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium",
                  data.reproducible
                    ? "bg-emerald-500/15 text-emerald-400"
                    : "bg-amber-500/15 text-amber-300",
                )}
                data-testid="operation-provenance-reproducible"
              >
                {data.reproducible ? (
                  <ShieldCheck className="h-3 w-3" aria-hidden />
                ) : (
                  <ShieldAlert className="h-3 w-3" aria-hidden />
                )}
                {data.reproducible ? "Verified evidence" : "Digest drift"}
              </span>
            )}
          </div>
          <div className="space-y-1.5 rounded-md border border-slate-800 bg-slate-950/50 p-2">
            <Row label="Contract">{contractIdentity}</Row>
            {modeIdentity && <Row label="Mode">{modeIdentity}</Row>}
            <Row label="Binding">{bindingSourceLabel(data)}</Row>
            {data.executionId && <Row label="Execution">{data.executionId}</Row>}
            {data.runId && <Row label="Run">{data.runId}</Row>}
            {data.attempt !== undefined && data.attempt > 0 && (
              <Row label="Attempt">
                {data.attempt}
                {data.priorExecutionId ? ` (retry of ${data.priorExecutionId})` : ""}
              </Row>
            )}
            {data.state && <Row label="State">{data.state}</Row>}
            {data.outcome && <Row label="Outcome">{data.outcome}</Row>}
            {data.provenanceDigest && (
              <Row label="Provenance">{shortDigest(data.provenanceDigest)}</Row>
            )}
            {data.compiledModeDigest && (
              <Row label="Mode digest">{shortDigest(data.compiledModeDigest)}</Row>
            )}
            {data.promptCatalogDigest && (
              <Row label="Prompt catalog">{shortDigest(data.promptCatalogDigest)}</Row>
            )}
            {data.callerInputDigest && (
              <Row label="Caller input">{shortDigest(data.callerInputDigest)}</Row>
            )}
            {data.recordedAt && (
              <Row label="Recorded">{formatRelativeTime(data.recordedAt)}</Row>
            )}
          </div>
          {data.snapshotFound === false && (
            <p className="text-[11px] text-amber-300">
              Execution snapshot missing on disk — provenance fields limited to the workflow
              record.
            </p>
          )}
        </div>
      </Popover>
    </>
  );
}
