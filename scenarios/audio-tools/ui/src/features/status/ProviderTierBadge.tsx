import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { Badge } from "../../components/ui/badge";

interface ProviderTierBadgeProps {
  tier: ProviderTier;
}

function tierLabel(tier: ProviderTier): string {
  switch (tier) {
    case ProviderTier.LOCAL:
      return "Local";
    case ProviderTier.BYOK:
      return "BYOK";
    case ProviderTier.VROOLI:
      return "Vrooli";
    default:
      return "—";
  }
}

function tierVariant(tier: ProviderTier): "info" | "primary" | "neutral" {
  switch (tier) {
    case ProviderTier.LOCAL:
      return "info";
    case ProviderTier.BYOK:
      return "primary";
    case ProviderTier.VROOLI:
      return "neutral";
    default:
      return "neutral";
  }
}

export function ProviderTierBadge({ tier }: ProviderTierBadgeProps) {
  return <Badge variant={tierVariant(tier)}>{tierLabel(tier)}</Badge>;
}
