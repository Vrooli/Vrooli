import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { PairingRequest } from "../realtime/events";
import { useApprovePairingMutation } from "./queries";

interface PendingPairingBannerProps {
  pending: PairingRequest;
  onDismiss: () => void;
}

/**
 * Banner surfaced when a `PAIRING_REQUESTED` event arrives over SSE. Approving
 * is owner-gated (the mutation calls `approvePairing`); dismissing just hides
 * the banner locally without rejecting (the request stays PENDING in the list).
 */
export function PendingPairingBanner({ pending, onDismiss }: PendingPairingBannerProps) {
  const { t } = useTranslation();
  const approve = useApprovePairingMutation();

  const handleApprove = () => {
    approve.mutate(pending.deviceId, { onSuccess: onDismiss });
  };

  return (
    <div
      data-testid={selectors.devices.pendingBanner}
      role="alert"
      className="flex flex-wrap items-center justify-between gap-3 rounded-panel border border-app-warning bg-app-surface-muted p-3"
    >
      <div>
        <p className="text-sm font-medium text-app-foreground">
          {t(strings.devices.pendingBanner.title, { name: pending.name || pending.deviceId })}
        </p>
        <p className="text-xs text-app-muted-foreground">
          {t(strings.devices.pendingBanner.subtitle, { kind: pending.kind || "device" })}
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={handleApprove} disabled={approve.isPending}>
          {t(strings.devices.pendingBanner.approve)}
        </Button>
        <Button size="sm" variant="outline" onClick={onDismiss}>
          {t(strings.devices.pendingBanner.dismiss)}
        </Button>
      </div>
    </div>
  );
}
