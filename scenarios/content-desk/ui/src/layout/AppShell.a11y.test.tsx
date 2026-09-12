/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";
import { Providers } from "../app/providers";
import { selectors } from "../consts/selectors";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <Providers><TestAppRouter initialEntries={["/"]} /></Providers>,
      { withoutRouter: true },
    );
    await expectNoA11yViolations(container);
  });

  it("exposes exactly one primary navigation landmark", () => {
    renderWithProviders(
      <Providers><TestAppRouter initialEntries={["/"]} /></Providers>,
      { withoutRouter: true },
    );

    expect(screen.getByRole("navigation", { name: "Primary navigation" })).toBeInTheDocument();
    expect(screen.getByTestId(`${selectors.layout.shell}-tabs`)).toBeInTheDocument();
  });
});
