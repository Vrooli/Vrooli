import { screen } from "@testing-library/react";
import { createElement } from "react";
import { describe, it, expect } from "vitest";
import { InsufficientDataCard } from "../../src/components/stats/InsufficientDataCard.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

describe("InsufficientDataCard", () => {
  it("renders the label, reason, and the have/required ratio", () => {
    renderWithProviders(
      createElement(InsufficientDataCard, {
        label: "Fallback rate",
        reason: "Need at least 5 fallback events",
        have: 2,
        required: 5,
        testId: "insufficient",
      }),
    );
    expect(screen.getByTestId("insufficient")).toBeTruthy();
    expect(screen.getByText("Fallback rate")).toBeTruthy();
    expect(screen.getByText("Not enough data yet")).toBeTruthy();
    expect(screen.getByText(/Need at least 5 fallback events/)).toBeTruthy();
    expect(screen.getByText(/\(2 of 5 needed\)/)).toBeTruthy();
  });

  it("renders without the ratio when have/required are omitted", () => {
    renderWithProviders(
      createElement(InsufficientDataCard, {
        label: "Health",
        reason: "No health events recorded",
      }),
    );
    expect(screen.getByText("Health")).toBeTruthy();
    expect(screen.queryByText(/needed/)).toBeNull();
  });
});
