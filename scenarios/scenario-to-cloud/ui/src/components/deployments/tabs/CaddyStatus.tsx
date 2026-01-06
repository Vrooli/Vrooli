import { useState } from "react";
import {
  Shield,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  ExternalLink,
  ArrowRight,
  Globe,
  Lock,
  RefreshCw,
  Loader2,
  Play,
  Square,
  RotateCcw,
  RefreshCcw,
  Wifi,
  WifiOff,
  Info,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import type { CaddyState, CaddyAction } from "../../../lib/api";
import { cn } from "../../../lib/utils";
import { useDNSCheck, useCaddyControl, useTLSInfo, useTLSRenew } from "../../../hooks/useLiveState";

interface CaddyStatusProps {
  caddy: CaddyState;
  deploymentId: string;
}

export function CaddyStatus({ caddy, deploymentId }: CaddyStatusProps) {
  const [showRoutes, setShowRoutes] = useState(true);
  const [showDNSHint, setShowDNSHint] = useState<string | null>(null);

  // Fetch DNS check status
  const { data: dnsCheck, isLoading: dnsLoading, refetch: refetchDNS } = useDNSCheck(deploymentId);

  // Fetch detailed TLS info
  const { data: tlsInfo, isLoading: tlsLoading, refetch: refetchTLS } = useTLSInfo(deploymentId);

  // Caddy control mutation
  const caddyControl = useCaddyControl(deploymentId);

  // TLS renewal mutation
  const tlsRenew = useTLSRenew(deploymentId);

  const handleCaddyAction = async (action: CaddyAction) => {
    await caddyControl.mutateAsync(action);
  };

  const handleTLSRenew = async () => {
    await tlsRenew.mutateAsync();
  };

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

          {/* Caddy Control Buttons */}
          <div className="flex items-center gap-2">
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

            {!caddy.running ? (
              <button
                onClick={() => handleCaddyAction("start")}
                disabled={caddyControl.isPending}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors",
                  "border border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20",
                  caddyControl.isPending && "opacity-50 cursor-not-allowed"
                )}
              >
                {caddyControl.isPending ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <Play className="h-3 w-3" />
                )}
                Start
              </button>
            ) : (
              <>
                <button
                  onClick={() => handleCaddyAction("restart")}
                  disabled={caddyControl.isPending}
                  className={cn(
                    "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors",
                    "border border-blue-500/30 bg-blue-500/10 text-blue-400 hover:bg-blue-500/20",
                    caddyControl.isPending && "opacity-50 cursor-not-allowed"
                  )}
                >
                  {caddyControl.isPending ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <RotateCcw className="h-3 w-3" />
                  )}
                  Restart
                </button>
                <button
                  onClick={() => handleCaddyAction("reload")}
                  disabled={caddyControl.isPending}
                  className={cn(
                    "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors",
                    "border border-amber-500/30 bg-amber-500/10 text-amber-400 hover:bg-amber-500/20",
                    caddyControl.isPending && "opacity-50 cursor-not-allowed"
                  )}
                >
                  {caddyControl.isPending ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <RefreshCcw className="h-3 w-3" />
                  )}
                  Reload
                </button>
                <button
                  onClick={() => handleCaddyAction("stop")}
                  disabled={caddyControl.isPending}
                  className={cn(
                    "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors",
                    "border border-red-500/30 bg-red-500/10 text-red-400 hover:bg-red-500/20",
                    caddyControl.isPending && "opacity-50 cursor-not-allowed"
                  )}
                >
                  {caddyControl.isPending ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : (
                    <Square className="h-3 w-3" />
                  )}
                  Stop
                </button>
              </>
            )}
          </div>
        </div>

        {/* Caddy Control Result */}
        {caddyControl.data && (
          <div
            className={cn(
              "mt-3 p-2 rounded text-xs",
              caddyControl.data.ok
                ? "bg-emerald-500/10 text-emerald-400"
                : "bg-red-500/10 text-red-400"
            )}
          >
            {caddyControl.data.message}
            {caddyControl.data.output && (
              <pre className="mt-1 text-slate-400 whitespace-pre-wrap">{caddyControl.data.output}</pre>
            )}
          </div>
        )}
      </div>

      {/* Domain & DNS Validation */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Domain with DNS Check */}
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-slate-800">
                <Globe className="h-5 w-5 text-slate-400" />
              </div>
              <div>
                <h4 className="font-medium text-white">Domain</h4>
                <p className="text-xs text-slate-500">DNS Configuration</p>
              </div>
            </div>
            <button
              onClick={() => refetchDNS()}
              disabled={dnsLoading}
              className="p-1.5 rounded border border-white/10 hover:bg-white/5 transition-colors"
            >
              <RefreshCw
                className={cn("h-3.5 w-3.5 text-slate-400", dnsLoading && "animate-spin")}
              />
            </button>
          </div>

          <div className="font-mono text-sm text-slate-300 mb-3">
            {caddy.domain || <span className="text-slate-500">Not configured</span>}
          </div>

          {/* DNS Status */}
          {dnsLoading ? (
            <div className="flex items-center gap-2 text-xs text-slate-400">
              <Loader2 className="h-3 w-3 animate-spin" />
              Checking DNS...
            </div>
          ) : dnsCheck ? (
            <div className="space-y-3">
              <div
                className={cn(
                  "flex items-center gap-2 text-sm",
                  dnsCheck.ok ? "text-emerald-400" : "text-amber-400"
                )}
              >
                {dnsCheck.ok ? (
                  <>
                    <Wifi className="h-4 w-4" />
                    <span>DNS checks passed</span>
                  </>
                ) : (
                  <>
                    <WifiOff className="h-4 w-4" />
                    <span>DNS checks need attention</span>
                  </>
                )}
              </div>

              {dnsCheck.vps_ips && dnsCheck.vps_ips.length > 0 && (
                <div className="text-xs text-slate-400">
                  VPS IPs: <span className="text-slate-300">{dnsCheck.vps_ips.join(", ")}</span>
                </div>
              )}

              <div className="space-y-3">
                {dnsCheck.domains.map((domainCheck) => {
                  const label =
                    domainCheck.role === "apex"
                      ? "Apex"
                      : domainCheck.role === "www"
                      ? "WWW"
                      : "Origin";

                  return (
                    <div key={domainCheck.domain} className="rounded border border-white/5 bg-slate-900/40 p-2">
                      <div className="flex items-center justify-between gap-2">
                        <div className="text-xs text-slate-400">
                          <span className="text-slate-200">{label}</span> •{" "}
                          <span className="font-mono text-slate-300">{domainCheck.domain}</span>
                        </div>
                        <span
                          className={cn(
                            "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium",
                            domainCheck.ok
                              ? "bg-emerald-500/10 text-emerald-400"
                              : "bg-amber-500/10 text-amber-400"
                          )}
                        >
                          {domainCheck.ok ? "OK" : "Check"}
                        </span>
                      </div>

                      <div className="mt-1 text-xs text-slate-400">{domainCheck.message}</div>

                      {domainCheck.domain_ips && domainCheck.domain_ips.length > 0 && (
                        <div className="mt-1 text-xs text-slate-400">
                          Domain IPs:{" "}
                          <span className="text-slate-300">{domainCheck.domain_ips.join(", ")}</span>
                        </div>
                      )}

                      {domainCheck.proxied && (
                        <div className="mt-1 text-xs text-blue-400">Proxied via Cloudflare</div>
                      )}

                      {!domainCheck.ok && domainCheck.hint && (
                        <div className="mt-2">
                          <button
                            onClick={() =>
                              setShowDNSHint(showDNSHint === domainCheck.domain ? null : domainCheck.domain)
                            }
                            className="flex items-center gap-1.5 text-xs text-blue-400 hover:text-blue-300"
                          >
                            <Info className="h-3 w-3" />
                            {showDNSHint === domainCheck.domain ? "Hide" : "Show"} how to fix
                            {showDNSHint === domainCheck.domain ? (
                              <ChevronUp className="h-3 w-3" />
                            ) : (
                              <ChevronDown className="h-3 w-3" />
                            )}
                          </button>
                          {showDNSHint === domainCheck.domain && (
                            <pre className="mt-2 p-2 rounded bg-slate-800 text-xs text-slate-300 whitespace-pre-wrap">
                              {domainCheck.hint}
                            </pre>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          ) : null}
        </div>

        {/* TLS Certificate */}
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-slate-800">
                <Lock className="h-5 w-5 text-slate-400" />
              </div>
              <div>
                <h4 className="font-medium text-white">TLS Certificate</h4>
                <p className="text-xs text-slate-500">SSL/TLS Status</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => refetchTLS()}
                disabled={tlsLoading}
                className="p-1.5 rounded border border-white/10 hover:bg-white/5 transition-colors"
              >
                <RefreshCw
                  className={cn("h-3.5 w-3.5 text-slate-400", tlsLoading && "animate-spin")}
                />
              </button>
              <button
                onClick={handleTLSRenew}
                disabled={tlsRenew.isPending || !caddy.running}
                className={cn(
                  "flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors",
                  "border border-purple-500/30 bg-purple-500/10 text-purple-400 hover:bg-purple-500/20",
                  (tlsRenew.isPending || !caddy.running) && "opacity-50 cursor-not-allowed"
                )}
              >
                {tlsRenew.isPending ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <RefreshCw className="h-3 w-3" />
                )}
                Renew
              </button>
            </div>
          </div>

          {/* TLS Status */}
          {tlsLoading ? (
            <div className="flex items-center gap-2 text-xs text-slate-400">
              <Loader2 className="h-3 w-3 animate-spin" />
              Checking certificate...
            </div>
          ) : tlsInfo ? (
            <TLSDetails tlsInfo={tlsInfo} />
          ) : (
            <TLSStatus tls={caddy.tls} />
          )}

          {/* TLS Renewal Result */}
          {tlsRenew.data && (
            <div
              className={cn(
                "mt-3 p-2 rounded text-xs",
                tlsRenew.data.ok
                  ? "bg-emerald-500/10 text-emerald-400"
                  : "bg-red-500/10 text-red-400"
              )}
            >
              {tlsRenew.data.message}
            </div>
          )}
        </div>
      </div>

      {/* Routes */}
      {caddy.routes.length > 0 && (
        <div className="border border-white/10 rounded-lg bg-slate-900/50 overflow-hidden">
          <button
            onClick={() => setShowRoutes(!showRoutes)}
            className="w-full px-4 py-3 border-b border-white/10 bg-slate-800/50 flex items-center justify-between hover:bg-slate-800/70 transition-colors"
          >
            <h4 className="font-medium text-white text-sm">Routes ({caddy.routes.length})</h4>
            {showRoutes ? (
              <ChevronUp className="h-4 w-4 text-slate-400" />
            ) : (
              <ChevronDown className="h-4 w-4 text-slate-400" />
            )}
          </button>
          {showRoutes && (
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
          )}
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

interface TLSDetailsProps {
  tlsInfo: {
    ok: boolean;
    valid: boolean;
    issuer?: string;
    subject?: string;
    not_before?: string;
    not_after?: string;
    days_remaining: number;
    serial_number?: string;
    sans?: string[];
    error?: string;
  };
}

function TLSDetails({ tlsInfo }: TLSDetailsProps) {
  if (tlsInfo.error) {
    return (
      <div className="flex items-center gap-2 text-amber-400 text-sm">
        <AlertTriangle className="h-4 w-4" />
        <span>{tlsInfo.error}</span>
      </div>
    );
  }

  if (!tlsInfo.valid) {
    return (
      <div className="flex items-center gap-2 text-red-400 text-sm">
        <XCircle className="h-4 w-4" />
        <span>Certificate expired or invalid</span>
      </div>
    );
  }

  return (
    <div className="space-y-2 text-sm">
      <div className="flex items-center gap-2 text-emerald-400">
        <CheckCircle2 className="h-4 w-4" />
        <span>Valid certificate</span>
      </div>

      {tlsInfo.issuer && (
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Issuer:</span>
          <span className="text-slate-300">{tlsInfo.issuer}</span>
        </div>
      )}

      {tlsInfo.not_before && (
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Valid from:</span>
          <span className="text-slate-300">{tlsInfo.not_before}</span>
        </div>
      )}

      {tlsInfo.not_after && (
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500">Expires:</span>
          <span className="text-slate-300">{tlsInfo.not_after}</span>
        </div>
      )}

      <div className="flex items-center justify-between text-xs">
        <span className="text-slate-500">Days remaining:</span>
        <span
          className={cn(
            tlsInfo.days_remaining < 14
              ? "text-amber-400"
              : tlsInfo.days_remaining < 30
                ? "text-yellow-400"
                : "text-emerald-400"
          )}
        >
          {tlsInfo.days_remaining}
        </span>
      </div>

      {tlsInfo.sans && tlsInfo.sans.length > 0 && (
        <div className="text-xs">
          <span className="text-slate-500">Domains:</span>
          <div className="mt-1 flex flex-wrap gap-1">
            {tlsInfo.sans.map((san, i) => (
              <span key={i} className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">
                {san}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
