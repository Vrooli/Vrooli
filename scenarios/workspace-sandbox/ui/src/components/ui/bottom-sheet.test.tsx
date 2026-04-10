import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BottomSheet, BottomSheetAction } from "./bottom-sheet";

describe("BottomSheet", () => {
  it("renders children when isOpen is true", () => {
    render(
      <BottomSheet isOpen onClose={vi.fn()}>
        <div>Sheet content</div>
      </BottomSheet>,
    );
    expect(screen.getByText("Sheet content")).toBeInTheDocument();
  });

  it("does not render when isOpen is false", () => {
    render(
      <BottomSheet isOpen={false} onClose={vi.fn()}>
        <div>Sheet content</div>
      </BottomSheet>,
    );
    expect(screen.queryByText("Sheet content")).not.toBeInTheDocument();
  });

  it("renders title when provided", () => {
    render(
      <BottomSheet isOpen onClose={vi.fn()} title="Test Title">
        <div>content</div>
      </BottomSheet>,
    );
    expect(screen.getByText("Test Title")).toBeInTheDocument();
  });

  it("calls onClose when Escape key is pressed", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
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
    render(
      <BottomSheet isOpen onClose={onClose}>
        <div>content</div>
      </BottomSheet>,
    );

    await user.click(screen.getByTestId("bottom-sheet-backdrop"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("has data-testid on the sheet container", () => {
    render(
      <BottomSheet isOpen onClose={vi.fn()}>
        <div>content</div>
      </BottomSheet>,
    );
    expect(screen.getByTestId("bottom-sheet")).toBeInTheDocument();
  });
});

describe("BottomSheetAction", () => {
  it("renders label", () => {
    render(<BottomSheetAction label="Test Action" onClick={vi.fn()} />);
    expect(screen.getByText("Test Action")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(
      <BottomSheetAction label="Test" description="A description" onClick={vi.fn()} />,
    );
    expect(screen.getByText("A description")).toBeInTheDocument();
  });

  it("calls onClick when clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<BottomSheetAction label="Click me" onClick={onClick} />);

    await user.click(screen.getByText("Click me"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("is disabled when disabled prop is true", () => {
    render(<BottomSheetAction label="Disabled" onClick={vi.fn()} disabled />);
    expect(screen.getByRole("button")).toBeDisabled();
  });
});
