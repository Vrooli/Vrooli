import { Cloud, CheckCircle, XCircle, RefreshCw, Edit, Trash2, ExternalLink } from "lucide-react";
import type { DistributionTarget, DistributionTestResult } from "../../lib/api";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";

interface DistributionTargetCardProps {
  target: DistributionTarget;
  testResult?: DistributionTestResult;
  onEdit: () => void;
  onDelete: () => void;
  onTest: () => void;
  isTesting?: boolean;
  isDeleting?: boolean;
}

const PROVIDER_LABELS: Record<string, string> = {
  s3: "AWS S3",
  r2: "Cloudflare R2",
  "s3-compatible": "S3-Compatible",
};

const PROVIDER_COLORS: Record<string, string> = {
  s3: "text-orange-400",
  r2: "text-amber-400",
  "s3-compatible": "text-blue-400",
};

export function DistributionTargetCard({
  target,
  testResult,
  onEdit,
  onDelete,
  onTest,
  isTesting,
  isDeleting,
}: DistributionTargetCardProps) {
  return (
    <div
      className={cn(
        "rounded-lg border p-4 space-y-3",
        target.enabled
          ? "border-slate-700 bg-slate-950/50"
          : "border-slate-800 bg-slate-950/30 opacity-60"
      )}
    >
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <Cloud className={cn("h-5 w-5", PROVIDER_COLORS[target.provider] || "text-slate-400")} />
          <div>
            <h3 className="font-semibold text-slate-100">{target.name}</h3>
            <p className="text-xs text-slate-400">
              {PROVIDER_LABELS[target.provider] || target.provider}
            </p>
          </div>
        </div>
        <div
          className={cn(
            "px-2 py-0.5 rounded text-xs font-medium",
            target.enabled
              ? "bg-green-950/50 text-green-300 border border-green-800/50"
              : "bg-slate-800 text-slate-400 border border-slate-700"
          )}
        >
          {target.enabled ? "Enabled" : "Disabled"}
        </div>
      </div>

      {/* Details */}
      <div className="space-y-1.5 text-sm">
        <div className="flex items-center justify-between">
          <span className="text-slate-400">Bucket:</span>
          <span className="font-mono text-slate-200">{target.bucket}</span>
        </div>

        {target.endpoint && (
          <div className="flex items-center justify-between">
            <span className="text-slate-400">Endpoint:</span>
            <span className="font-mono text-slate-200 text-xs truncate max-w-[200px]" title={target.endpoint}>
              {target.endpoint}
            </span>
          </div>
        )}

        {target.region && (
          <div className="flex items-center justify-between">
            <span className="text-slate-400">Region:</span>
            <span className="font-mono text-slate-200">{target.region}</span>
          </div>
        )}

        {target.path_prefix && (
          <div className="flex items-center justify-between">
            <span className="text-slate-400">Path Prefix:</span>
            <span className="font-mono text-slate-200">{target.path_prefix}</span>
          </div>
        )}

        <div className="flex items-center justify-between">
          <span className="text-slate-400">Credentials:</span>
          <span className="font-mono text-slate-300 text-xs">
            ${target.access_key_id_env}
          </span>
        </div>

        {target.cdn_url && (
          <div className="flex items-center justify-between">
            <span className="text-slate-400">CDN:</span>
            <a
              href={target.cdn_url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 text-blue-300 hover:text-blue-200 text-xs"
            >
              {target.cdn_url}
              <ExternalLink className="h-3 w-3" />
            </a>
          </div>
        )}
      </div>

      {/* Test Result */}
      {testResult && (
        <div
          className={cn(
            "p-2 rounded text-sm",
            testResult.success
              ? "bg-green-950/30 border border-green-800/30"
              : "bg-red-950/30 border border-red-800/30"
          )}
        >
          <div className="flex items-center gap-2">
            {testResult.success ? (
              <CheckCircle className="h-4 w-4 text-green-400" />
            ) : (
              <XCircle className="h-4 w-4 text-red-400" />
            )}
            <span className={testResult.success ? "text-green-300" : "text-red-300"}>
              {testResult.success ? "Connection successful" : "Connection failed"}
            </span>
          </div>
          {testResult.message && (
            <p className="text-xs text-slate-400 mt-1">{testResult.message}</p>
          )}
          {testResult.error && (
            <p className="text-xs text-red-300 mt-1">{testResult.error}</p>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2 pt-2 border-t border-slate-800">
        <Button
          variant="outline"
          size="sm"
          onClick={onTest}
          disabled={isTesting || !target.enabled}
          className="flex-1"
        >
          {isTesting ? (
            <RefreshCw className="h-4 w-4 mr-1 animate-spin" />
          ) : (
            <CheckCircle className="h-4 w-4 mr-1" />
          )}
          Test
        </Button>
        <Button variant="outline" size="sm" onClick={onEdit}>
          <Edit className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={onDelete}
          disabled={isDeleting}
          className="text-red-400 hover:text-red-300 hover:border-red-800"
        >
          {isDeleting ? (
            <RefreshCw className="h-4 w-4 animate-spin" />
          ) : (
            <Trash2 className="h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  );
}
