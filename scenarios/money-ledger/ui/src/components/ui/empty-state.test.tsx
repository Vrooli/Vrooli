import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EmptyState } from "./empty-state";
import { renderWithProviders } from "../../test-utils";

describe("EmptyState", () => {
  it("renders optional icon, description, action, and merged classes", () => {
    renderWithProviders(
      <EmptyState
        title="No books"
        description="Create a book to begin."
        icon={<span aria-label="ledger icon">+</span>}
        action={<button type="button">Create book</button>}
        className="custom-state"
      />,
      { withoutRouter: true },
    );

    expect(screen.getByRole("heading", { name: /No books/ })).toBeVisible();
    expect(screen.getByText(/Create a book to begin\./)).toBeVisible();
    expect(screen.getByLabelText(/ledger icon/)).toBeVisible();
    expect(screen.getByRole("button", { name: /Create book/ })).toBeVisible();
    expect(screen.getByRole("heading").parentElement?.parentElement).toHaveClass("custom-state");
  });
});
