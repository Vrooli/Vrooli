import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";

describe("EmptyState", () => {
  it("renders optional description, icon, action, and class name", () => {
    renderWithProviders(<EmptyState title="Nothing here" description="Capture a signal first." icon={<span>◎</span>} action={<button type="button">Capture</button>} className="custom-empty" />);
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
    expect(screen.getByText("Capture a signal first.")).toBeInTheDocument();
    expect(screen.getByText("◎")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Capture" })).toBeInTheDocument();
  });
});
