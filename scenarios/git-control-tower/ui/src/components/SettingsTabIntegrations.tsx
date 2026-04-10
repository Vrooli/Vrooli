import { CheckCircle, AlertCircle, Circle } from "lucide-react";
import { useCapabilities } from "../lib/hooks";
import type { CapabilityState, CapabilityStatus } from "../lib/api";

interface SettingsTabIntegrationsProps {
  isMobile: boolean;
  repoId?: string | null;
}

function statusIcon(status: CapabilityStatus, size: string) {
  switch (status) {
    case "available":
      return <CheckCircle className={`${size} text-emerald-500 shrink-0`} />;
    case "unavailable":
      return <AlertCircle className={`${size} text-red-500 shrink-0`} />;
    default:
      return <Circle className={`${size} text-slate-600 shrink-0`} />;
  }
}

function borderColor(status: CapabilityStatus): string {
  switch (status) {
    case "available":
      return "border-l-emerald-500";
    case "unavailable":
      return "border-l-red-500";
    default:
      return "border-l-slate-700";
  }
}

function CapabilityCard({ cap, isMobile }: { cap: CapabilityState; isMobile: boolean }) {
  const textSm = isMobile ? "text-sm" : "text-xs";
  const textXs = isMobile ? "text-xs" : "text-[11px]";
  const py = isMobile ? "py-3" : "py-2";
  const px = isMobile ? "px-4" : "px-3";
  const isUnavailable = cap.status === "unavailable";

  return (
    <div
      className={`rounded-lg border border-slate-800 ${borderColor(cap.status)} border-l-2 bg-slate-900/40 ${px} ${py} space-y-1.5`}
    >
      <div className="flex items-center gap-2">
        {statusIcon(cap.status, "h-4 w-4")}
        <span className={`${textSm} font-medium text-slate-200`}>{cap.name}</span>
        <span className={`${textXs} px-1.5 py-0.5 rounded bg-slate-800 text-slate-400`}>
          {cap.dependencyKind}
        </span>
      </div>

      <p className={`${textXs} text-slate-400`}>{cap.description}</p>

      {cap.message && (
        <p className={`${textXs} text-slate-500 font-mono`}>{cap.message}</p>
      )}

      {cap.features.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-1">
          {cap.features.map((feature) => (
            <span
              key={feature}
              className={`${textXs} px-1.5 py-0.5 rounded-full border ${
                isUnavailable
                  ? "border-slate-800 text-slate-600 line-through"
                  : "border-slate-700 text-slate-300"
              }`}
            >
              {feature}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

export function SettingsTabIntegrations({ isMobile }: SettingsTabIntegrationsProps) {
  const { data, isLoading, isError, error } = useCapabilities();

  const textSm = isMobile ? "text-sm" : "text-xs";
  const textXs = isMobile ? "text-xs" : "text-[11px]";
  const gap = isMobile ? "gap-3" : "gap-2";

  if (isLoading) {
    return (
      <div className="space-y-4">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>Integrations</h3>
        <p className={`${textXs} text-slate-500`}>Checking integrations...</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>Integrations</h3>
        <p className={`${textXs} text-red-400`}>Failed to load integrations: {error.message}</p>
      </div>
    );
  }

  const capabilities = data?.capabilities ?? [];
  const activeCount = capabilities.filter((c) => c.status === "available").length;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>Integrations</h3>
        <span className={`${textXs} text-slate-500`}>
          {activeCount}/{capabilities.length} active
        </span>
      </div>

      {capabilities.length === 0 ? (
        <p className={`${textXs} text-slate-500`}>No integrations configured.</p>
      ) : (
        <div className={`flex flex-col ${gap}`}>
          {capabilities.map((cap) => (
            <CapabilityCard key={cap.id} cap={cap} isMobile={isMobile} />
          ))}
        </div>
      )}
    </div>
  );
}
