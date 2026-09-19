import { TRUST_TIERS, type TrustTier } from "../../api/console";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

interface TierBadgeProps {
  tier: TrustTier;
  size?: "sm" | "md";
  testId?: string;
  className?: string;
}

export const TIER_LABEL_KEY: Record<TrustTier, (typeof strings.console.tiers)[TrustTier]> = {
  stranger: strings.console.tiers.stranger,
  known: strings.console.tiers.known,
  trusted: strings.console.tiers.trusted,
  owner: strings.console.tiers.owner,
};

export function tierRank(tier: TrustTier): number {
  return TRUST_TIERS.indexOf(tier);
}

/**
 * A trust tier rendered as a rank, not a category: four segments fill from the
 * left so `trusted` is visibly more than `known` without reading the label,
 * and the label is always present so hue is never the only carrier.
 */
export function TierBadge({ tier, size = "sm", testId, className }: TierBadgeProps) {
  const { t } = useTranslation();
  const rank = tierRank(tier);
  const tone =
    tier === "owner" ? "text-app-primary" : tier === "trusted" ? "text-app-success" : tier === "known" ? "text-app-info" : "text-app-muted-foreground";
  return (
    <span
      role="status"
      data-testid={testId}
      data-tier={tier}
      className={[
        "inline-flex items-center gap-1.5 rounded-pill border border-app-border bg-app-surface px-2 font-medium",
        size === "sm" ? "py-0.5 text-xs" : "py-1 text-sm",
        tone,
        className ?? "",
      ].join(" ")}
    >
      <span aria-hidden="true" className="flex items-end gap-px">
        {TRUST_TIERS.map((step, index) => (
          <span
            key={step}
            className={["w-1 rounded-sm", index <= rank ? "bg-current" : "bg-app-border"].join(" ")}
            style={{ height: 4 + index * 2 }}
          />
        ))}
      </span>
      <span>{t(TIER_LABEL_KEY[tier])}</span>
    </span>
  );
}
