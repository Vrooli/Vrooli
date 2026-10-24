import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { TestAppRouter } from "../app/routes";

describe("AppShell accessibility", () => {
  afterEach(() => cleanup());

  it("renders the shell without axe violations", async () => {
    const { container } = renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );
    await expectNoA11yViolations(container);
    expect(container).toBeTruthy();
  });

  it("exposes the primary navigation", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getAllByRole("navigation").length).toBeGreaterThan(0);
  });
});
