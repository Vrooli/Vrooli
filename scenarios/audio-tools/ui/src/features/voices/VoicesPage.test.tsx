import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeApiError } from "../../api/client";
import { strings } from "../../consts/strings";

vi.mock("../../services/settings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../services/settings")>();
  return { ...actual, getVoiceOverrides: vi.fn(), setVoiceOverride: vi.fn() };
});

vi.mock("../../services/tts", () => ({
  getStatus: vi.fn(),
}));

vi.mock("../../components/ui/toast", () => ({
  pushToast: vi.fn(),
  dismissToast: vi.fn(),
  Toaster: () => null,
}));

import { VoicesPage } from "./VoicesPage";
import { getVoiceOverrides, setVoiceOverride } from "../../services/settings";
import { getStatus } from "../../services/tts";
import { pushToast } from "../../components/ui/toast";

const CANONICAL = "voice.feminine.warm";
const ADAPTER = "local:kokoro";

const happyOverrides = {
  ok: true as const,
  data: [{ canonicalVoice: CANONICAL, tierProvider: ADAPTER, adapterVoice: "af_heart" }],
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

function voiceRow() {
  return screen.findByRole("row", { name: new RegExp(CANONICAL.replace(/\./g, "\\.")) });
}

async function openEditor(user: ReturnType<typeof userEvent.setup>, row: HTMLElement) {
  const editButtons = within(row).getAllByRole("button", { name: strings.voices.editAria });
  // TTS_ADAPTERS[0] is "local:kokoro", the cell that holds the seeded override.
  await user.click(editButtons[0] as HTMLElement);
  return within(row).getByRole("textbox", { name: strings.voices.editorAria });
}

beforeEach(() => {
  vi.mocked(getVoiceOverrides).mockResolvedValue(happyOverrides);
  vi.mocked(getStatus).mockResolvedValue(happyStatus);
  vi.mocked(setVoiceOverride).mockResolvedValue({ ok: true, data: [] });
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

  it("seeds the editor with the existing override and saves a new value", async () => {
    vi.mocked(setVoiceOverride).mockResolvedValue({
      ok: true,
      data: [{ canonicalVoice: CANONICAL, tierProvider: ADAPTER, adapterVoice: "af_bella" }],
    });
    const user = userEvent.setup();
    renderWithProviders(<VoicesPage />);

    const row = await voiceRow();
    const input = await openEditor(user, row);
    expect(input).toHaveValue("af_heart");

    await user.clear(input);
    await user.type(input, "  af_bella  ");
    await user.click(
      within(row).getByRole("button", { name: strings.voices.saveAria }),
    );

    await waitFor(() => {
      expect(vi.mocked(setVoiceOverride)).toHaveBeenCalledWith(CANONICAL, ADAPTER, "af_bella");
    });
    // The server-returned list replaces the cached one, so the new value wins.
    await waitFor(() => {
      expect(within(row).getByText(/af_bella/)).toBeInTheDocument();
    });
    expect(within(row).queryByRole("textbox")).not.toBeInTheDocument();
    expect(vi.mocked(pushToast)).toHaveBeenCalledWith(
      expect.objectContaining({ title: strings.voices.saved }),
    );
  });

  it("deletes the override when saved with an empty value and shows default", async () => {
    const user = userEvent.setup();
    renderWithProviders(<VoicesPage />);

    const row = await voiceRow();
    const input = await openEditor(user, row);
    expect(input).toHaveValue("af_heart");

    await user.clear(input);
    await user.click(
      within(row).getByRole("button", { name: strings.voices.saveAria }),
    );

    await waitFor(() => {
      expect(vi.mocked(setVoiceOverride)).toHaveBeenCalledWith(CANONICAL, ADAPTER, "");
    });
    await waitFor(() => {
      // Every adapter column in this row now falls back to the default.
      expect(within(row).getAllByText(strings.common.default)).toHaveLength(3);
    });
    expect(within(row).queryByText(/af_heart/)).not.toBeInTheDocument();
    expect(vi.mocked(pushToast)).toHaveBeenCalledWith(
      expect.objectContaining({ title: strings.voices.deleted }),
    );
  });

  it("cancel closes the editor without calling the service", async () => {
    const user = userEvent.setup();
    renderWithProviders(<VoicesPage />);

    const row = await voiceRow();
    const input = await openEditor(user, row);
    await user.clear(input);
    await user.type(input, "af_sky");
    await user.click(
      within(row).getByRole("button", { name: strings.voices.cancelAria }),
    );

    expect(within(row).queryByRole("textbox")).not.toBeInTheDocument();
    expect(within(row).getByText(/af_heart/)).toBeInTheDocument();
    expect(vi.mocked(setVoiceOverride)).not.toHaveBeenCalled();
  });

  it("Escape cancels the edit without calling the service", async () => {
    const user = userEvent.setup();
    renderWithProviders(<VoicesPage />);

    const row = await voiceRow();
    const input = await openEditor(user, row);
    await user.type(input, "{Escape}");

    expect(within(row).queryByRole("textbox")).not.toBeInTheDocument();
    expect(vi.mocked(setVoiceOverride)).not.toHaveBeenCalled();
  });

  it("surfaces an inline error and keeps the editor open when saving fails", async () => {
    vi.mocked(setVoiceOverride).mockResolvedValue({
      ok: false,
      error: makeApiError("internal", "override-failed", 500),
    });
    const user = userEvent.setup();
    renderWithProviders(<VoicesPage />);

    const row = await voiceRow();
    const input = await openEditor(user, row);
    await user.clear(input);
    await user.type(input, "af_bella");
    await user.click(
      within(row).getByRole("button", { name: strings.voices.saveAria }),
    );

    const alert = await within(row).findByRole("alert");
    expect(alert).toHaveTextContent(strings.voices.saveError);
    expect(within(row).getByRole("textbox")).toHaveValue("af_bella");
    expect(vi.mocked(pushToast)).toHaveBeenCalledWith(
      expect.objectContaining({ title: strings.voices.saveError }),
    );
  });

  it("keeps only one cell editable at a time", async () => {
    renderWithProviders(<VoicesPage />);

    const user = userEvent.setup();
    const firstRow = await voiceRow();
    await openEditor(user, firstRow);

    const secondRow = await screen.findByRole("row", {
      name: /voice\.feminine\.neutral/,
    });
    const secondEditButtons = within(secondRow).getAllByRole("button", {
      name: strings.voices.editAria,
    });
    await user.click(secondEditButtons[0] as HTMLElement);

    expect(screen.getAllByRole("textbox")).toHaveLength(1);
    expect(within(secondRow).getByRole("textbox")).toBeInTheDocument();
    expect(within(firstRow).queryByRole("textbox")).not.toBeInTheDocument();
  });
});
