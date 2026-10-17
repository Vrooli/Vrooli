import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { NotFoundPage } from "./NotFoundPage";

describe("NotFoundPage accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<NotFoundPage />);
    await expectNoA11yViolations(container);
  });
});
