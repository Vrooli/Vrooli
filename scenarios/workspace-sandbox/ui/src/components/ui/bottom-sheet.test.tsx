import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import userEvent from "@testing-library/user-event";
import { BottomSheet, BottomSheetAction } from "./bottom-sheet";

describe("BottomSheet", () => {
  it("renders children when isOpen is true", () => {
    renderWithProviders(
      <BottomSheet isOpen onClose={vi.fn()}>
        <div>Sheet content</div>
      </BottomSheet>,
    );
    expect(screen.getByText("Sheet content")).toBeInTheDocument();
  });

  it("does not render when isOpen is false", () => {
    renderWithProviders(
      <BottomSheet isOpen={false} onClose={vi.fn()}>
        <div>Sheet content</div>
      </BottomSheet>,
    );
    expect(screen.queryByText("Sheet content")).not.toBeInTheDocument();
  });

  it("renders title when provided", () => {
    renderWithProviders(
      <BottomSheet isOpen onClose={vi.fn()} title="Test Title">
        <div>content</div>
      </BottomSheet>,
    );
    expect(screen.getByText("Test Title")).toBeInTheDocument();
  });

  it("calls onClose when Escape key is pressed", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(
      <BottomSheet isOpen onClose={onClose}>
        <div>content</div>
      </BottomSheet>,
    );

    await user.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when backdrop is clicked", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(
      <BottomSheet isOpen onClose={onClose}>
        <div>content</div>
      </BottomSheet>,
    );

    await user.click(screen.getByTestId("bottom-sheet-backdrop"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("has data-testid on the sheet container", () => {
    renderWithProviders(
      <BottomSheet isOpen onClose={vi.fn()}>
        <div>content</div>
      </BottomSheet>,
    );
    expect(screen.getByTestId("bottom-sheet")).toBeInTheDocument();
  });
});

describe("BottomSheetAction", () => {
  it("renders label", () => {
    renderWithProviders(<BottomSheetAction label="Test Action" onClick={vi.fn()} />);
    expect(screen.getByText("Test Action")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    renderWithProviders(
      <BottomSheetAction label="Test" description="A description" onClick={vi.fn()} />,
    );
    expect(screen.getByText("A description")).toBeInTheDocument();
  });

  it("calls onClick when clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    renderWithProviders(<BottomSheetAction label="Click me" onClick={onClick} />);

    await user.click(screen.getByText("Click me"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("is disabled when disabled prop is true", () => {
    renderWithProviders(<BottomSheetAction label="Disabled" onClick={vi.fn()} disabled />);
    expect(screen.getByRole("button")).toBeDisabled();
  });
});
