import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { WorkshopTab } from "./WorkshopTab";

describe("WorkshopTab", () => {
  it("explains the explicit Plan Workshop operator flow without retired controls", () => {
    renderWithProviders(<WorkshopTab />);

    expect(screen.getByRole("heading", { name: "Plan Workshop" })).toBeInTheDocument();
    expect(screen.getByText(/no automatic rounds or readiness controls/i)).toBeInTheDocument();
    expect(screen.queryByText(/Auto-Advance Workshop/i)).not.toBeInTheDocument();
  });
});
