import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { TagInput } from "./TagInput";

afterEach(cleanup);

describe("TagInput", () => {
  it("adds a tag on Enter and reports the new list", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<TagInput tags={[]} onChange={onChange} />);
    await user.type(screen.getByTestId(selectors.dictationStudio.tagInput), "meeting{Enter}");
    expect(onChange).toHaveBeenCalledWith(["meeting"]);
  });

  it("adds a tag via the Add button", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<TagInput tags={["a"]} onChange={onChange} />);
    await user.type(screen.getByTestId(selectors.dictationStudio.tagInput), "b");
    await user.click(screen.getByRole("button", { name: strings.dictationStudio.addTag }));
    expect(onChange).toHaveBeenCalledWith(["a", "b"]);
  });

  it("ignores duplicate and blank tags", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<TagInput tags={["a"]} onChange={onChange} />);
    await user.type(screen.getByTestId(selectors.dictationStudio.tagInput), "a{Enter}");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("renders tags as badges and removes one", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<TagInput tags={["a", "b"]} onChange={onChange} />);
    const removeButtons = screen.getAllByRole("button", { name: strings.dictationStudio.removeTag });
    expect(removeButtons).toHaveLength(2);
    await user.click(removeButtons[0]!);
    expect(onChange).toHaveBeenCalledWith(["b"]);
  });
});
