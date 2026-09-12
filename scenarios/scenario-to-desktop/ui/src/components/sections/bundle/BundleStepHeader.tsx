/**
 * Header component for each bundle step.
 * Displays step number, title, subtitle, and status badge.
 * Follows the same pattern as PreflightStepHeader for consistency.
 */

import {
  PREFLIGHT_STEP_CIRCLE_STYLES,
  PREFLIGHT_STEP_STYLES,
  type PreflightStepStatus,
} from "../../../lib/preflight-constants";

interface BundleStepHeaderProps {
  index: number;
  title: string;
  status: PreflightStepStatus;
  subtitle?: string;
}

export function BundleStepHeader({
  index,
  title,
  status,
  subtitle,
}: BundleStepHeaderProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-start gap-3">
        <div
          className={`flex h-7 w-7 items-center justify-center rounded-full border text-xs font-semibold ${PREFLIGHT_STEP_CIRCLE_STYLES[status.state]}`}
        >
          {index}
        </div>
        <div>
          <p className="text-sm font-semibold text-slate-100">{title}</p>
          {subtitle && <p className="text-[11px] text-slate-400">{subtitle}</p>}
        </div>
      </div>
      <span
        className={`rounded-full border px-2 py-0.5 text-[11px] uppercase tracking-wide ${PREFLIGHT_STEP_STYLES[status.state]}`}
      >
        {status.label}
      </span>
    </div>
  );
}
