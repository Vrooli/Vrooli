// DOC: docs/internal/SEAMS.md#capability-registry-seam
import { CheckCircle, AlertCircle, Circle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useCapabilities } from "../hooks/useCapabilities";
import { strings } from "../consts/strings";
import type { CapabilityState, CapabilityStatus } from "../api/capabilities";

interface IntegrationsPanelProps {
  open: boolean;
}

function statusIcon(status: CapabilityStatus) {
  switch (status) {
    case "available":
      return <CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" />;
    case "unavailable":
      return <AlertCircle className="h-4 w-4 text-red-500 shrink-0" />;
    default:
      return <Circle className="h-4 w-4 text-wc-text-faint shrink-0" />;
  }
}

function borderColor(status: CapabilityStatus): string {
  switch (status) {
    case "available":
      return "border-l-emerald-500";
    case "unavailable":
      return "border-l-red-500";
    default:
      return "border-l-wc-default";
  }
}

function CapabilityCard({ cap }: { cap: CapabilityState }) {
  const isUnavailable = cap.status === "unavailable";

  return (
    <div
      data-testid={`cap-card-${cap.id}`}
      className={`rounded-lg border border-wc-default ${borderColor(cap.status)} border-l-2 bg-wc-surface-input px-3 py-2 space-y-1.5`}
    >
      <div className="flex items-center gap-2">
        {statusIcon(cap.status)}
        <span className="text-xs font-medium text-wc-text-primary">{cap.name}</span>
        <span className="text-[11px] px-1.5 py-0.5 rounded bg-wc-surface text-wc-text-faint">
          {cap.dependencyKind}
        </span>
      </div>

      <p className="text-[11px] text-wc-text-muted">{cap.description}</p>

      {cap.message && (
        <p data-testid={`cap-message-${cap.id}`} className="text-[11px] text-wc-text-faint font-mono">
          {cap.message}
        </p>
      )}

      {cap.features.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-1">
          {cap.features.map((feature) => (
            <span
              key={feature}
              className={`text-[11px] px-1.5 py-0.5 rounded-full border ${
                isUnavailable
                  ? "border-wc-default text-wc-text-faint line-through"
                  : "border-wc-default text-wc-text-muted"
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

export default function IntegrationsPanel({ open }: IntegrationsPanelProps) {
  const { t } = useTranslation();
  const { data, isLoading, isError, error } = useCapabilities(open);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.checking)}</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <p className="text-[11px] text-red-400">{t(strings.integrationsPanel.loadFailed, { message: error.message })}</p>
      </div>
    );
  }

  const capabilities = data?.capabilities ?? [];
  const activeCount = capabilities.filter((c) => c.status === "available").length;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between px-1">
        <span className="text-[11px] text-wc-text-faint">
          {t(strings.integrationsPanel.activeCount, { active: activeCount, total: capabilities.length })}
        </span>
      </div>

      {capabilities.length === 0 ? (
        <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.noneConfigured)}</p>
      ) : (
        <div className="flex flex-col gap-2">
          {capabilities.map((cap) => (
            <CapabilityCard key={cap.id} cap={cap} />
          ))}
        </div>
      )}
    </div>
  );
}
