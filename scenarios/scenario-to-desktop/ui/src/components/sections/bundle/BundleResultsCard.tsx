/**
 * Displays bundle generation results.
 * Shows bundle directory, manifest path, total size, and artifact counts.
 */

import { CheckCircle, AlertTriangle } from "lucide-react";
import type { BundleStageDetails } from "../../../lib/api";

interface BundleResultsCardProps {
  result: BundleStageDetails;
}

export function BundleResultsCard({ result }: BundleResultsCardProps) {
  const runtimeBinaries = result.runtime_binaries ?? {};
  const copiedArtifacts = result.copied_artifacts ?? [];

  return (
    <div className="space-y-3">
      <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 space-y-2">
        {result.bundle_dir && (
          <div className="flex items-center gap-2 text-[11px]">
            <CheckCircle className="h-4 w-4 text-emerald-400 flex-shrink-0" />
            <span className="text-slate-300">Bundle directory:</span>
            <span className="text-slate-400 truncate max-w-[300px]" title={result.bundle_dir}>
              {result.bundle_dir}
            </span>
          </div>
        )}
        {result.manifest_path && (
          <div className="flex items-center gap-2 text-[11px]">
            <CheckCircle className="h-4 w-4 text-emerald-400 flex-shrink-0" />
            <span className="text-slate-300">Manifest:</span>
            <span className="text-slate-400 truncate max-w-[300px]" title={result.manifest_path}>
              {result.manifest_path}
            </span>
          </div>
        )}
        {result.total_size_human && (
          <div className="flex items-center gap-2 text-[11px]">
            <span className="text-slate-300">Total size:</span>
            <span className="text-slate-400">{result.total_size_human}</span>
          </div>
        )}
        {copiedArtifacts.length > 0 && (
          <div className="flex items-center gap-2 text-[11px]">
            <span className="text-slate-300">Artifacts:</span>
            <span className="text-slate-400">{copiedArtifacts.length} files</span>
          </div>
        )}
      </div>

      {result.size_warning && (
        <div className="flex items-start gap-2 text-[11px] text-amber-300 p-2 rounded-md border border-amber-800/40 bg-amber-950/20">
          <AlertTriangle className="h-4 w-4 flex-shrink-0 mt-0.5" />
          <span>{result.size_warning.message}</span>
        </div>
      )}

      {Object.keys(runtimeBinaries).length > 0 && (
        <div className="space-y-1.5">
          <p className="text-[11px] uppercase tracking-wide text-slate-500">Platform builds</p>
          <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-2 space-y-1">
            {Object.entries(runtimeBinaries).map(([platform, binaryPath]) => (
              <div key={platform} className="flex items-center gap-2 text-[11px]">
                <CheckCircle className="h-3 w-3 text-emerald-400 flex-shrink-0" />
                <span className="font-medium text-slate-300 capitalize">{platform.replace("-", " ")}</span>
                <span className="text-slate-500 truncate ml-auto max-w-[200px]" title={binaryPath}>
                  {binaryPath.split("/").pop()}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
