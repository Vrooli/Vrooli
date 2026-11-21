import { describe, expect, it } from "vitest";
import { getDefaultVoiceTransport } from "@vrooli/audio-capture-browser";
import { registerScenarioVoiceTransport } from "./index";

describe("scenario voice bootstrap", () => {
  it("registers the Connect bidi adapter without opening a capture stream", () => {
    registerScenarioVoiceTransport();
    const transport = getDefaultVoiceTransport();
    expect(transport).not.toBeNull();
    expect(typeof transport?.openStream).toBe("function");
  });
});
