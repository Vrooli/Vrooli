// DOC: docs/internal/SEAMS.md#capability-registry-seam
// DOC: docs/internal/SEAMS.md#connected-scenarios-registry-seam
import { CheckCircle, AlertCircle, Circle, Boxes, Plug } from "lucide-react";
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
  const { t } = useTranslation();
  const isUnavailable = cap.status === "unavailable";
  const isScenario = cap.dependencyKind === "scenario";
  // For scenario integrations that are unavailable, the message is a
  // CLI-install hint rather than a diagnostic. Treat the badge accordingly.
  const showNotYetBadge = isScenario && cap.status !== "available";

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
        {showNotYetBadge && (
          <span
            data-testid={`cap-not-yet-${cap.id}`}
            className="text-[11px] px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-300 border border-amber-500/30"
          >
            {t(strings.integrationsPanel.scenarioNotYetAvailable)}
          </span>
        )}
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

interface GroupProps {
  testId: string;
  icon: React.ReactNode;
  heading: string;
  description: string;
  items: CapabilityState[];
}

function IntegrationsGroup({ testId, icon, heading, description, items }: GroupProps) {
  if (items.length === 0) return null;
  return (
    <section data-testid={testId} className="space-y-2">
      <header className="flex items-center gap-2 px-1">
        <span className="text-wc-text-muted">{icon}</span>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">{heading}</h3>
      </header>
      <p className="text-[11px] text-wc-text-faint px-1">{description}</p>
      <div className="flex flex-col gap-2">
        {items.map((cap) => (
          <CapabilityCard key={cap.id} cap={cap} />
        ))}
      </div>
    </section>
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
  const scenarios = capabilities.filter((c) => c.dependencyKind === "scenario");
  const resources = capabilities.filter((c) => c.dependencyKind !== "scenario");

  if (capabilities.length === 0) {
    return (
      <div className="space-y-3">
        <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.noneConfigured)}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between px-1">
        <span className="text-[11px] text-wc-text-faint">
          {t(strings.integrationsPanel.activeCount, { active: activeCount, total: capabilities.length })}
        </span>
      </div>

      <IntegrationsGroup
        testId="integrations-group-scenarios"
        icon={<Boxes className="h-4 w-4" />}
        heading={t(strings.integrationsPanel.connectedScenariosHeading)}
        description={t(strings.integrationsPanel.connectedScenariosDescription)}
        items={scenarios}
      />

      <IntegrationsGroup
        testId="integrations-group-resources"
        icon={<Plug className="h-4 w-4" />}
        heading={t(strings.integrationsPanel.localResourcesHeading)}
        description={t(strings.integrationsPanel.localResourcesDescription)}
        items={resources}
      />
    </div>
  );
}
