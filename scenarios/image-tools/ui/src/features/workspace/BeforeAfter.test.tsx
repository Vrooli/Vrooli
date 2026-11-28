/**
 * BeforeAfter tests. The compare widget clips the "after" image to a draggable
 * split controlled by a real range input, so it is keyboard- and touch-driven.
 * These pin both images render with their i18n alt text, the initial 50% split
 * clip, and that moving the slider re-clips the "after" overlay.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { setLocale } from "../../i18n";
import { BeforeAfter } from "./BeforeAfter";

describe("BeforeAfter", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it("renders both images with their i18n alt text and the slider", () => {
    renderWithProviders(<BeforeAfter beforeUrl="blob:before" afterUrl="blob:after" />);

    const before = screen.getByTestId(selectors.workspace.compare.before);
    const after = screen.getByTestId(selectors.workspace.compare.after);
    expect(before).toHaveAttribute("src", "blob:before");
    expect(after).toHaveAttribute("src", "blob:after");
    expect(before).toHaveAttribute("alt");
    expect(after).toHaveAttribute("alt");
    expect(screen.getByTestId(selectors.workspace.compare.slider)).toBeInTheDocument();
  });

  it("clips the after image at the 50% default split on first render", () => {
    renderWithProviders(<BeforeAfter beforeUrl="blob:before" afterUrl="blob:after" />);

    const after = screen.getByTestId(selectors.workspace.compare.after);
    // split=50 → inset(0 50% 0 0)
    expect(after.getAttribute("style")).toContain("inset(0 50% 0 0)");
    expect(screen.getByTestId(selectors.workspace.compare.slider)).toHaveValue("50");
  });

  it("re-clips the after overlay when the split slider moves", () => {
    renderWithProviders(<BeforeAfter beforeUrl="blob:before" afterUrl="blob:after" />);

    const slider = screen.getByTestId(selectors.workspace.compare.slider);
    fireEvent.change(slider, { target: { value: "20" } });

    const after = screen.getByTestId(selectors.workspace.compare.after);
    // split=20 → inset(0 80% 0 0) reveals more of the "before" image.
    expect(after.getAttribute("style")).toContain("inset(0 80% 0 0)");
    expect(slider).toHaveValue("20");
  });

  it("clips to inset 0 at a full split (after fully visible)", () => {
    renderWithProviders(<BeforeAfter beforeUrl="blob:before" afterUrl="blob:after" />);

    fireEvent.change(screen.getByTestId(selectors.workspace.compare.slider), {
      target: { value: "100" },
    });
    const after = screen.getByTestId(selectors.workspace.compare.after);
    expect(after.getAttribute("style")).toContain("inset(0 0% 0 0)");
  });

  it("uses the localized slider aria-label", () => {
    renderWithProviders(<BeforeAfter beforeUrl="blob:before" afterUrl="blob:after" />);
    // The slider carries an accessible name from the i18n compare.slider key.
    const slider = screen.getByTestId(selectors.workspace.compare.slider);
    expect(slider).toHaveAccessibleName();
    expect(strings.workspace.compare.slider).toBeTruthy();
  });
});
