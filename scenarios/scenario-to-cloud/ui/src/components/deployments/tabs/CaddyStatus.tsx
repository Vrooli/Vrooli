import {
  Shield,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  ExternalLink,
  ArrowRight,
  Globe,
  Lock,
} from "lucide-react";
import type { CaddyState } from "../../../lib/api";
import { cn } from "../../../lib/utils";

interface CaddyStatusProps {
  caddy: CaddyState;
}

export function CaddyStatus({ caddy }: CaddyStatusProps) {
  return (
    <div className="space-y-6">
      {/* Caddy Status Card */}
      <div
        className={cn(
          "border rounded-lg p-4",
          caddy.running
            ? "border-emerald-500/30 bg-emerald-500/5"
            : "border-red-500/30 bg-red-500/5"
        )}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div
              className={cn(
                "p-2 rounded-lg",
                caddy.running ? "bg-emerald-500/20" : "bg-red-500/20"
              )}
            >
              <Shield
                className={cn(
                  "h-5 w-5",
                  caddy.running ? "text-emerald-400" : "text-red-400"
                )}
              />
            </div>
            <div>
              <h4 className="font-medium text-white">Caddy Server</h4>
              <div className="flex items-center gap-2 mt-1">
                {caddy.running ? (
                  <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium bg-emerald-500/20 text-emerald-400">
                    <CheckCircle2 className="h-3 w-3" />
                    Running
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium bg-red-500/20 text-red-400">
                    <XCircle className="h-3 w-3" />
                    Stopped
                  </span>
                )}
              </div>
            </div>
          </div>

          {caddy.domain && caddy.running && (
            <a
              href={`https://${caddy.domain}`}
              target="_blank"
              rel="noopener noreferrer"
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium",
                "border border-white/10 hover:bg-white/5 transition-colors text-blue-400"
              )}
            >
              <ExternalLink className="h-3 w-3" />
              Visit Site
            </a>
          )}
        </div>
      </div>

      {/* Domain & TLS */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Domain */}
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-slate-800">
              <Globe className="h-5 w-5 text-slate-400" />
            </div>
            <div>
              <h4 className="font-medium text-white">Domain</h4>
              <p className="text-xs text-slate-500">Configured hostname</p>
            </div>
          </div>
          <div className="font-mono text-sm text-slate-300">
            {caddy.domain || <span className="text-slate-500">Not configured</span>}
          </div>
        </div>

        {/* TLS */}
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-slate-800">
              <Lock className="h-5 w-5 text-slate-400" />
            </div>
            <div>
              <h4 className="font-medium text-white">TLS Certificate</h4>
              <p className="text-xs text-slate-500">SSL/TLS status</p>
            </div>
          </div>
          <TLSStatus tls={caddy.tls} />
        </div>
      </div>

      {/* Routes */}
      {caddy.routes.length > 0 && (
        <div className="border border-white/10 rounded-lg bg-slate-900/50 overflow-hidden">
          <div className="px-4 py-3 border-b border-white/10 bg-slate-800/50">
            <h4 className="font-medium text-white text-sm">Routes ({caddy.routes.length})</h4>
          </div>
          <div className="divide-y divide-white/5">
            {caddy.routes.map((route, index) => (
              <div
                key={index}
                className="px-4 py-3 flex items-center justify-between"
              >
                <div className="flex items-center gap-3">
                  <span className="font-mono text-sm text-blue-400">{route.path}</span>
                  <ArrowRight className="h-4 w-4 text-slate-600" />
                  <span className="font-mono text-sm text-slate-300">{route.upstream}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface TLSStatusProps {
  tls: CaddyState["tls"];
}

function TLSStatus({ tls }: TLSStatusProps) {
  if (tls.error) {
    return (
      <div className="flex items-center gap-2 text-red-400 text-sm">
        <XCircle className="h-4 w-4" />
        <span>{tls.error}</span>
      </div>
    );
  }

  if (!tls.valid) {
    return (
      <div className="flex items-center gap-2 text-amber-400 text-sm">
        <AlertTriangle className="h-4 w-4" />
        <span>Certificate not valid</span>
      </div>
    );
  }

  return (
    <div className="space-y-2 text-sm">
      <div className="flex items-center gap-2 text-emerald-400">
        <CheckCircle2 className="h-4 w-4" />
        <span>Valid certificate</span>
      </div>
      {tls.issuer && (
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Issuer:</span>
          <span className="text-slate-300">{tls.issuer}</span>
        </div>
      )}
      {tls.expires && (
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Expires:</span>
          <span className="text-slate-300">{tls.expires}</span>
        </div>
      )}
      {tls.days_remaining !== undefined && (
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Days remaining:</span>
          <span
            className={cn(
              tls.days_remaining < 14 ? "text-amber-400" : "text-slate-300"
            )}
          >
            {tls.days_remaining}
          </span>
        </div>
      )}
    </div>
  );
}
