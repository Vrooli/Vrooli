import { selectors } from "../consts/selectors";
import { ReceivePanel } from "../features/transfer/ReceivePanel";
import { SendPanel } from "../features/transfer/SendPanel";
import { PendingPairingBanner } from "../features/devices/PendingPairingBanner";
import { useRealtimeContext } from "../features/realtime/RealtimeProvider";

/**
 * The signature surface: a full-height horizontal split — Receive on top, Send
 * on bottom — with distinct accent borders (primary vs accent) marking the two
 * halves. A pairing-request banner overlays the top when one arrives over SSE.
 */
export function TransferPage() {
  const { pendingPairing, dismissPairing } = useRealtimeContext();

  return (
    <div
      data-testid={selectors.pages.transfer}
      className="flex h-full min-h-0 flex-col"
    >
      {pendingPairing && (
        <div className="shrink-0 p-2">
          <PendingPairingBanner pending={pendingPairing} onDismiss={dismissPairing} />
        </div>
      )}
      <div className="grid min-h-0 flex-1 grid-rows-2">
        <ReceivePanel />
        <SendPanel />
      </div>
    </div>
  );
}
