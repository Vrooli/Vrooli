import { Capability } from "@vrooli/proto-types/audio-tools/v1/diagnostics/diagnostics_pb";
import {
  type CapabilityHealth,
  type ProviderHealth,
  State,
} from "@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb";
import { Badge } from "../../components/ui/badge";
import { Card } from "../../components/ui/card";
import { ProviderTierBadge } from "./ProviderTierBadge";

function capabilityLabel(c: Capability): string {
  switch (c) {
    case Capability.STT:
      return "Speech → Text";
    case Capability.TTS:
      return "Text → Speech";
    case Capability.SUMMARIZE:
      return "Summarize";
    case Capability.TRANSCODE:
      return "Transcode";
    default:
      return "Unknown";
  }
}

function stateLabel(s: State): string {
  switch (s) {
    case State.AVAILABLE:
      return "Available";
    case State.UNAVAILABLE:
      return "Unavailable";
    case State.UNKNOWN:
      return "Unknown";
    default:
      return "—";
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

function ProviderCard({ provider }: { provider: ProviderHealth }) {
  return (
    <div className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="font-medium text-app-foreground">{provider.providerId}</span>
        <Badge variant={stateVariant(provider.state)}>{stateLabel(provider.state)}</Badge>
      </div>
      <div className="flex items-center gap-2 text-xs text-app-muted-foreground">
        <ProviderTierBadge tier={provider.tier} />
        {provider.lastCheckedAt && <span>Checked {provider.lastCheckedAt}</span>}
      </div>
      {provider.errorMessage && (
        <p className="text-xs text-app-danger">{provider.errorMessage}</p>
      )}
    </div>
  );
}

interface CapabilityRowProps {
  capability: CapabilityHealth;
}

export function CapabilityRow({ capability }: CapabilityRowProps) {
  return (
    <Card className="flex flex-col gap-3 p-4">
      <header className="flex items-center justify-between gap-2">
        <h2 className="text-lg font-semibold text-app-foreground">
          {capabilityLabel(capability.capability)}
        </h2>
        <Badge variant={stateVariant(capability.effectiveState)}>
          {stateLabel(capability.effectiveState)}
        </Badge>
      </header>
      {capability.providers.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">No providers registered.</p>
      ) : (
        <div className="grid gap-2 sm:grid-cols-2">
          {capability.providers.map((p) => (
            <ProviderCard key={`${capability.capability}-${p.providerId}`} provider={p} />
          ))}
        </div>
      )}
    </Card>
  );
}
