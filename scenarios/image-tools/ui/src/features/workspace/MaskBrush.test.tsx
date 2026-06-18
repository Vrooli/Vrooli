/**
 * MaskBrush tests. The pointer-painting path needs a real 2D canvas (exercised
 * in the browser BAS journey), so these cover the parts that run headless: the
 * accessible file-upload fallback (the non-pointer path), clearing, and status.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { MaskBrush } from "./MaskBrush";

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("MaskBrush", () => {
  it("renders the paint canvas and the upload fallback for a loaded image", () => {
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={vi.fn()} />);
    expect(screen.getByTestId(selectors.workspace.mask.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.mask.canvas)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.mask.upload)).toBeInTheDocument();
  });

  it("emits the uploaded mask file via the accessible fallback", async () => {
    const user = userEvent.setup();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);

    const mask = new File(["m"], "mask.png", { type: "image/png" });
    await user.upload(screen.getByTestId(selectors.workspace.mask.upload), mask);

    expect(onMask).toHaveBeenCalledWith(mask);
    expect(screen.getByTestId(selectors.workspace.mask.status)).toHaveTextContent("mask.png");
  });

  it("clears the mask back to null", async () => {
    const user = userEvent.setup();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);

    await user.click(screen.getByTestId(selectors.workspace.mask.clear));
    expect(onMask).toHaveBeenCalledWith(null);
  });
});
