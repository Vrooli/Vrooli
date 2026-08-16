import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EmptyState } from "./empty-state";
import { renderWithProviders } from "../../test-utils";

describe("EmptyState", () => {
  it("renders optional icon, description, action, and merged classes", () => {
    renderWithProviders(
      <EmptyState
        title="No proposals"
        description="Wait for an agent proposal to arrive."
        icon={<span aria-label="offer icon">+</span>}
        action={<button type="button">Refresh</button>}
        className="custom-state"
      />,
      { withoutRouter: true },
    );

    expect(screen.getByRole("heading", { name: /No proposals/ })).toBeVisible();
    expect(screen.getByText(/Wait for an agent proposal to arrive\./)).toBeVisible();
    expect(screen.getByLabelText(/offer icon/)).toBeVisible();
    expect(screen.getByRole("button", { name: /Refresh/ })).toBeVisible();
    expect(screen.getByRole("heading").parentElement?.parentElement).toHaveClass("custom-state");
  });
});
