import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { makeOperationInfo } from "./mocks/factories";
import { OpPicker } from "./OpPicker";

const operations = [
  makeOperationInfo({ name: "resize" }),
  makeOperationInfo({ name: "crop" }),
];

// Stays in the cimode default (see test-setup.ts) so `t()` returns the key,
// letting assertions compare against the typed `strings.*` registry.
describe("OpPicker", () => {
  afterEach(() => cleanup());

  it("renders a radio per op and marks the active one (text is the friendly label)", () => {
    renderWithProviders(
      <OpPicker operations={operations} operation="resize" onSelect={vi.fn()} />,
    );
    const resize = screen.getByTestId(selectors.workspace.opOption({ name: "resize" }));
    const crop = screen.getByTestId(selectors.workspace.opOption({ name: "crop" }));
    expect(resize).toHaveAttribute("aria-checked", "true");
    expect(crop).toHaveAttribute("aria-checked", "false");
    // cimode renders the key, so the container's text contains the label keys.
    expect(
      screen.getByTestId(selectors.workspace.operationSelect).textContent,
    ).toContain(strings.workspace.op.resize.label);
  });

  it("shows the selected op's description and emits the op name on click", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    renderWithProviders(
      <OpPicker operations={operations} operation="resize" onSelect={onSelect} />,
    );
    expect(screen.getByTestId(selectors.workspace.opDescription).textContent).toContain(
      strings.workspace.op.resize.desc,
    );
    await user.click(screen.getByTestId(selectors.workspace.opOption({ name: "crop" })));
    expect(onSelect).toHaveBeenCalledWith("crop");
  });
});
