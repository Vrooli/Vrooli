import { cleanup } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { Layout } from "./Layout";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "@vrooli/api-base/testing";

describe("Layout accessibility", () => {
  afterEach(() => cleanup());

  it("keeps the navigation shell free of axe violations", async () => {
    const { container } = renderWithProviders(
      <Layout>
        <h2>Evidence review</h2>
        <p>Review the latest target verdicts.</p>
      </Layout>,
      { route: "/evidence" },
    );
    expect(container.querySelector("main")).toBeTruthy();
    await expectNoA11yViolations(container);
  });
});
