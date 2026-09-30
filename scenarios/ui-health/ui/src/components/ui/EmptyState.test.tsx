import { render, screen } from "@testing-library/react";
import { Inbox } from "lucide-react";
import { describe, expect, it } from "vitest";

import { EmptyState } from "./EmptyState";

describe("EmptyState", () => {
  it("renders title, description, and action", () => {
    render(
      <EmptyState
        data-testid="empty"
        icon={Inbox}
        title="t-x"
        description={<span data-testid="desc">d-x</span>}
        action={<button data-testid="cta" type="button">cta-x</button>}
      />,
    );
    const root = screen.getByTestId("empty");
    expect(root).toHaveAttribute("role", "status");
    expect(root).toHaveTextContent("t-x");
    expect(screen.getByTestId("desc")).toHaveTextContent("d-x");
    expect(screen.getByTestId("cta")).toBeInTheDocument();
  });
});
