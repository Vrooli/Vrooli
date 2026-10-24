import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { makeApiError } from "../../api/client";

vi.mock("../../services/tts", () => ({
  synthesize: vi.fn(),
  listVoices: vi.fn(),
}));

import { SynthesizeTryIt } from "./SynthesizeTryIt";
import { listVoices, synthesize } from "../../services/tts";

beforeEach(() => {
  vi.mocked(listVoices).mockResolvedValue({ ok: true, data: [] });
  vi.mocked(synthesize).mockResolvedValue({
    ok: true,
    data: {
      audio: new Uint8Array([1, 2, 3, 4]),
      contentType: "audio/wav",
      providerTier: "local",
      providerId: "kokoro",
      modelId: "v1",
      latencyMs: 8,
    },
  });
  // jsdom doesn't implement URL.createObjectURL
  Object.defineProperty(URL, "createObjectURL", { value: vi.fn(() => "blob:fake"), writable: true });
  Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), writable: true });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SynthesizeTryIt", () => {
  it("disables the synthesize action until text is entered", () => {
    renderWithProviders(<SynthesizeTryIt onTrace={() => {}} />);
    const btn = screen.getByRole("button", { name: new RegExp(strings.diagnostics.synthesizeAction, "i") });
    expect(btn).toBeDisabled();
  });

  it("on success: invokes synthesize and emits a trace", async () => {
    const onTrace = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<SynthesizeTryIt onTrace={onTrace} />);
    await user.type(screen.getByPlaceholderText(strings.diagnostics.synthesizeTextPlaceholder), "hello");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.synthesizeAction, "i") }));
    await waitFor(() => expect(vi.mocked(synthesize)).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(onTrace).toHaveBeenCalledWith(expect.objectContaining({ providerId: "kokoro" })));
  });

  it("on error: renders the ApiErrorState load-failed copy key", async () => {
    vi.mocked(synthesize).mockResolvedValue({ ok: false, error: makeApiError("internal", "synth-failed", 500) });
    const user = userEvent.setup();
    renderWithProviders(<SynthesizeTryIt onTrace={() => {}} />);
    await user.type(screen.getByPlaceholderText(strings.diagnostics.synthesizeTextPlaceholder), "hello");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.synthesizeAction, "i") }));
    await waitFor(() => expect(screen.getByText(strings.apiError.loadFailedTitle)).toBeInTheDocument());
  });
});
