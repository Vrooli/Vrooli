import { describe, it, expect, vi, beforeAll } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../test-utils/renderWithProviders";
import { Drawer } from "./drawer";

// jsdom doesn't provide matchMedia (needed by useIsMobile).
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

describe("Drawer", () => {
  it("does not render when isOpen is false", () => {
    renderWithCanonicalProviders(
      <Drawer isOpen={false} onClose={vi.fn()} title="Test">
        <p>content</p>
      </Drawer>,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders when isOpen is true", () => {
    renderWithCanonicalProviders(
      <Drawer isOpen={true} onClose={vi.fn()} title="Test Title">
        <p>drawer content</p>
      </Drawer>,
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Test Title")).toBeInTheDocument();
    expect(screen.getByText("drawer content")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    renderWithCanonicalProviders(
      <Drawer isOpen={true} onClose={vi.fn()} title="T" description="A description">
        <p>body</p>
      </Drawer>,
    );
    expect(screen.getByText("A description")).toBeInTheDocument();
  });

  it("renders footer when provided", () => {
    renderWithCanonicalProviders(
      <Drawer isOpen={true} onClose={vi.fn()} title="T" footer={<button>Save</button>}>
        <p>body</p>
      </Drawer>,
    );
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("calls onClose when the shared mobile dismiss affordance is clicked", () => {
    const onClose = vi.fn();
    renderWithCanonicalProviders(
      <Drawer isOpen={true} onClose={onClose} title="T">
        <p>body</p>
      </Drawer>,
    );
    fireEvent.click(screen.getByTestId("overlays.responsive-dialog.grabber"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose on Escape key press", () => {
    const onClose = vi.fn();
    renderWithCanonicalProviders(
      <Drawer isOpen={true} onClose={onClose} title="T">
        <p>body</p>
      </Drawer>,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("has correct ARIA attributes", () => {
    renderWithCanonicalProviders(
      <Drawer isOpen={true} onClose={vi.fn()} title="Accessible Title" testId="my-drawer">
        <p>body</p>
      </Drawer>,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAttribute("data-testid", "my-drawer");
    expect(dialog).toHaveAttribute("aria-label", "Accessible Title");
  });
});
