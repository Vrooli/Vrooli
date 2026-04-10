/**
 * Inline signing configuration section for the generator form.
 */

import { useEffect, useState } from "react";
import { AlertTriangle, ExternalLink, RefreshCw, ShieldCheck } from "lucide-react";
import { Button } from "../ui/button";
import type { SigningConfig, SigningReadinessResponse } from "../../lib/api";

export interface SigningInlineSectionProps {
  scenarioName: string;
  signingEnabled: boolean;
  signingConfig?: SigningConfig | null;
  readiness?: SigningReadinessResponse;
  loading: boolean;
  onToggleSigning: (enabled: boolean) => void;
  onOpenSigning: () => void;
  onRefresh: () => void;
}

export function SigningInlineSection({
  scenarioName,
  signingEnabled,
  signingConfig,
  readiness,
  loading,
  onToggleSigning,
  onOpenSigning,
  onRefresh
}: SigningInlineSectionProps) {
  const hasConfig = Boolean(signingConfig);
  const ready = readiness?.ready;
  const readinessNote =
    readiness?.issues && readiness.issues.length > 0 ? readiness.issues[0] : undefined;
  const [expiryWarning, setExpiryWarning] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const stored = window.localStorage.getItem("std_signing_expiry_warning");
    if (stored) {
      setExpiryWarning(stored);
    }
  }, []);

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-4 space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <p className="text-sm font-semibold text-slate-100">Signing (recommended for production)</p>
          <p className="text-xs text-slate-400">
            Signed installers avoid OS warnings. Configure details in the Signing tab or reuse saved config here.
          </p>
        </div>
        <label className="flex items-center gap-2 text-sm text-slate-100">
          <input
            type="checkbox"
            className="h-4 w-4"
            checked={signingEnabled}
            disabled={!scenarioName}
            onChange={(e) => onToggleSigning(e.target.checked)}
          />
          <span>{signingEnabled ? "Sign this build" : "Skip signing for this build"}</span>
        </label>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-xs text-slate-300">
        {loading ? (
          <div className="flex items-center gap-2">
            <RefreshCw className="h-4 w-4 animate-spin text-blue-300" />
            Checking signing status…
          </div>
        ) : signingEnabled ? (
          hasConfig ? (
            ready ? (
              <div className="flex items-center gap-2 text-green-300">
                <ShieldCheck className="h-4 w-4" />
                Signing ready for at least one platform.
              </div>
            ) : (
              <div className="flex items-center gap-2 text-amber-300">
                <AlertTriangle className="h-4 w-4" />
                {readinessNote || "Signing config needs fixes before packaging."}
              </div>
            )
          ) : (
            <div className="flex items-center gap-2 text-amber-300">
              <AlertTriangle className="h-4 w-4" />
              No signing config saved yet. Open the Signing tab to add certificates.
            </div>
          )
        ) : (
          <div className="flex items-center gap-2 text-slate-300">
            <AlertTriangle className="h-4 w-4 text-slate-400" />
            Unsigned installers may trigger OS warnings. Enable signing when you&apos;re ready.
          </div>
        )}

        {signingEnabled && expiryWarning && (
          <div className="flex items-center gap-2 text-amber-300">
            <AlertTriangle className="h-4 w-4" />
            {expiryWarning}
          </div>
        )}

        <div className="flex items-center gap-2">
          <Button type="button" size="sm" variant="outline" onClick={onRefresh} disabled={!scenarioName || loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          {!signingEnabled && (
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => onToggleSigning(true)}
              disabled={!scenarioName}
              className="gap-1 text-blue-200 hover:text-blue-100"
            >
              Enable signing now
            </Button>
          )}
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={onOpenSigning}
            disabled={!scenarioName}
            className="gap-1 text-blue-200 hover:text-blue-100"
          >
            Open Signing tab
            <ExternalLink className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
