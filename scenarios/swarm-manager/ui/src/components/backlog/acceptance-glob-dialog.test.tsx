import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AcceptanceGlobDialog } from "./acceptance-glob-dialog";

// Mock the API client to prevent real network calls during tests
vi.mock("../../lib/api-client", () => ({
  defaultApiClient: {
    post: vi.fn().mockResolvedValue({ results: [] }),
  },
}));

const defaultProps = {
  isOpen: true,
  onClose: vi.fn(),
  initialAllow: [] as string[],
  initialDeny: [] as string[],
  onSave: vi.fn(),
  isSubmitting: false,
};

describe("AcceptanceGlobDialog", () => {
  it("renders two textareas with correct labels", () => {
    render(<AcceptanceGlobDialog {...defaultProps} />);
    expect(screen.getByText("Allowed Paths")).toBeInTheDocument();
    expect(screen.getByText("Denied Paths")).toBeInTheDocument();
    expect(screen.getByTestId("allow-textarea")).toBeInTheDocument();
    expect(screen.getByTestId("deny-textarea")).toBeInTheDocument();
  });

  it("pre-populates from initialAllow and initialDeny", () => {
    render(
      <AcceptanceGlobDialog
        {...defaultProps}
        initialAllow={["src/**", "docs/**"]}
        initialDeny={["*.lock"]}
      />,
    );
    expect(screen.getByTestId("allow-textarea")).toHaveValue("src/**\ndocs/**");
    expect(screen.getByTestId("deny-textarea")).toHaveValue("*.lock");
  });

  it("shows placeholder text when textarea is empty", () => {
    render(<AcceptanceGlobDialog {...defaultProps} />);
    const allowTextarea = screen.getByTestId("allow-textarea");
    expect(allowTextarea).toHaveAttribute("placeholder");
  });

  it("shows helper text below each label", () => {
    render(<AcceptanceGlobDialog {...defaultProps} />);
    const helpers = screen.getAllByText("One glob pattern per line. Relative to project root.");
    expect(helpers).toHaveLength(2);
  });

  it("shows client-side validation errors after blur", async () => {
    render(<AcceptanceGlobDialog {...defaultProps} />);
    const allowTextarea = screen.getByTestId("allow-textarea");

    fireEvent.change(allowTextarea, { target: { value: "/absolute/path" } });
    fireEvent.blur(allowTextarea);

    await waitFor(() => {
      const errContainer = screen.getByTestId("allow-errors");
      expect(errContainer).toBeInTheDocument();
      expect(errContainer.textContent).toMatch(/absolute/i);
    });
  });

  it("disables save button when validation errors exist", async () => {
    render(<AcceptanceGlobDialog {...defaultProps} />);
    const allowTextarea = screen.getByTestId("allow-textarea");

    fireEvent.change(allowTextarea, { target: { value: "/bad" } });
    fireEvent.blur(allowTextarea);

    await waitFor(() => {
      expect(screen.getByTestId("glob-dialog-save")).toBeDisabled();
    });
  });

  it("enables save button when input is valid", () => {
    render(
      <AcceptanceGlobDialog
        {...defaultProps}
        initialAllow={["src/**"]}
      />,
    );
    expect(screen.getByTestId("glob-dialog-save")).not.toBeDisabled();
  });

  it("calls onSave with correctly parsed arrays on save click", async () => {
    const onSave = vi.fn();
    render(
      <AcceptanceGlobDialog
        {...defaultProps}
        onSave={onSave}
        initialAllow={["src/**"]}
        initialDeny={["*.lock", "node_modules/**"]}
      />,
    );

    await userEvent.click(screen.getByTestId("glob-dialog-save"));

    expect(onSave).toHaveBeenCalledWith(
      ["src/**"],
      ["*.lock", "node_modules/**"],
    );
  });

  it("calls onClose on cancel click", async () => {
    const onClose = vi.fn();
    render(<AcceptanceGlobDialog {...defaultProps} onClose={onClose} />);

    await userEvent.click(screen.getByTestId("glob-dialog-cancel"));

    expect(onClose).toHaveBeenCalled();
  });

  it("shows loading state when isSubmitting is true", () => {
    render(<AcceptanceGlobDialog {...defaultProps} isSubmitting />);
    expect(screen.getByText("Saving…")).toBeInTheDocument();
    expect(screen.getByTestId("glob-dialog-save")).toBeDisabled();
  });

  it("does not render when isOpen is false", () => {
    render(<AcceptanceGlobDialog {...defaultProps} isOpen={false} />);
    expect(screen.queryByText("Edit Acceptance Globs")).not.toBeInTheDocument();
  });
});
