import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { LogoTargetPreview } from "./LogoTargetPreview";

describe("LogoTargetPreview", () => {
  beforeEach(() => {
    setLocale("en");
  });
  afterEach(() => {
    cleanup();
  });

  it("previews every declared size on light and dark grounds", () => {
    renderWithProviders(<LogoTargetPreview brandId="brand-1" markAssetId="asset-mark" />);
    // 5 sizes x 2 grounds.
    expect(screen.getAllByTestId(selectors.logo.previewTile)).toHaveLength(10);
    expect(screen.getAllByText("16 px")).toHaveLength(2);
    expect(screen.getAllByText("512 px")).toHaveLength(2);
    expect(screen.getAllByAltText("")).toHaveLength(10);
  });

  it("shows the maskable safe-zone overlay", () => {
    renderWithProviders(<LogoTargetPreview brandId="brand-1" markAssetId="asset-mark" />);
    expect(screen.getByTestId(selectors.logo.previewMaskable)).toBeInTheDocument();
  });

  it("prompts for a pick when the brand has no mark", () => {
    renderWithProviders(<LogoTargetPreview brandId="brand-1" markAssetId="" />);
    expect(screen.getByTestId(selectors.logo.preview)).toHaveTextContent(
      "Pick a candidate to preview the applied icon.",
    );
    expect(screen.queryByTestId(selectors.logo.previewTile)).not.toBeInTheDocument();
  });
});
