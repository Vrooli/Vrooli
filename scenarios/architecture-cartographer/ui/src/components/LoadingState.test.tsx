import { renderWithProviders } from "../test-utils";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { LoadingState } from "./LoadingState";

describe("LoadingState", () => {
  it("renders aria-label and role=status", () => {
    renderWithProviders(<LoadingState label="Loading…" />);
    const el = screen.getByRole("status");
    expect(el).toHaveAttribute("aria-label", "Loading…");
    expect(el).toHaveAttribute("aria-live", "polite");
  });

  it("uses the shared loading-state testid", () => {
    renderWithProviders(<LoadingState label="please wait" />);
    expect(screen.getByTestId(selectors.shared.loadingState.root)).toBeInTheDocument();
  });
});
