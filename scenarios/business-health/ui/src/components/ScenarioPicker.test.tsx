import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { ScenarioPicker } from "./ScenarioPicker";

const noop = () => {};

describe("ScenarioPicker", () => {
  afterEach(() => cleanup());

  it("emits the trimmed slug on submit", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    renderWithProviders(
      <ScenarioPicker onSelect={onSelect} recents={[]} onClearRecents={noop} />,
    );
    await user.type(screen.getByTestId(selectors.scenarioPicker.input), "  web-console  ");
    await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
    expect(onSelect).toHaveBeenCalledWith("web-console");
  });

  it("does not emit for a blank input", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    renderWithProviders(
      <ScenarioPicker onSelect={onSelect} recents={[]} onClearRecents={noop} />,
    );
    await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("renders recent chips and selects one on click", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    renderWithProviders(
      <ScenarioPicker onSelect={onSelect} recents={["fleet-a", "fleet-b"]} onClearRecents={noop} />,
    );
    await user.click(screen.getByTestId(selectors.scenarioPicker.recentItem({ scenario: "fleet-b" })));
    expect(onSelect).toHaveBeenCalledWith("fleet-b");
  });

  it("clears recents via the clear affordance", async () => {
    const user = userEvent.setup();
    const onClear = vi.fn();
    renderWithProviders(
      <ScenarioPicker onSelect={noop} recents={["x"]} onClearRecents={onClear} />,
    );
    await user.click(screen.getByTestId(selectors.scenarioPicker.clear));
    expect(onClear).toHaveBeenCalled();
  });
});
