import { Link } from "react-router-dom";
import { ScanSearch } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageCoverageRoot } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { useUsageCoverage } from "./api";
import { coverageRootLabel, formatShare } from "./format";

/** SPEC A1 asks for at least 95% of transcript turns captured. */
function coverageTone(share: number): StatusTone {
  if (share >= 0.95) return "success";
  if (share >= 0.8) return "warning";
  return "danger";
}

/** Header chip: captured share of the trailing 7 days; hidden until coverage is measured. */
export function CoverageChip() {
  const coverage = useUsageCoverage();
  const totals = coverage.data?.totals;
  if (!totals || totals.files === 0n) return null;
  return (
    <Link to="/usage?tab=coverage" className="inline-flex min-h-11 items-center sm:min-h-8" aria-label={`Coverage ${formatShare(totals.capturedShare)} of the last 7 days; open coverage`} data-testid="usage-coverage-chip">
      <StatusBadge tone={coverageTone(totals.capturedShare)}>Coverage {formatShare(totals.capturedShare)} · 7 days</StatusBadge>
    </Link>
  );
}

function RootTile({ root }: { root: UsageCoverageRoot }) {
  const captured = root.indexed + root.owned;
  return (
    <div className="rounded-md border border-border p-3" data-testid="usage-coverage-root">
      <div className="text-xs text-muted-foreground">{coverageRootLabel(root.kind)}</div>
      <div className="text-lg font-bold tabular-nums">{formatShare(root.capturedShare)}</div>
      <div className="text-xs text-muted-foreground">{captured.toString()} of {root.files.toString()} files{root.pending > 0n ? ` · ${root.pending} pending` : ""}</div>
    </div>
  );
}

function issueSummary(unreadable: bigint): string {
  return `${unreadable} ${unreadable === 1n ? "file" : "files"} could not be read`;
}

/**
 * Capture coverage against the on-disk truth. The overview shows the per-root
 * shares and a link here; the Coverage tab adds every named issue and the
 * method for checking it against an independent disk scan.
 */
export function UsageCoverage({ detailed = false }: { detailed?: boolean }) {
  const coverage = useUsageCoverage();
  const roots = coverage.data?.roots ?? [];
  const totals = coverage.data?.totals;
  const issues = roots.flatMap((root) => root.issues.map((issue) => ({ ...issue, root: root.kind })));
  const unreadable = totals?.unreadable ?? 0n;
  return (
    <Card className="space-y-3 p-4 sm:p-5">
      <DataState
        compact
        subject="Coverage"
        loading={coverage.isLoading}
        error={coverage.error}
        isEmpty={roots.length === 0}
        onRetry={() => void coverage.refetch()}
        empty={<EmptyState density="compact" icon={<ScanSearch aria-hidden="true" />} title="Coverage not measured yet" description="The transcript indexer reports coverage here once it has scanned the transcript folders on this machine." />}
      >
        {totals ? (
          <p className="text-sm">
            Captured <strong>{formatShare(totals.capturedShare)}</strong> of transcript bytes since {new Date(coverage.data?.since || Date.now()).toLocaleDateString()} · {totals.turns.toString()} turns · {totals.usageReports.toString()} usage reports
          </p>
        ) : null}
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          {roots.map((root) => <RootTile key={`${root.kind}/${root.harness}`} root={root} />)}
        </div>
        {unreadable > 0n ? (
          detailed ? (
            <details className="rounded-md border border-warning/40 bg-warning/5 px-3 py-2 text-sm" data-testid="usage-coverage-issues">
              <summary className="flex min-h-8 cursor-pointer items-center font-medium">{issueSummary(unreadable)} · details</summary>
              <ul className="mt-2 space-y-1">
                {issues.map((issue) => (
                  <li key={issue.path} className="break-all font-mono text-xs">
                    {issue.path} <span className="font-sans text-muted-foreground">— {issue.reason} ({coverageRootLabel(issue.root)})</span>
                  </li>
                ))}
              </ul>
            </details>
          ) : (
            <Link to="/usage?tab=coverage" className="flex min-h-11 items-center justify-between rounded-md border border-warning/40 bg-warning/5 px-3 text-sm sm:min-h-8">
              <span>{issueSummary(unreadable)}</span><span aria-hidden="true">details →</span>
            </Link>
          )
        ) : null}
        {detailed && coverage.data?.method ? <p className="text-xs text-muted-foreground">{coverage.data.method}</p> : null}
      </DataState>
    </Card>
  );
}
