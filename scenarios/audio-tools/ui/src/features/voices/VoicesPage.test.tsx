import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeApiError } from "../../api/client";
import { strings } from "../../consts/strings";

vi.mock("../../services/settings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../services/settings")>();
  return { ...actual, getVoiceOverrides: vi.fn() };
});

vi.mock("../../services/tts", () => ({
  getStatus: vi.fn(),
}));

import { VoicesPage } from "./VoicesPage";
import { getVoiceOverrides } from "../../services/settings";
import { getStatus } from "../../services/tts";

const happyOverrides = {
  ok: true as const,
  data: [
    { canonicalVoice: "voice.feminine.warm", tierProvider: "local:kokoro", adapterVoice: "af_heart" },
  ],
};

const happyStatus = {
  ok: true as const,
  data: {
    capability: "tts",
    capabilityLabel: "TTS",
    availability: [
      { tier: "local", providerId: "kokoro", available: true },
      { tier: "byok", providerId: "openai-tts", available: false },
    ],
  },
};

beforeEach(() => {
  vi.mocked(getVoiceOverrides).mockResolvedValue(happyOverrides);
  vi.mocked(getStatus).mockResolvedValue(happyStatus);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("VoicesPage", () => {
  it("renders happy data: matrix, adapter override, and availability probes", async () => {
    renderWithProviders(<VoicesPage />);
    expect(await screen.findByText(strings.voices.title)).toBeInTheDocument();
    expect(await screen.findByText(/af_heart/)).toBeInTheDocument();
    expect(screen.getAllByText(/kokoro/).length).toBeGreaterThan(0);
  });

  it("renders empty/default placeholders when there are no overrides", async () => {
    vi.mocked(getVoiceOverrides).mockResolvedValue({ ok: true, data: [] });
    renderWithProviders(<VoicesPage />);
    // All matrix cells fall back to the "default" placeholder when no
    // override matches the (canonical, adapter) tuple.
    await waitFor(() => {
      expect(screen.getAllByText(strings.common.default).length).toBeGreaterThan(0);
    });
  });

  it("renders error state when getVoiceOverrides fails", async () => {
    vi.mocked(getVoiceOverrides).mockResolvedValue({
      ok: false,
      error: makeApiError("internal", "voices-failed", 500),
    });
    renderWithProviders(<VoicesPage />);
    await waitFor(() => expect(screen.getByText(/voices-failed/)).toBeInTheDocument());
  });

  it("calls getStatus exactly once on mount", async () => {
    renderWithProviders(<VoicesPage />);
    await waitFor(() => {
      expect(vi.mocked(getStatus)).toHaveBeenCalledTimes(1);
    });
    // react-query forwards a query-function context object; the production
    // queryFn ignores it, so this test pins the call-count contract only.
  });
});
