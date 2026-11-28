import { renderWithProviders as render } from "../../../test-utils";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import TopSafeArea from "../../TopSafeArea";
import BannerRegion from "../BannerRegion";
import { INSTANT_DAMPING } from "../damping";
import type { BannerDescriptor } from "../types";

/**
 * The notch has one owner. `TopSafeArea`'s status-bar strip reserves it; the
 * banner region sits directly beneath and must not reserve it a second time,
 * or the banner lands a whole status-bar height below the strip on iOS.
 */
describe("BannerRegion safe area", () => {
  it("leaves the top safe area to TopSafeArea's strip", () => {
    const notice: BannerDescriptor = {
      id: "notice", testId: "notice-banner", tone: "warning", priority: 50, title: "notice",
    };
    render(
      <TopSafeArea testId="edge">
        <BannerRegion banners={[notice]} damping={INSTANT_DAMPING} />
      </TopSafeArea>,
    );

    expect(screen.getByTestId("edge-fill")).toHaveAttribute("data-rcl-status-fill-strip");
    const region = screen.getByTestId("banner-region");
    expect(region.className).not.toMatch(/safe-area|safe-top/);
    expect(region.getAttribute("style") ?? "").not.toMatch(/safe-area|safe-top/);
  });
});
