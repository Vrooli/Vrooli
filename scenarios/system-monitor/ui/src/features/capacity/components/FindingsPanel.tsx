import { formatBytes } from '../../../shared/utils/formatters';
import type { CapacityFinding } from '../types';

interface FindingsPanelProps {
  findings: CapacityFinding[];
  available: boolean;
}

const CLASS_LABEL: Record<string, string> = {
  unclaimed: 'Unclaimed',
  over_claim: 'Over-claim',
  claimed: 'Claimed',
};

/**
 * The reconciliation panel: observed GPU consumers classified against the
 * ledger. Unclaimed/over-claim findings are warnings (the adoption-tracker
 * signal); claimed findings are informational. Surfaces sensing unavailability
 * honestly rather than implying "no consumers".
 */
export const FindingsPanel = ({ findings, available }: FindingsPanelProps) => {
  if (!available) {
    return (
      <div className="card" data-sm-style="sm-style-2dce25a9dc">
        Capacity sensing is unavailable on this host (no GPU probe), so unclaimed
        consumers cannot be reconciled.
      </div>
    );
  }

  if (findings.length === 0) {
    return (
      <div className="card" data-sm-style="sm-style-7b635e08e2">
        No GPU consumers above the tracking threshold.
      </div>
    );
  }

  return (
    <div className="card" data-sm-style="sm-style-7b635e08e2">
      <ul data-sm-style="sm-style-5897a24ddc">
        {findings.map((finding) => {
          const isWarn = finding.severity === 'warn';
          return (
            <li
              key={`${finding.ownerId}-${String(finding.pid)}-${finding.class}`}
              className={`finding-item ${isWarn ? 'finding-warning' : 'finding-neutral'}`}
            >
              <span
                className={`finding-label ${isWarn ? 'text-warning' : 'text-muted'}`}
              >
                {CLASS_LABEL[finding.class] ?? finding.class}
              </span>
              <span data-sm-style="sm-style-0c1277dcaf">
                {finding.message || `${finding.ownerId} — ${formatBytes(Number(finding.observedBytes))}`}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
};
