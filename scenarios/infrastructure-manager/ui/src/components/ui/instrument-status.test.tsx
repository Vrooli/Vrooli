import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings.generated";
import { RatioConfidence, StatusToken, TrustTriple } from "./instrument-status";

/**
 * Tests run in i18next `cimode`, so `t()` returns the KEY rather than the
 * interpolated copy. Asserting against `strings.*` therefore proves the right
 * key is wired without pinning any wording, in any locale.
 */
describe("instrument status language", () => {
  it("keeps the trust distribution and both denominators atomic", () => {
    renderWithProviders(<TrustTriple value={{ distribution: { VALID: 4, GHOST: 1 }, checked: 5, total: 7 }} />);
    const region = screen.getByRole("region", { name: strings.instrument.trustDistributionLabel });
    expect(region).toHaveTextContent(strings.instrument.trust.valid);
    expect(region).toHaveTextContent("4");
    expect(region).toHaveTextContent(strings.instrument.checkedOf);
  });

  it("names the empty distribution rather than rendering a zero", () => {
    renderWithProviders(<TrustTriple value={{ distribution: {}, checked: 0, total: 0 }} />);
    const region = screen.getByRole("region", { name: strings.instrument.trustDistributionLabel });
    expect(region).toHaveTextContent(strings.instrument.noVerdicts);
  });

  it("renders ratio confidence and its rationale together", () => {
    renderWithProviders(<RatioConfidence value={{ ratio: 0.5, confidence: "SKETCH", rationale: "obligation list is not authored" }} />);
    const region = screen.getByRole("region", { name: strings.instrument.ratioLabel });
    expect(region).toHaveTextContent("50%");
    expect(region).toHaveTextContent(strings.instrument.confidence.sketch);
    expect(region).toHaveTextContent(/obligation list/);
  });

  it("renders an em dash for a ratio that could not be computed, never a zero", () => {
    renderWithProviders(
      <RatioConfidence value={{ ratio: null, confidence: "PARTIAL", rationale: "the space could not be read" }} />,
    );
    const region = screen.getByRole("region", { name: strings.instrument.ratioUncomputedLabel });
    expect(region).toHaveTextContent("—");
    expect(region).not.toHaveTextContent("0");
    // The confidence and its rationale stay adjacent even when the figure is
    // absent: an uncomputed ratio still has a denominator claim to qualify it.
    expect(region).toHaveTextContent(strings.instrument.confidence.partial);
    expect(region).toHaveTextContent(/could not be read/);
  });

  it("retains a non-colour mark for every supported verdict", () => {
    const verdicts = ["VALID", "GHOST", "SATURATED", "SHELVED", "UNIT_MISMATCH", "UNAVAILABLE", "UNTRUSTED", "IN_BAND", "OUT_OF_BAND", "PENDING_SUSTAIN", "NEEDS_BASELINE", "NOT_EVALUATED"] as const;
    const { container } = renderWithProviders(<>{verdicts.map((verdict) => <StatusToken key={verdict} verdict={verdict} />)}</>);
    for (const verdict of verdicts) {
      const token = container.querySelector(`.status-token--${verdict.toLowerCase().replace(/_/g, "-")}`);
      expect(token).toBeVisible();
      // A mark AND a label, never one without the other: two child spans,
      // the first of them hidden from assistive technology.
      expect(token?.childElementCount).toBe(2);
      expect(token?.querySelector("[aria-hidden='true']")?.textContent).toBeTruthy();
    }
  });
});
