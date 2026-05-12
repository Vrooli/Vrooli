/**
 * AppShell accessibility regression test.
 *
 * Renders the shell directly instead of `<App />` so this test owns only
 * cross-cutting layout, headings, locale controls, and landmarks. Feature
 * cards carry their own a11y tests beside the feature they exercise.
 */
import { afterEach, beforeEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { AppShell } from "./AppShell";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <AppShell>
        <main aria-label="Feature content">Stable feature slot</main>
      </AppShell>,
    );

    await expectNoA11yViolations(container);
  });
});
