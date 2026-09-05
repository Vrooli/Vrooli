import { formatBytes, formatPercentage } from '../../../shared/utils/formatters';
import type { GpuCapacity } from '../types';

interface GpuContentionCardProps {
  gpu: GpuCapacity;
}

/**
 * One GPU's contention picture: observed used VRAM and the sum of active claims
 * against the total, shown as a dual-track bar. Claimed vs used divergence is
 * the contention signal (claims under-/over-shoot observed usage).
 */
export const GpuContentionCard = ({ gpu }: GpuContentionCardProps) => {
  const total = Number(gpu.totalBytes);
  const used = Number(gpu.usedBytes);
  const claimed = Number(gpu.claimedBytes);
  const free = Number(gpu.freeBytes);

  const usedPct = total > 0 ? Math.min(100, (used / total) * 100) : 0;
  const claimedPct = total > 0 ? Math.min(100, (claimed / total) * 100) : 0;

  return (
    <div className="card" data-sm-style="sm-style-7b635e08e2">
      <h3 data-sm-style="sm-style-ae54b9d902">
        GPU {gpu.index} · {gpu.name || 'unknown'}
      </h3>

      <div
        aria-label={`GPU ${String(gpu.index)} observed VRAM usage`}
        role="meter"
        aria-valuenow={Math.round(usedPct)}
        aria-valuemin={0}
        aria-valuemax={100}
        data-sm-style="sm-style-1c3eec730d"
      >
        <div style={{ position: 'absolute', inset: 0, width: `${String(usedPct)}%`, background: 'var(--color-primary)' }} />
      </div>
      <div data-sm-style="sm-style-90be12c424">
        <span>Used (observed): {formatBytes(used)} ({formatPercentage(usedPct)})</span>
        <span>Free: {formatBytes(free)}</span>
      </div>

      <div
        aria-label={`GPU ${String(gpu.index)} claimed VRAM`}
        role="meter"
        aria-valuenow={Math.round(claimedPct)}
        aria-valuemin={0}
        aria-valuemax={100}
        data-sm-style="sm-style-71c578afd9"
      >
        <div className="gpu-contention-claimed" style={{ width: `${String(claimedPct)}%` }} />
      </div>
      <div data-sm-style="sm-style-90be12c424">
        <span>Claimed: {formatBytes(claimed)} ({formatPercentage(claimedPct)})</span>
        <span>Total: {formatBytes(total)}</span>
      </div>
    </div>
  );
};
