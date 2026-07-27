import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PreflightStepHeader } from "./PreflightStepHeader";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

describe("PreflightStepHeader", () => {
  it("presents indexed title, optional subtitle, and status label", () => {
    renderWithProviders(<PreflightStepHeader index={2} title="Runtime readiness" subtitle="Verify bundled services" status={{ state: "warning", label: "Needs attention" }} />);
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("Runtime readiness")).toBeInTheDocument();
    expect(screen.getByText("Verify bundled services")).toBeInTheDocument();
    expect(screen.getByText("Needs attention")).toBeInTheDocument();
  });

  it("does not render an absent subtitle", () => {
    renderWithProviders(<PreflightStepHeader index={1} title="Bundle" status={{ state: "pass", label: "Ready" }} />);
    expect(screen.queryByText("Verify bundled services")).not.toBeInTheDocument();
  });
});
