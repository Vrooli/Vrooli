import { describe, expect, it } from "vitest";
import { getDefaultVoiceTransport, PcmVoiceStreamProvider } from "@vrooli/audio-capture-browser";
import { registerScenarioVoiceTransport } from "./index";
import { buildVoiceStreamWsUrl } from "./api/voice";

describe("scenario voice bootstrap", () => {
  it("registers the scenario adapter and constructs the concrete provider without capture or requests", () => {
    registerScenarioVoiceTransport();
    const transport = getDefaultVoiceTransport();
    expect(transport).toBeDefined();
    expect(transport).not.toBeNull();
    expect(transport?.buildStreamUrl("en", "fixture-session", "fixture-resume")).toBe(
      buildVoiceStreamWsUrl("en", "fixture-session", "fixture-resume"),
    );
    const provider = new PcmVoiceStreamProvider();
    provider.dispose();
  });
});
