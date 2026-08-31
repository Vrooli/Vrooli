import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EmptyState } from "./EmptyState.tsx";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

describe("EmptyState", () => {
  it("renders optional description, icon, action, and custom class", () => {
    renderWithProviders(
      <EmptyState
        title="Nothing here"
        description="Try another filter"
        icon={<span data-testid="empty-icon">!</span>}
        action={<button type="button">Retry</button>}
        className="custom-empty"
      />,
    );
    expect(screen.getByText(/Nothing here/)).toBeInTheDocument();
    expect(screen.getByText(/Try another filter/)).toBeInTheDocument();
    expect(screen.getByTestId("empty-icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Retry/ })).toBeInTheDocument();
    expect(
      screen.getByText(/Nothing here/).parentElement?.parentElement,
    ).toHaveClass("custom-empty");
  });
});
