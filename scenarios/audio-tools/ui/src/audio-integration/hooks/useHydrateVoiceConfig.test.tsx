import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const getConfigMock = vi.hoisted(() => vi.fn());
const unavailableMock = vi.hoisted(() => vi.fn());

vi.mock("../api/voice", () => ({ getVoiceStreamConfig: getConfigMock }));
vi.mock("../client", () => ({ useAudioToolsUnavailableReason: unavailableMock }));

import { useHydrateVoiceConfig } from "./useHydrateVoiceConfig";
import { _resetVoiceConfigForTesting, useVoiceConfigStore } from "./useVoiceConfigStore";

describe("useHydrateVoiceConfig", () => {
  afterEach(() => {
    vi.clearAllMocks();
    _resetVoiceConfigForTesting();
  });

  it("hydrates the local voice configuration from the API once", async () => {
    unavailableMock.mockReturnValue(undefined);
    getConfigMock.mockResolvedValue({ vadSilenceMs: 900, segmentSilenceMs: 1200, persistentMode: true, wakeWordEnabled: true });
    renderHook(() => useHydrateVoiceConfig());

    await waitFor(() => expect(getConfigMock).toHaveBeenCalledOnce());
    expect(useVoiceConfigStore.getState()).toMatchObject({ vadSilenceTimeoutMs: 900, segmentSilenceMs: 900, persistentMode: true, wakeWordEnabled: true });
  });

  it("does not request configuration while the client is unavailable", () => {
    unavailableMock.mockReturnValue("offline");
    renderHook(() => useHydrateVoiceConfig());
    expect(getConfigMock).not.toHaveBeenCalled();
  });
});
