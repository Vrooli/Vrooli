import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { EmptyState } from "./empty-state";

describe("EmptyState", () => {
  it("renders optional icon, description, action, and class name", () => {
    renderWithProviders(
      <EmptyState title="No memories" description="Write one to begin" icon={<span>Icon</span>} action={<button>Write memory</button>} className="custom-empty" />,
    );

    expect(screen.getByText("Icon")).toBeInTheDocument();
    expect(screen.getByText("Write one to begin")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Write memory" })).toBeInTheDocument();
    expect(screen.getByText("No memories").closest("div")?.parentElement).toHaveClass("custom-empty");
  });
});
