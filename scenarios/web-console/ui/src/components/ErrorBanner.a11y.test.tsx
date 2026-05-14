/**
 * ErrorBanner accessibility regression test.
 *
 * Baseline a11y scan for a high-visibility surface — anything user-facing
 * that surfaces failures *must* itself be accessible, or recovery is gated
 * on sighted-user UX. Catches missing aria-label/role/contrast regressions
 * in the banner shell without locking us into a specific markup shape.
 *
 * Feature-level a11y tests live next to the feature; cross-cutting
 * landmark/locale-switcher tests would belong with their owners.
 */
import { afterEach, beforeEach, describe, it } from "vitest";
import { render, cleanup } from "@testing-library/react";

import ErrorBanner from "./ErrorBanner";
import { setLocale } from "../i18n";
import { expectNoA11yViolations } from "../test-utils";

describe("ErrorBanner accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders without axe violations for a minimal error", async () => {
    const { container } = render(
      <ErrorBanner
        error={{ message: "Connection refused", retry: false }}
        onDismiss={() => {}}
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("renders without axe violations when retry + recovery hint are present", async () => {
    const { container } = render(
      <ErrorBanner
        error={{
          message: "Connection refused",
          retry: true,
          recovery: "Check the server status and try again.",
        }}
        onDismiss={() => {}}
        onRetry={() => {}}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
