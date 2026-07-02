import { describe, expect, it } from "vitest";
import {
  decideMicLifecycle,
  selectStaleLeases,
  type MicLifecycleEvent,
} from "./micLifecyclePolicy";
import type { MicOwner } from "./micOwnership";

describe("decideMicLifecycle", () => {
  it("releases ALL leases and stops the active recording on hidden in a standalone/PWA", () => {
    const d = decideMicLifecycle({ event: "hidden", standalonePwa: true });
    expect(d.release).toBe("all");
    expect(d.stopActiveRecording).toBe(true);
  });

  it("releases only non-active leases on hidden in a normal desktop tab", () => {
    const d = decideMicLifecycle({ event: "hidden", standalonePwa: false });
    expect(d.release).toBe("non-active");
    // The active recording is still stopped by the controller for UI honesty.
    expect(d.stopActiveRecording).toBe(true);
  });

  it.each<MicLifecycleEvent>(["pagehide", "freeze"])(
    "releases ALL leases on %s regardless of platform",
    (event) => {
      expect(decideMicLifecycle({ event, standalonePwa: false }).release).toBe("all");
      expect(decideMicLifecycle({ event, standalonePwa: true }).release).toBe("all");
    },
  );

  it("releases nothing on visible without arming the mic", () => {
    const d = decideMicLifecycle({ event: "visible", standalonePwa: true });
    expect(d.release).toBe("none");
    expect(d.stopActiveRecording).toBe(false);
    expect(Object.prototype.hasOwnProperty.call(d, "rearm")).toBe(false);
  });
});

describe("selectStaleLeases", () => {
  const lease = (owner: MicOwner, id = `${owner}-1`) => ({ id, owner });
  const base = { lowLatencyVoice: false, passiveListenerActive: false };

  it("flags an active-recording lease only when the workflow is idle", () => {
    for (const owner of ["voice-stream", "whisper", "web-speech"] as MicOwner[]) {
      expect(
        selectStaleLeases({ ...base, leases: [lease(owner)], voiceState: "idle" }),
      ).toHaveLength(1);
      // Not flagged while genuinely capturing or settling.
      for (const voiceState of ["preparing", "recording", "listening", "transcribing"] as const) {
        expect(
          selectStaleLeases({ ...base, leases: [lease(owner)], voiceState }),
        ).toHaveLength(0);
      }
    }
  });

  it("flags a prewarm lease only when low-latency voice is disabled", () => {
    expect(
      selectStaleLeases({ ...base, lowLatencyVoice: false, leases: [lease("low-latency-prewarm")], voiceState: "idle" }),
    ).toHaveLength(1);
    expect(
      selectStaleLeases({ ...base, lowLatencyVoice: true, leases: [lease("low-latency-prewarm")], voiceState: "idle" }),
    ).toHaveLength(0);
  });

  it("flags a passive lease only when no passive listener is installed", () => {
    expect(
      selectStaleLeases({ ...base, passiveListenerActive: false, leases: [lease("passive-wake-word")], voiceState: "idle" }),
    ).toHaveLength(1);
    expect(
      selectStaleLeases({ ...base, passiveListenerActive: true, leases: [lease("passive-wake-word")], voiceState: "idle" }),
    ).toHaveLength(0);
  });

  it("never flags transient settings-capture owners", () => {
    const owners: MicOwner[] = [
      "wake-word-enrollment",
      "wake-word-test",
      "speaker-enrollment",
      "mic-permission-probe",
    ];
    expect(
      selectStaleLeases({ ...base, leases: owners.map((o) => lease(o)), voiceState: "idle" }),
    ).toHaveLength(0);
  });
});
