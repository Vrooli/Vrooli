import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { TranscriptEditor } from "./TranscriptEditor";

afterEach(cleanup);

describe("TranscriptEditor", () => {
  it("renders the current value", () => {
    renderWithProviders(<TranscriptEditor value="hello there" onChange={() => {}} />);
    expect(screen.getByTestId(selectors.dictationStudio.transcriptEditor)).toHaveValue("hello there");
  });

  it("reports edits via onChange", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<TranscriptEditor value="" onChange={onChange} />);
    await user.type(screen.getByTestId(selectors.dictationStudio.transcriptEditor), "x");
    expect(onChange).toHaveBeenCalledWith("x");
  });
});
