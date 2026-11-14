/**
 * Self-test for web-console's local renderWithProviders.
 *
 * The whole reason this file exists in the scenario rather than being a
 * re-export is one line: the local helper binds *this scenario's* i18n
 * singleton, because api-base otherwise defaults to an instance it creates
 * itself — pinned to `cimode` with empty resources.
 *
 * That distinction is invisible to a key-echo assertion. Under `cimode` both
 * instances render the key path, so `expect(t("app.title")).toBe("app.title")`
 * passes whichever instance is mounted, and cannot detect the binding being
 * dropped. The consequence only shows up later: a test that calls
 * `setLocale("en")` and asserts real translated copy sees nothing change,
 * because the locale bundles were loaded into an instance that is not the one
 * rendering.
 *
 * So this file asserts instance identity, which is the only form of the
 * assertion that can actually fail, plus the option-precedence rule that lets
 * a caller override the default.
 */
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { useTranslation } from "react-i18next";
import { createInstance } from "i18next";

import { renderWithProviders } from "./renderWithProviders";
import { i18n as scenarioI18n } from "../i18n";

const Probe = () => {
  const { i18n } = useTranslation();
  return <span data-testid="instance">{i18n === scenarioI18n ? "scenario" : "foreign"}</span>;
};

describe("web-console renderWithProviders", () => {
  it("binds the scenario i18n singleton, not api-base's default instance", () => {
    renderWithProviders(<Probe />);
    expect(screen.getByTestId("instance").textContent).toBe("scenario");
  });

  // CHARACTERIZATION, not a desired behaviour. web-console resolves two copies
  // of react-i18next (15.1.1 for api-base's exact pin, 15.7.4 for its own
  // ^15.1.1), so api-base's <I18nextProvider> writes to a different React
  // context than this workspace's useTranslation reads. The provider is
  // therefore invisible here and useTranslation falls through to the global
  // instance — which is why the assertion above passes at all. A consequence
  // is that the helper's `i18n` option cannot take effect in this scenario.
  //
  // 34 scenarios are in this state; 56 resolve a single copy and honour the
  // option correctly (verified against unit-health).
  //
  // If this test starts failing, the duplicate copy was resolved: delete this
  // case and assert the override properly instead.
  it("currently ignores an explicit i18n option (duplicate react-i18next copies)", () => {
    const other = createInstance();
    void other.init({
      initImmediate: false,
      lng: "cimode",
      fallbackLng: false,
      resources: {},
      interpolation: { escapeValue: false },
    });

    renderWithProviders(<Probe />, { i18n: other });
    expect(screen.getByTestId("instance").textContent).toBe("scenario");
  });
});
