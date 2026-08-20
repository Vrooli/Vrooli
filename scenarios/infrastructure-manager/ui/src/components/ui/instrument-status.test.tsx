import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { RatioConfidence, StatusToken, TrustTriple } from "./instrument-status";

describe("instrument status language", () => {
  it("keeps the trust distribution and both denominators atomic", () => {
    renderWithProviders(<TrustTriple value={{ distribution: { VALID: 4, GHOST: 1 }, checked: 5, total: 7 }} />);
    expect(screen.getByRole("region", { name: "Trust distribution" })).toHaveTextContent("4 Valid");
    expect(screen.getByRole("region", { name: "Trust distribution" })).toHaveTextContent("5 checked of 7 readings");
  });

  it("renders ratio confidence and its rationale together", () => {
    renderWithProviders(<RatioConfidence value={{ ratio: 0.5, confidence: "SKETCH", rationale: "obligation list is not authored" }} />);
    expect(screen.getByRole("region", { name: /Coverage 50%/ })).toHaveTextContent("SKETCH");
    expect(screen.getByRole("region", { name: /Coverage 50%/ })).toHaveTextContent("obligation list");
  });

  it("retains a non-colour mark for every supported verdict", () => {
    const verdicts = ["VALID", "GHOST", "SATURATED", "SHELVED", "UNIT_MISMATCH", "UNAVAILABLE", "UNTRUSTED", "IN_BAND", "OUT_OF_BAND", "PENDING_SUSTAIN", "NEEDS_BASELINE", "NOT_EVALUATED"] as const;
    renderWithProviders(<>{verdicts.map((verdict) => <StatusToken key={verdict} verdict={verdict} />)}</>);
    for (const verdict of verdicts) expect(screen.getByText(/./, { selector: `.status-token--${verdict.toLowerCase().replace(/_/g, "-")}` })).toBeVisible();
  });
});
