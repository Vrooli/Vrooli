import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { makeApiError } from "../../api/client";

vi.mock("../../services/diagnostics", () => ({
  transcribe: vi.fn(),
}));

import { FileTry } from "./FileTry";
import { transcribe } from "../../services/diagnostics";

beforeEach(() => {
  vi.mocked(transcribe).mockResolvedValue({
    ok: true,
    data: {
      text: "hello world",
      trace: { providerTier: "local", providerId: "whisper", modelId: "v3", latencyMs: 7 },
    },
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("FileTry", () => {
  it("disables the transcribe action until a file is selected", () => {
    renderWithProviders(<FileTry onTrace={() => {}} />);
    expect(screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcribeAction, "i") })).toBeDisabled();
  });

  it("on success: uploads the file, renders the transcript, and emits a trace", async () => {
    const onTrace = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<FileTry onTrace={onTrace} />);
    const file = new File(["x"], "clip.wav", { type: "audio/wav" });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioFileLabel), file);
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcribeAction, "i") }));
    await waitFor(() => expect(vi.mocked(transcribe)).toHaveBeenCalledTimes(1));
    expect(await screen.findByText(/^hello world$/)).toBeInTheDocument();
    expect(onTrace).toHaveBeenCalledWith(expect.objectContaining({ providerId: "whisper" }));
  });

  it("on error: renders the ApiErrorState load-failed copy key", async () => {
    vi.mocked(transcribe).mockResolvedValue({ ok: false, error: makeApiError("internal", "transcribe-failed", 500) });
    const user = userEvent.setup();
    renderWithProviders(<FileTry onTrace={() => {}} />);
    const file = new File(["x"], "clip.wav", { type: "audio/wav" });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioFileLabel), file);
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcribeAction, "i") }));
    await waitFor(() => expect(screen.getByText(strings.apiError.loadFailedTitle)).toBeInTheDocument());
  });
});
