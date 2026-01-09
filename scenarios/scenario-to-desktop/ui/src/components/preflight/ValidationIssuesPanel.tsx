/**
 * Panel showing validation issues with detailed guidance.
 */

import type { BundleValidationResult, BundleValidationError, BundleMissingBinary, BundleMissingAsset, BundleInvalidChecksum } from "../../lib/api";
import { PREFLIGHT_ISSUE_GUIDANCE } from "../../lib/preflight-constants";

interface ValidationIssuesPanelProps {
  validation: BundleValidationResult;
}

export function ValidationIssuesPanel({ validation }: ValidationIssuesPanelProps) {
  if (validation.valid) {
    return null;
  }

  return (
    <details className="rounded-md border border-red-900/50 bg-red-950/20 p-3 text-[11px] text-red-200" open>
      <summary className="cursor-pointer text-xs font-semibold text-red-100">Validation issues</summary>
      <div className="mt-2 space-y-3">
        {validation.errors && validation.errors.length > 0 && (
          <div className="space-y-2">
            {validation.errors.map((err: BundleValidationError, idx: number) => {
              const guidance = PREFLIGHT_ISSUE_GUIDANCE[err.code];
              return (
                <div key={`${err.code}-${idx}`} className="rounded-md border border-red-900/50 bg-red-950/40 p-2 space-y-1">
                  <p className="font-semibold text-red-100">
                    {guidance?.title || err.code}
                  </p>
                  <p>{err.message}</p>
                  {(err.service || err.path) && (
                    <p className="text-[11px] text-red-200/80">
                      {err.service ? `Service: ${err.service}` : ""}{err.service && err.path ? " · " : ""}{err.path ? `Path: ${err.path}` : ""}
                    </p>
                  )}
                  {guidance && (
                    <>
                      <p className="text-[11px] text-red-200/80">{guidance.meaning}</p>
                      <ul className="space-y-1 text-[11px] text-red-100">
                        {guidance.remediation.map((step, stepIdx) => (
                          <li key={`${err.code}-remedy-${stepIdx}`}>• {step}</li>
                        ))}
                      </ul>
                    </>
                  )}
                </div>
              );
            })}
          </div>
        )}
        {validation.missing_binaries && validation.missing_binaries.length > 0 && (
          <div className="space-y-1 text-[11px] text-red-100">
            <p className="font-semibold text-red-100">Missing binaries</p>
            <ul className="space-y-1">
              {validation.missing_binaries.map((item: BundleMissingBinary, idx: number) => (
                <li key={`${item.service_id}-${item.path}-${idx}`}>
                  {item.service_id}: {item.path} ({item.platform})
                </li>
              ))}
            </ul>
            <p className="text-red-200/80">
              Rebuild the service binaries and re-export the bundle to update manifest paths.
            </p>
          </div>
        )}
        {validation.missing_assets && validation.missing_assets.length > 0 && (
          <div className="space-y-1 text-[11px] text-red-100">
            <p className="font-semibold text-red-100">Missing assets</p>
            <ul className="space-y-1">
              {validation.missing_assets.map((item: BundleMissingAsset, idx: number) => (
                <li key={`${item.service_id}-${item.path}-${idx}`}>
                  {item.service_id}: {item.path}
                </li>
              ))}
            </ul>
            <p className="text-red-200/80">
              Rebuild UI/assets and re-export the bundle so the staged paths exist.
            </p>
          </div>
        )}
        {validation.invalid_checksums && validation.invalid_checksums.length > 0 && (
          <div className="space-y-1 text-[11px] text-red-100">
            <p className="font-semibold text-red-100">Invalid checksums</p>
            <ul className="space-y-1">
              {validation.invalid_checksums.map((item: BundleInvalidChecksum, idx: number) => (
                <li key={`${item.service_id}-${item.path}-${idx}`}>
                  {item.service_id}: {item.path}
                </li>
              ))}
            </ul>
            <p className="text-red-200/80">
              Re-export the bundle after rebuilding assets so checksums match.
            </p>
          </div>
        )}
      </div>
    </details>
  );
}

interface ValidationWarningsPanelProps {
  warnings: Array<{ code: string; message: string }>;
}

export function ValidationWarningsPanel({ warnings }: ValidationWarningsPanelProps) {
  if (!warnings || warnings.length === 0) {
    return null;
  }

  return (
    <details className="rounded-md border border-amber-800/50 bg-amber-950/20 p-3 text-[11px] text-amber-200">
      <summary className="cursor-pointer text-xs font-semibold">Warnings</summary>
      <ul className="mt-2 space-y-1">
        {warnings.map((warn, idx) => (
          <li key={`${warn.code}-${idx}`}>{warn.message}</li>
        ))}
      </ul>
    </details>
  );
}
