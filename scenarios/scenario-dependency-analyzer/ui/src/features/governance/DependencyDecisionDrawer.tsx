import { useEffect, useMemo, useState } from "react";
import { ShieldCheck, ShieldX, X } from "lucide-react";

import type { DependencyUsageGroup, UpsertApprovedDependencyResponse, VulnerabilityRemediationResponse } from "../../api/governance";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import { governanceRecordFromDecision } from "./governanceTypes";

type DecisionState = "approved" | "denied" | "deprecated" | "needs_review";

export function DependencyDecisionDrawer({
  group,
  initialState,
  open,
  preview,
  applied,
  busy,
  error,
  onClose,
  onPreview,
  onApply
}: {
  group: DependencyUsageGroup | null;
  initialState: DecisionState;
  open: boolean;
  preview: UpsertApprovedDependencyResponse | null;
  applied: UpsertApprovedDependencyResponse | null;
  busy: boolean;
  error: string | null;
  onClose: () => void;
  onPreview: (record: ReturnType<typeof governanceRecordFromDecision>) => void;
  onApply: (record: ReturnType<typeof governanceRecordFromDecision>) => void;
}) {
  const [state, setState] = useState<DecisionState>(initialState);
  const [versionRange, setVersionRange] = useState("*");
  const [rationale, setRationale] = useState("");
  const [approvedBy, setApprovedBy] = useState("operator");
  const [replacement, setReplacement] = useState("");

  useEffect(() => {
    setState(initialState);
  }, [initialState]);

  const record = useMemo(() => {
    if (!group) return null;
    return governanceRecordFromDecision({
      ecosystem: group.ecosystem,
      packageName: group.packageName,
      state,
      versionRange,
      rationale,
      approvedBy,
      replacement
    });
  }, [approvedBy, group, rationale, replacement, state, versionRange]);

  if (!open || !group) return null;

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/45 p-2 sm:p-4" role="dialog" aria-modal="true" aria-label="Dependency decision">
      <div className="flex h-full w-full max-w-2xl flex-col overflow-hidden rounded-lg border border-border/60 bg-card shadow-2xl">
        <div className="flex items-start justify-between gap-3 border-b border-border/50 p-4">
          <div>
            <p className="text-xs uppercase text-muted-foreground">{group.ecosystem}</p>
            <h2 className="break-all text-lg font-semibold">{group.packageName}</h2>
          </div>
          <Button size="icon" variant="ghost" onClick={onClose} aria-label="Close dependency decision">
            <X className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>

        <div className="flex-1 overflow-auto p-4">
          <form
            className="grid gap-4"
            data-testid={selectors.governance.decisionForm}
            onSubmit={(event) => {
              event.preventDefault();
              if (record) onPreview(record);
            }}
          >
            <div className="grid gap-2 sm:grid-cols-2">
              <label className={state === "approved" ? selectedChoice : choice}>
                <input className="sr-only" checked={state === "approved"} onChange={() => setState("approved")} type="radio" />
                <ShieldCheck className="h-4 w-4" aria-hidden="true" />
                Approve
              </label>
              <label className={state === "denied" ? selectedChoice : choice}>
                <input className="sr-only" checked={state === "denied"} onChange={() => setState("denied")} type="radio" />
                <ShieldX className="h-4 w-4" aria-hidden="true" />
                Deny
              </label>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="governance-version-range">Version range</Label>
                <Input id="governance-version-range" value={versionRange} onChange={(event) => setVersionRange(event.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="governance-approved-by">Reviewer</Label>
                <Input id="governance-approved-by" value={approvedBy} onChange={(event) => setApprovedBy(event.target.value)} />
              </div>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="governance-rationale">Rationale</Label>
              <Input id="governance-rationale" value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder="Why this decision is correct" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="governance-replacement">Replacement</Label>
              <Input id="governance-replacement" value={replacement} onChange={(event) => setReplacement(event.target.value)} placeholder="Required for denied or deprecated dependencies when known" />
            </div>
            <div className="flex flex-wrap gap-2">
              <Button disabled={busy || !rationale.trim()} type="submit">Dry-run preview</Button>
              <Button disabled={busy || !preview?.record || !rationale.trim()} onClick={() => record && onApply(record)} type="button" variant="secondary">
                Apply decision
              </Button>
            </div>
          </form>

          <MutationFeedback preview={preview} applied={applied} error={error} />
        </div>
      </div>
    </div>
  );
}

export function VulnerabilityRemediationDrawer({
  group,
  initialVulnerabilityId,
  open,
  preview,
  result,
  busy,
  error,
  onClose,
  onPreview,
  onApply
}: {
  group: DependencyUsageGroup | null;
  initialVulnerabilityId?: string;
  open: boolean;
  preview: VulnerabilityRemediationResponse | null;
  result: VulnerabilityRemediationResponse | null;
  busy: boolean;
  error: string | null;
  onClose: () => void;
  onPreview: (vulnerabilityId: string) => void;
  onApply: (input: { vulnerabilityId: string; affectedRange: string; fixedRange: string; rationale: string; approvedBy: string }) => void;
}) {
  const [vulnerabilityId, setVulnerabilityId] = useState("");
  const [rationale, setRationale] = useState("");
  const [approvedBy, setApprovedBy] = useState("operator");
  const affectedRange = preview?.suggestedRecord?.versionRange || preview?.vulnerability?.affectedRanges[0]?.range || preview?.vulnerability?.affectedRanges[0]?.fixed || "";
  const fixedRange = preview?.vulnerability?.fixedRanges[0]?.range || preview?.vulnerability?.fixedRanges[0]?.fixed || "";

  useEffect(() => {
    setVulnerabilityId(initialVulnerabilityId ?? "");
  }, [initialVulnerabilityId, open]);

  if (!open || !group) return null;

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/45 p-2 sm:p-4" role="dialog" aria-modal="true" aria-label="Vulnerability remediation">
      <div className="flex h-full w-full max-w-2xl flex-col overflow-hidden rounded-lg border border-border/60 bg-card shadow-2xl">
        <div className="flex items-start justify-between gap-3 border-b border-border/50 p-4">
          <div>
            <p className="text-xs uppercase text-muted-foreground">Security-derived decision</p>
            <h2 className="break-all text-lg font-semibold">{group.ecosystem}/{group.packageName}</h2>
          </div>
          <Button size="icon" variant="ghost" onClick={onClose} aria-label="Close vulnerability remediation">
            <X className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
        <div className="flex-1 overflow-auto p-4">
          <form
            className="grid gap-4"
            data-testid={selectors.governance.remediationForm}
            onSubmit={(event) => {
              event.preventDefault();
              onPreview(vulnerabilityId);
            }}
          >
            <div className="grid gap-2">
              <Label htmlFor="governance-vulnerability-id">Vulnerability ID</Label>
              <Input id="governance-vulnerability-id" value={vulnerabilityId} onChange={(event) => setVulnerabilityId(event.target.value)} placeholder="GHSA, CVE, GO, or OSV id" />
            </div>
            <div className="flex flex-wrap gap-2">
              <Button disabled={busy || !vulnerabilityId.trim()} type="submit">Preview remediation</Button>
              <Button
                disabled={busy || !preview?.found || !rationale.trim()}
                onClick={() => onApply({ vulnerabilityId, affectedRange, fixedRange, rationale, approvedBy })}
                type="button"
                variant="secondary"
              >
                Apply denied range
              </Button>
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="governance-remediation-rationale">Rationale</Label>
                <Input id="governance-remediation-rationale" value={rationale} onChange={(event) => setRationale(event.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="governance-remediation-reviewer">Reviewer</Label>
                <Input id="governance-remediation-reviewer" value={approvedBy} onChange={(event) => setApprovedBy(event.target.value)} />
              </div>
            </div>
          </form>

          {preview?.found ? (
            <div className="mt-4 rounded-md border border-border/50 bg-background/35 p-3 text-sm">
              <p className="font-medium">{preview.vulnerability?.vulnerabilityId}</p>
              <p className="mt-1 text-muted-foreground">{preview.vulnerability?.summary || preview.guidance}</p>
              <dl className="mt-3 grid gap-2 sm:grid-cols-2">
                <div><dt className="text-xs text-muted-foreground">Affected range</dt><dd className="break-all">{affectedRange || "-"}</dd></div>
                <div><dt className="text-xs text-muted-foreground">Fixed range</dt><dd className="break-all">{fixedRange || "-"}</dd></div>
                <div><dt className="text-xs text-muted-foreground">Confidence</dt><dd>{preview.vulnerability?.confidence || "-"}</dd></div>
                <div><dt className="text-xs text-muted-foreground">Reachability</dt><dd>{preview.vulnerability?.reachability || "-"}</dd></div>
              </dl>
            </div>
          ) : null}

          {result ? (
            <div className="mt-4 rounded-md border border-emerald-400/30 bg-emerald-500/10 p-3 text-sm">
              {result.mutation?.message || result.guidance || "Denied range was applied."}
            </div>
          ) : null}
          {error ? <p className="mt-4 rounded-md border border-rose-400/30 bg-rose-500/10 p-3 text-sm text-rose-100">{error}</p> : null}
        </div>
      </div>
    </div>
  );
}

const choice = "flex min-h-11 cursor-pointer items-center gap-2 rounded-md border border-border/50 bg-background/35 px-3 py-2 text-sm";
const selectedChoice = cn(choice, "border-primary/70 bg-primary/15 text-primary-foreground");

function MutationFeedback({
  preview,
  applied,
  error
}: {
  preview: UpsertApprovedDependencyResponse | null;
  applied: UpsertApprovedDependencyResponse | null;
  error: string | null;
}) {
  if (error) {
    return <p className="mt-4 rounded-md border border-rose-400/30 bg-rose-500/10 p-3 text-sm text-rose-100">{error}</p>;
  }
  const response = applied ?? preview;
  if (!response) return null;
  return (
    <div className="mt-4 rounded-md border border-border/50 bg-background/35 p-3 text-sm">
      <p className="font-medium">{response.dryRun ? "Dry-run preview" : "Applied decision"}</p>
      <p className="mt-1 text-muted-foreground">{response.message || response.guidance}</p>
      {response.record ? (
        <code className="mt-2 block break-all rounded bg-black/20 p-2 text-xs">
          {response.record.ecosystem}/{response.record.packageName} {response.record.state} {response.record.versionRange}
        </code>
      ) : null}
    </div>
  );
}
