import { describe, it } from "vitest";

import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "../../test-utils";

import { CaptureGalleryPage } from "./CaptureGalleryPage";

describe("Capture gallery accessibility", () => {
  it("has no axe violations", async () => {
    const { container } = renderWithProviders(<CaptureGalleryPage />);
    await expectNoA11yViolations(container);
  });
});
