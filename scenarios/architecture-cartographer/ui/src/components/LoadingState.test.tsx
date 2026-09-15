import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { LoadingState } from "./LoadingState";

describe("LoadingState", () => {
  it("renders aria-label and role=status", () => {
    render(<LoadingState label="Loading…" />);
    const el = screen.getByRole("status");
    expect(el).toHaveAttribute("aria-label", "Loading…");
    expect(el).toHaveAttribute("aria-live", "polite");
  });

  it("uses the shared loading-state testid", () => {
    render(<LoadingState label="please wait" />);
    expect(screen.getByTestId(selectors.shared.loadingState.root)).toBeInTheDocument();
  });
});
