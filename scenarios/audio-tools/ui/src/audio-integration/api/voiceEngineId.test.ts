import { describe, expect, it, vi } from "vitest";

import { createVoiceApi } from "./voice";
import type { AudioToolsClient } from "../client";

describe("getVoiceStreamConfig engine selection", () => {
  it("retains the server-resolved engine id for route-specific capture policy", async () => {
    const client = {
      baseUrl: "http://test",
      stt: {},
      sttAdmin: { getStreamConfig: vi.fn().mockResolvedValue({ config: { engineId: "sherpa-streaming" } }) },
      tts: {},
      summarize: {},
    } as unknown as AudioToolsClient;

    await expect(createVoiceApi(client).getVoiceStreamConfig()).resolves.toMatchObject({ engineId: "sherpa-streaming" });
  });
});
