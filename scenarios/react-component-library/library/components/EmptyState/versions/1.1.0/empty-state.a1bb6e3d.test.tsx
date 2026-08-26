import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { EmptyState } from "./EmptyState.tsx";

describe("EmptyState", () => {
  afterEach(cleanup);

  it("renders only the required title when optional content is absent", () => {
    renderWithProviders(<EmptyState title="No pending approvals" />);

    expect(screen.getByRole("heading", { name: "No pending approvals" })).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("renders description, icon, action, and caller styling together", () => {
    const { container } = renderWithProviders(
      <EmptyState
        title="No mandates"
        description="Issue a bounded mandate before an agent can spend."
        icon={<span aria-label="mandate icon">M</span>}
        action={<button type="button">Issue mandate</button>}
        className="audit-empty-state"
      />,
    );

    expect(screen.getByText("Issue a bounded mandate before an agent can spend.")).toBeInTheDocument();
    expect(screen.getByLabelText("mandate icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Issue mandate" })).toBeInTheDocument();
    expect(container.firstChild).toHaveClass("audit-empty-state");
  });
});
