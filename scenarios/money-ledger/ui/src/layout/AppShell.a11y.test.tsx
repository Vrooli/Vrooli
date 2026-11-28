/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav). Feature cards keep
 * their own a11y tests.
 *
 * The shell's cards and the write-access notice load on mount, so each test
 * renders and then waits for every query to settle: axe and the landmark
 * assertions judge the page a reader sees, not a half-loaded one.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  async function renderSettledShell() {
    const rendered = renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );
    await waitFor(() => expect(rendered.queryClient.isFetching()).toBe(0));
    return rendered;
  }

  it("renders the shell without axe violations in English", async () => {
    const { container } = await renderSettledShell();
    await expectNoA11yViolations(container);
  });

  it("exposes exactly one primary navigation landmark", async () => {
    await renderSettledShell();

    expect(screen.getAllByRole("navigation", { name: "Primary navigation" })).toHaveLength(1);
    expect(screen.getByTestId(`${selectors.layout.shell}-tabs`)).toBeInTheDocument();
  });
});
