import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FrameworkModal } from "./FrameworkModal";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

describe("FrameworkModal", () => {
  it("does not mount while closed", () => {
    renderWithProviders(<FrameworkModal open={false} selectedFramework="electron" onSelect={vi.fn()} onClose={vi.fn()} />);
    expect(screen.queryByText("Choose a framework")).not.toBeInTheDocument();
  });

  it("explains and selects the supported Electron framework", () => {
    const onSelect = vi.fn();
    renderWithProviders(<FrameworkModal open selectedFramework="electron" onSelect={onSelect} onClose={vi.fn()} />);
    expect(screen.getByText("Most compatible today")).toBeInTheDocument();
    expect(screen.getByText("Mature tooling · Full Node access · Largest community")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Select" }));
    expect(onSelect).toHaveBeenCalledWith("electron");
  });

  it("closes from its accessible control and backdrop", () => {
    const onClose = vi.fn();
    renderWithProviders(<FrameworkModal open selectedFramework="electron" onSelect={vi.fn()} onClose={onClose} />);
    fireEvent.click(screen.getByRole("button", { name: "Close framework chooser" }));
    const backdrop = screen.getByText("Choose a framework").closest("div.fixed");
    if (!backdrop) throw new Error("framework backdrop is not mounted");
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
