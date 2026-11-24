import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { makeApiError } from "../../api/client";

vi.mock("../../services/diagnostics", () => ({
  summarize: vi.fn(),
  listSummarizeModels: vi.fn().mockResolvedValue({
    ok: true,
    data: [
      { id: "summarize.audio_tools", name: "Qwen3 1.7B — compact default", notes: "Short-report evidence only.", defaultEligible: true },
      { id: "summarize.audio_tools.quality", name: "Qwen3.5 4B — quality alternative", notes: "The 1.7B default kept 14/22 facts on one 300-word report.", defaultEligible: true },
    ],
  }),
}));

import { SummarizeTryIt } from "./SummarizeTryIt";
import { summarize } from "../../services/diagnostics";

beforeEach(() => {
  vi.mocked(summarize).mockResolvedValue({
    ok: true,
    data: {
      text: "the summary",
      promptTokens: 4,
      outputTokens: 3,
      trace: { providerTier: "local", providerId: "ollama", modelId: "llama3", latencyMs: 11 },
    },
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SummarizeTryIt", () => {
  it("disables the action until text is entered", () => {
    renderWithProviders(<SummarizeTryIt onTrace={() => {}} />);
    const btn = screen.getByRole("button", { name: new RegExp(strings.diagnostics.summarizeAction, "i") });
    expect(btn).toBeDisabled();
  });

  it("keeps model choice separate from summary length and sends both selections", async () => {
    const onTrace = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<SummarizeTryIt onTrace={onTrace} />);
    expect(await screen.findByRole("combobox", { name: strings.diagnostics.summarizeModelLabel })).toHaveValue("summarize.audio_tools");
    await user.selectOptions(screen.getByRole("combobox", { name: strings.diagnostics.summarizeModelLabel }), "summarize.audio_tools.quality");
    expect(screen.getByText(/14\/22 facts on one 300-word report/)).toBeInTheDocument();
    await user.selectOptions(screen.getByRole("combobox", { name: strings.diagnostics.levelLabel }), "heavy");
    await user.type(screen.getByLabelText(strings.diagnostics.summarizeInputLabel), "text to summarize");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.summarizeAction, "i") }));
    await waitFor(() => expect(vi.mocked(summarize)).toHaveBeenCalledWith("text to summarize", "heavy", "summarize.audio_tools.quality"));
    expect(await screen.findByText(/^the summary$/)).toBeInTheDocument();
    expect(onTrace).toHaveBeenCalledWith(expect.objectContaining({ providerId: "ollama" }));
  });

  it("on error: renders the ApiErrorState load-failed copy key", async () => {
    vi.mocked(summarize).mockResolvedValue({ ok: false, error: makeApiError("internal", "boom", 500) });
    const user = userEvent.setup();
    renderWithProviders(<SummarizeTryIt onTrace={() => {}} />);
    await user.type(screen.getByLabelText(strings.diagnostics.summarizeInputLabel), "x");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.summarizeAction, "i") }));
    await waitFor(() => expect(screen.getByText(strings.apiError.loadFailedTitle)).toBeInTheDocument());
  });
});
