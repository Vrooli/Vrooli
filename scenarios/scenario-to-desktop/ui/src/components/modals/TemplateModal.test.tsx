import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TemplateModal } from "./TemplateModal";
import { renderWithProviders } from "@vrooli/api-base/testing";

vi.mock("../generator/TemplateGrid", () => ({
  TemplateGrid: ({
    selectedTemplate,
    onSelect,
  }: {
    selectedTemplate: string;
    onSelect: (template: string) => void;
  }) => (
    <div>
      Template grid: {selectedTemplate}
      <button
        type="button"
        onClick={() => {
          onSelect("advanced");
        }}
      >
        Select advanced
      </button>
    </div>
  ),
}));

describe("TemplateModal", () => {
  it("does not mount while closed", () => {
    renderWithProviders(
      <TemplateModal
        open={false}
        selectedTemplate="basic"
        onSelect={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.queryByText("Choose a template")).not.toBeInTheDocument();
  });

  it("selects a template and closes the chooser", () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    renderWithProviders(
      <TemplateModal
        open
        selectedTemplate="basic"
        onSelect={onSelect}
        onClose={onClose}
      />,
    );
    expect(
      screen.getByText(/All templates share the same Electron base/),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Select advanced" }));
    expect(onSelect).toHaveBeenCalledWith("advanced");
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("closes from the named control and backdrop", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <TemplateModal
        open
        selectedTemplate="basic"
        onSelect={vi.fn()}
        onClose={onClose}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Close template chooser" }),
    );
    const backdrop = screen.getByText("Choose a template").closest("div.fixed");
    if (!backdrop) throw new Error("template backdrop is not mounted");
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
