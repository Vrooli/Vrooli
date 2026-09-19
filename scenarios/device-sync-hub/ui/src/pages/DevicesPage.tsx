import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useSession } from "../features/session/SessionProvider";
import { OwnerSignIn } from "../features/session/OwnerSignIn";
import { DeviceList } from "../features/devices/DeviceList";
import { IssuePairingCode } from "../features/devices/IssuePairingCode";
import { PendingPairingBanner } from "../features/devices/PendingPairingBanner";
import { useRealtimeContext } from "../features/realtime/RealtimeProvider";

/**
 * Owner-gated device management. Without an owner token it shows the owner
 * sign-in prompt; with one it lists devices (rename/revoke/approve), issues
 * pairing codes (with QR), and surfaces inbound pairing requests.
 */
export function DevicesPage() {
  const { t } = useTranslation();
  const { isOwner } = useSession();
  const { pendingPairing, dismissPairing } = useRealtimeContext();

  return (
    <section
      data-testid={selectors.pages.devices}
      aria-labelledby="devices-heading"
      className="flex flex-col gap-6 overflow-auto p-6"
    >
      <h2 id="devices-heading" className="text-2xl font-semibold">
        {t(strings.pages.devices.title)}
      </h2>

      <OwnerSignIn />

      {isOwner ? (
        <div data-testid={selectors.devices.panel} className="flex flex-col gap-6">
          {pendingPairing && (
            <PendingPairingBanner pending={pendingPairing} onDismiss={dismissPairing} />
          )}
          <IssuePairingCode />
          <DeviceList />
        </div>
      ) : (
        <p data-testid={selectors.devices.signInPrompt} className="text-sm text-app-muted-foreground">
          {t(strings.devices.signInPrompt)}
        </p>
      )}
    </section>
  );
}
