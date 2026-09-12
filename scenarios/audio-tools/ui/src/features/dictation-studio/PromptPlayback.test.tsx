import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { PromptPlayback } from "./PromptPlayback";

afterEach(cleanup);

describe("PromptPlayback", () => {
  it("renders the prompt and reports edits", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<PromptPlayback prompt="" onChange={onChange} />);
    await user.type(screen.getByTestId(selectors.dictationStudio.promptInput), "r");
    expect(onChange).toHaveBeenCalledWith("r");
  });

  it("disables playback when no reference audio is provided", () => {
    renderWithProviders(<PromptPlayback prompt="read me" onChange={() => {}} />);
    expect(screen.getByRole("button", { name: strings.dictationStudio.playPrompt })).toBeDisabled();
  });

  it("enables playback when a reference URL is provided", () => {
    renderWithProviders(<PromptPlayback prompt="read me" onChange={() => {}} audioUrl="blob:abc" />);
    expect(screen.getByRole("button", { name: strings.dictationStudio.playPrompt })).not.toBeDisabled();
  });
});
