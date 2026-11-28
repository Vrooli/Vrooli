import { describe, expect, it, vi } from "vitest";
import { installMicLeaseRecovery } from "./micLeaseRecovery";
import type { MicLeaseSnapshot } from "./micOwnership";

const staleLease: MicLeaseSnapshot = {
  id: "lease-1", owner: "voice-stream", acquiredAt: 1,
  trackCount: 1, liveTrackCount: 1,
};

describe("installMicLeaseRecovery", () => {
  it("treats an in-flight start as preparing before React commits state", () => {
    const onStaleState = vi.fn();
    const recover = vi.fn();
    installMicLeaseRecovery({
      getVoiceState: () => "idle",
      isStarting: () => true,
      isPassiveListenerActive: () => false,
      onStaleState,
      recover,
      getLeases: () => [staleLease],
      subscribe: () => () => {},
    });
    expect(onStaleState).toHaveBeenCalledWith(false);
    expect(recover).not.toHaveBeenCalled();
  });

  it("guards synchronous re-entry while recovering an orphan lease", () => {
    let notify!: (leases: MicLeaseSnapshot[]) => void;
    const recover = vi.fn(() => notify([staleLease]));
    const onStaleState = vi.fn();
    installMicLeaseRecovery({
      getVoiceState: () => "idle",
      isStarting: () => false,
      isPassiveListenerActive: () => false,
      onStaleState,
      recover,
      getLeases: () => [],
      subscribe: (listener) => { notify = listener; return () => {}; },
    });
    notify([staleLease]);
    expect(recover).toHaveBeenCalledTimes(1);
    expect(onStaleState).toHaveBeenCalledTimes(3);
  });
});
