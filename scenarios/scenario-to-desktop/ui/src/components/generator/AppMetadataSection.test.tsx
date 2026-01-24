/**
 * Tests for AppMetadataSection component.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AppMetadataSection } from "./AppMetadataSection";

describe("AppMetadataSection", () => {
  const defaultProps = {
    scenarioName: "test-scenario",
    appDisplayName: "",
    onAppDisplayNameChange: vi.fn(),
    iconPath: "",
    onIconPathChange: vi.fn(),
    iconPreviewUrl: null,
    iconPreviewError: false,
    onIconPreviewError: vi.fn(),
    appDescription: "",
    onAppDescriptionChange: vi.fn(),
  };

  it("renders all form fields", () => {
    render(<AppMetadataSection {...defaultProps} />);

    expect(screen.getByLabelText("App display name")).toBeInTheDocument();
    expect(screen.getByLabelText("Icon path (PNG)")).toBeInTheDocument();
    expect(screen.getByLabelText("App description")).toBeInTheDocument();
  });

  it("displays scenario name in placeholders", () => {
    render(<AppMetadataSection {...defaultProps} scenarioName="my-app" />);

    const displayNameInput = screen.getByLabelText("App display name");
    expect(displayNameInput).toHaveAttribute("placeholder", "my-app Desktop");
  });

  it("calls onAppDisplayNameChange when display name is typed", () => {
    const onAppDisplayNameChange = vi.fn();
    render(
      <AppMetadataSection
        {...defaultProps}
        onAppDisplayNameChange={onAppDisplayNameChange}
      />
    );

    const input = screen.getByLabelText("App display name");
    fireEvent.change(input, { target: { value: "My App" } });

    expect(onAppDisplayNameChange).toHaveBeenCalledWith("My App");
  });

  it("calls onIconPathChange when icon path is typed", () => {
    const onIconPathChange = vi.fn();
    render(
      <AppMetadataSection {...defaultProps} onIconPathChange={onIconPathChange} />
    );

    const input = screen.getByLabelText("Icon path (PNG)");
    fireEvent.change(input, { target: { value: "/path/to/icon.png" } });

    expect(onIconPathChange).toHaveBeenCalledWith("/path/to/icon.png");
  });

  it("calls onAppDescriptionChange when description is typed", () => {
    const onAppDescriptionChange = vi.fn();
    render(
      <AppMetadataSection
        {...defaultProps}
        onAppDescriptionChange={onAppDescriptionChange}
      />
    );

    const textarea = screen.getByLabelText("App description");
    fireEvent.change(textarea, { target: { value: "My app description" } });

    expect(onAppDescriptionChange).toHaveBeenCalledWith("My app description");
  });

  it("shows 'No icon' when iconPreviewUrl is null", () => {
    render(<AppMetadataSection {...defaultProps} iconPreviewUrl={null} />);

    expect(screen.getByText("No icon")).toBeInTheDocument();
  });

  it("shows icon preview when iconPreviewUrl is provided and no error", () => {
    render(
      <AppMetadataSection
        {...defaultProps}
        iconPreviewUrl="/path/to/icon.png"
        iconPreviewError={false}
      />
    );

    const img = screen.getByAltText("Icon preview");
    expect(img).toBeInTheDocument();
    expect(img).toHaveAttribute("src", "/path/to/icon.png");
  });

  it("shows 'No icon' when iconPreviewError is true", () => {
    render(
      <AppMetadataSection
        {...defaultProps}
        iconPreviewUrl="/path/to/icon.png"
        iconPreviewError={true}
      />
    );

    expect(screen.getByText("No icon")).toBeInTheDocument();
    expect(screen.queryByAltText("Icon preview")).not.toBeInTheDocument();
  });

  it("calls onIconPreviewError when image fails to load", () => {
    const onIconPreviewError = vi.fn();
    render(
      <AppMetadataSection
        {...defaultProps}
        iconPreviewUrl="/path/to/invalid.png"
        iconPreviewError={false}
        onIconPreviewError={onIconPreviewError}
      />
    );

    const img = screen.getByAltText("Icon preview");
    fireEvent.error(img);

    expect(onIconPreviewError).toHaveBeenCalledWith(true);
  });

  it("displays current values in inputs", () => {
    render(
      <AppMetadataSection
        {...defaultProps}
        appDisplayName="My Desktop App"
        iconPath="/icons/app.png"
        appDescription="A great desktop app"
      />
    );

    expect(screen.getByLabelText("App display name")).toHaveValue("My Desktop App");
    expect(screen.getByLabelText("Icon path (PNG)")).toHaveValue("/icons/app.png");
    expect(screen.getByLabelText("App description")).toHaveValue("A great desktop app");
  });

  it("shows preview success message when icon loads", () => {
    render(
      <AppMetadataSection
        {...defaultProps}
        iconPreviewUrl="/path/to/icon.png"
        iconPreviewError={false}
      />
    );

    expect(screen.getByText("Previewing selected icon.")).toBeInTheDocument();
  });

  it("shows preview hint message when no icon", () => {
    render(<AppMetadataSection {...defaultProps} iconPreviewUrl={null} />);

    expect(
      screen.getByText("Preview will appear once a valid PNG path is set.")
    ).toBeInTheDocument();
  });
});
