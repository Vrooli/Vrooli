import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { EmptyState } from "./EmptyState.tsx";

describe("EmptyState", () => {
  it("renders optional icon, description, action, and custom class", () => {
    renderWithProviders(
      <EmptyState
        title="No plans"
        description="Create a plan to preview cleanup."
        icon={<span data-testid="empty-icon" />}
        action={<button type="button">Create plan</button>}
        className="custom-empty"
      />,
    );

    expect(screen.getByText("No plans")).toBeInTheDocument();
    expect(screen.getByText("Create a plan to preview cleanup.")).toBeInTheDocument();
    expect(screen.getByTestId("empty-icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create plan" })).toBeInTheDocument();
  });

  it("renders with only a title", () => {
    renderWithProviders(<EmptyState title="Nothing here" />);

    expect(screen.getByText("Nothing here")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
