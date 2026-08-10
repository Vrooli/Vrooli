import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { FrameworkTemplateSection } from "./FrameworkTemplateSection";

describe("FrameworkTemplateSection", () => {
  it("shows the supported Electron framework and selected template", () => {
    const onOpenFrameworkModal = vi.fn();
    const onOpenTemplateModal = vi.fn();

    renderWithProviders(
      <FrameworkTemplateSection
        framework="electron"
        selectedTemplate="basic"
        onOpenFrameworkModal={onOpenFrameworkModal}
        onOpenTemplateModal={onOpenTemplateModal}
      />,
    );

    expect(screen.getByText("Electron")).toBeInTheDocument();
    expect(screen.getByText("Basic")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Browse frameworks" }));
    fireEvent.click(screen.getByRole("button", { name: "Browse templates" }));
    expect(onOpenFrameworkModal).toHaveBeenCalledOnce();
    expect(onOpenTemplateModal).toHaveBeenCalledOnce();
  });

  it("falls back to readable labels for unknown framework and template ids", () => {
    renderWithProviders(
      <FrameworkTemplateSection
        framework="custom-framework"
        selectedTemplate="custom_template"
        onOpenFrameworkModal={vi.fn()}
        onOpenTemplateModal={vi.fn()}
      />,
    );

    expect(screen.getByText("custom-framework")).toBeInTheDocument();
    expect(screen.getByText("custom template")).toBeInTheDocument();
  });
});
