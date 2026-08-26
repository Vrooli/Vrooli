import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EmptyState } from "./EmptyState.tsx";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

describe("EmptyState", () => {
  it("renders optional context and an operator action", () => {
    renderWithProviders(<EmptyState title="No drafts" description="Create one to continue" icon={<span>◎</span>} action={<button type="button">Create draft</button>} />);
    expect(screen.getByRole("heading", { name: "No drafts" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create draft" })).toBeInTheDocument();
  });
});
