import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { AutonomyTab } from "./AutonomyTab";
import { DEFAULT_SETTINGS } from "../../services/settings-service";

const transition = {
  key: "proposal.apply",
  humanGates: [{ id: "suggested-to-backlog", decides: "Accept the suggestion", defaultMode: "manual", threshold: 0.9, minSample: 20 }],
} as never;

describe("AutonomyTab", () => {
  it("renders every declared gate and distinguishes insufficient evidence", () => {
    renderWithProviders(<AutonomyTab form={DEFAULT_SETTINGS} patch={vi.fn()} transitions={[transition]} />);
    expect(screen.getByText("suggested-to-backlog")).toBeInTheDocument();
    expect(screen.getByText("proposal.apply")).toBeInTheDocument();
    expect(screen.getByText(/Attributed sample: 0/)).toBeInTheDocument();
    expect(screen.getByText(/Readiness: insufficient sample/)).toBeInTheDocument();
  });

  it("persists a three-way mode choice through the page patch callback", async () => {
    const user = userEvent.setup();
    const patch = vi.fn();
    renderWithProviders(<AutonomyTab form={DEFAULT_SETTINGS} patch={patch} transitions={[transition]} />);
    await user.click(screen.getByRole("button", { name: "Automatic" }));
    expect(patch).toHaveBeenCalledWith({ autonomyGateModes: { "suggested-to-backlog": "auto" } });
  });
});
