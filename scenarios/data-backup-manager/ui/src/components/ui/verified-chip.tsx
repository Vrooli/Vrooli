import { StatusChip } from "./status-chip";
import { VERIFIED_STRINGS } from "../../consts/statusStrings";
import { verifiedMeta } from "../../lib/status";

/**
 * The product's spine, as a component: turns a target's last-verified timestamp
 * into the verified / verify-stale / unverified chip. A target that is backed
 * up but never verified renders the *unverified* warning chip — never a green
 * success — which is the distinction this whole UI is organized around.
 */
export function VerifiedChip({
  lastVerifiedAt,
  className,
  "data-testid": testId,
}: {
  lastVerifiedAt: Date | undefined;
  className?: string;
  "data-testid"?: string;
}) {
  const meta = verifiedMeta(lastVerifiedAt);
  return (
    <StatusChip
      tone={meta.tone}
      labelKey={VERIFIED_STRINGS[meta.slug]}
      className={className}
      data-testid={testId}
    />
  );
}
