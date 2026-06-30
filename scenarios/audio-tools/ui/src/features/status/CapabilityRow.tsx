import { Capability } from "@vrooli/proto-types/audio-tools/v1/diagnostics/diagnostics_pb";
import {
  type CapabilityHealth,
  type ProviderHealth,
  State,
} from "@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb";
import { Badge } from "../../components/ui/badge";
import { Card } from "../../components/ui/card";
import { ProviderTierBadge } from "./ProviderTierBadge";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

// capabilitySlug doubles as the DOM id for scroll anchors (e.g. /status#stt).
function capabilitySlug(c: Capability): string {
  switch (c) {
    case Capability.STT:
      return "stt";
    case Capability.TTS:
      return "tts";
    case Capability.SUMMARIZE:
      return "summarize";
    case Capability.TRANSCODE:
      return "transcode";
    default:
      return "unknown";
  }
}

type TFn = (key: string, options?: Record<string, unknown>) => string;

function capabilityLabel(t: TFn, c: Capability): string {
  switch (c) {
    case Capability.STT:
      return t(strings.status.capabilityLabelSTT);
    case Capability.TTS:
      return t(strings.status.capabilityLabelTTS);
    case Capability.SUMMARIZE:
      return t(strings.status.capabilityLabelSummarize);
    case Capability.TRANSCODE:
      return t(strings.status.capabilityLabelTranscode);
    default:
      return t(strings.status.capabilityLabelUnknown);
  }
}

function stateLabel(t: TFn, s: State): string {
  switch (s) {
    case State.AVAILABLE:
      return t(strings.status.stateAvailable);
    case State.UNAVAILABLE:
      return t(strings.status.stateUnavailable);
    case State.UNKNOWN:
      return t(strings.status.stateUnknown);
    default:
      return t(strings.status.stateDash);
  }
}

function stateVariant(s: State): "success" | "danger" | "warning" | "neutral" {
  switch (s) {
    case State.AVAILABLE:
      return "success";
    case State.UNAVAILABLE:
      return "danger";
    case State.UNKNOWN:
      return "warning";
    default:
      return "neutral";
  }
}

function ProviderCard({
  provider,
  renderActions,
}: {
  provider: ProviderHealth;
  renderActions?: (providerId: string) => React.ReactNode;
}) {
  const { t } = useTranslation();
  const tr = t as unknown as TFn;
  return (
    <div className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="font-medium text-app-foreground">{provider.providerId}</span>
        <Badge variant={stateVariant(provider.state)}>{stateLabel(tr, provider.state)}</Badge>
      </div>
      <div className="flex items-center gap-2 text-xs text-app-muted-foreground">
        <ProviderTierBadge tier={provider.tier} />
        {provider.lastCheckedAt && <span>{tr(strings.status.checkedAt, { when: provider.lastCheckedAt })}</span>}
      </div>
      {provider.errorMessage && (
        <p className="text-xs text-app-danger">{provider.errorMessage}</p>
      )}
      {renderActions?.(provider.providerId)}
    </div>
  );
}

interface CapabilityRowProps {
  capability: CapabilityHealth;
  // Render-prop slot for per-provider lifecycle action buttons. The
  // status page passes a function that consults ListLocalProviders to
  // decide which buttons to show; non-local providers return null and
  // render no controls.
  renderProviderActions?: (providerId: string) => React.ReactNode;
}

export function CapabilityRow({ capability, renderProviderActions }: CapabilityRowProps) {
  const { t } = useTranslation();
  const tr = t as unknown as TFn;
  return (
    <Card id={capabilitySlug(capability.capability)} className="flex flex-col gap-3 p-4 scroll-mt-20">
      <header className="flex items-center justify-between gap-2">
        <h2 className="text-lg font-semibold text-app-foreground">
          {capabilityLabel(tr, capability.capability)}
        </h2>
        <Badge variant={stateVariant(capability.effectiveState)}>
          {stateLabel(tr, capability.effectiveState)}
        </Badge>
      </header>
      {capability.providers.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">{tr(strings.status.noProviders)}</p>
      ) : (
        <div className="grid gap-2 sm:grid-cols-2">
          {capability.providers.map((p) => (
            <ProviderCard
              key={`${capability.capability}-${p.providerId}`}
              provider={p}
              renderActions={renderProviderActions}
            />
          ))}
        </div>
      )}
    </Card>
  );
}
