/**
 * Collapsible list component for displaying preflight validation checks.
 */

import type { BundlePreflightCheck } from "../../lib/api";
import { PREFLIGHT_CHECK_LABELS, PREFLIGHT_CHECK_STYLES } from "../../lib/preflight-constants";
import { getListenURL } from "../../lib/preflight-utils";

interface PreflightCheckListProps {
  checks: BundlePreflightCheck[];
}

export function PreflightCheckList({ checks }: PreflightCheckListProps) {
  if (checks.length === 0) {
    return null;
  }

  return (
    <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-[11px] text-slate-200">
      <summary className="cursor-pointer text-xs font-semibold text-slate-100">
        Test cases ({checks.length})
      </summary>
      <ul className="mt-2 space-y-2">
        {checks.map((check) => {
          const listenURL = getListenURL(check.detail);
          return (
            <li key={check.id} className="rounded-md border border-slate-800/70 bg-slate-950/70 px-3 py-2 space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wide ${PREFLIGHT_CHECK_STYLES[check.status]}`}>
                  {PREFLIGHT_CHECK_LABELS[check.status]}
                </span>
                <span className="text-slate-200">{check.name}</span>
              </div>
              {check.detail && (
                <p className="text-slate-400">
                  <span>{check.detail}</span>
                  {listenURL && (
                    <a
                      className="ml-2 inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide text-sky-300 hover:text-sky-200"
                      href={listenURL}
                      target="_blank"
                      rel="noreferrer"
                    >
                      Open
                    </a>
                  )}
                </p>
              )}
            </li>
          );
        })}
      </ul>
    </details>
  );
}
