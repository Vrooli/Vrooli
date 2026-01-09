import { useState } from "react";
import { ChevronDown, ChevronUp, Package, Shield, AlertTriangle, EyeOff, CircleDot } from "lucide-react";
import { Button } from "../../components/ui/button";
import type { ManifestSummary } from "./types";
import type { DeploymentManifestResponse } from "../../lib/api";

interface ManifestSummaryBarProps {
  summary: ManifestSummary;
  exportPreview: DeploymentManifestResponse;
}

export function ManifestSummaryBar({ summary, exportPreview }: ManifestSummaryBarProps) {
  const [showPreview, setShowPreview] = useState(false);

  return (
    <div className="border-t border-white/10 bg-black/30">
      <div className="flex items-center justify-between px-4 py-2">
        <div className="flex flex-wrap items-center gap-4 text-xs text-white/70">
          <div className="flex items-center gap-1.5">
            <Package className="h-3.5 w-3.5" />
            <span>Resources: <strong className="text-white">{summary.resourceCount}</strong></span>
          </div>
          <div className="flex items-center gap-1.5">
            <Shield className="h-3.5 w-3.5 text-emerald-400" />
            <span>Covered: <strong className="text-emerald-200">{summary.strategizedSecrets}/{summary.totalSecrets}</strong></span>
          </div>
          <div className="flex items-center gap-1.5">
            <AlertTriangle className="h-3.5 w-3.5 text-amber-400" />
            <span>Blocking: <strong className="text-amber-200">{summary.blockingSecrets}</strong></span>
          </div>
          <div className="flex items-center gap-1.5">
            <EyeOff className="h-3.5 w-3.5 text-white/40" />
            <span>Excluded: <strong className="text-white/50">{summary.excludedSecrets}</strong></span>
          </div>
          {summary.overriddenSecrets > 0 && (
            <div className="flex items-center gap-1.5">
              <CircleDot className="h-3.5 w-3.5 text-purple-400" />
              <span>Overridden: <strong className="text-purple-200">{summary.overriddenSecrets}</strong></span>
            </div>
          )}
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={() => setShowPreview(!showPreview)}
          className="gap-1.5 text-xs text-white/60"
        >
          {showPreview ? (
            <>
              <ChevronDown className="h-3.5 w-3.5" />
              Hide JSON
            </>
          ) : (
            <>
              <ChevronUp className="h-3.5 w-3.5" />
              Preview JSON
            </>
          )}
        </Button>
      </div>

      {showPreview && (
        <div className="border-t border-white/10 px-4 py-3">
          <div className="max-h-64 overflow-auto rounded-xl border border-white/10 bg-black/50 p-3">
            <pre className="text-xs text-white/80 font-mono whitespace-pre-wrap">
              {JSON.stringify(exportPreview, null, 2)}
            </pre>
          </div>
          <p className="mt-2 text-[10px] text-white/40 text-center">
            This is a preview of what will be exported (with exclusions applied)
          </p>
        </div>
      )}
    </div>
  );
}
