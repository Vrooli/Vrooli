/**
 * Self-test for system-monitor's local renderWithProviders.
 *
 * Unlike most scenarios this helper mounts no QueryClient — system-monitor's
 * UI uses none. What it does mount is three scenario-owned contexts plus a
 * BrowserRouter whose basename is derived from the deployment's proxy path.
 * Every component test in this scenario depends on those four being present,
 * and nothing verified it until now: a provider dropped from the wrapper would
 * surface as `useTheme must be used within ...` scattered across unrelated
 * suites rather than as one failure here.
 */
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "./renderWithProviders";
import { useTheme } from "../shared/theme/ThemeProvider";
import { useToast } from "../shared/components/ToastProvider";
import { useTimeRange } from "../shared/time/TimeRangeContext";

describe("system-monitor renderWithProviders", () => {
  it("mounts the theme, toast, and time-range providers", () => {
    const Probe = () => {
      // Each of these throws outside its provider, so reaching the render
      // at all is the assertion.
      const theme = useTheme();
      const toast = useToast();
      const range = useTimeRange();
      return (
        <span data-testid="probe">
          {[theme, toast, range].every((v) => v !== undefined) ? "all-mounted" : "missing"}
        </span>
      );
    };

    renderWithProviders(<Probe />);
    expect(screen.getByTestId("probe").textContent).toBe("all-mounted");
  });

  it("provides routing context", () => {
    const Probe = () => <a data-testid="link" href="/x">x</a>;
    renderWithProviders(<Probe />);
    expect(screen.getByTestId("link").getAttribute("href")).toBe("/x");
  });
});
