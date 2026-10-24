import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { makeApiError } from "../../api/client";

vi.mock("../../services/diagnostics", () => ({
  summarize: vi.fn(),
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

  it("on success: invokes summarize with the typed level and renders the summary", async () => {
    const onTrace = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<SummarizeTryIt onTrace={onTrace} />);
    await user.type(screen.getByLabelText(strings.diagnostics.summarizeInputLabel), "text to summarize");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.summarizeAction, "i") }));
    await waitFor(() => expect(vi.mocked(summarize)).toHaveBeenCalledWith("text to summarize", "moderate"));
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
