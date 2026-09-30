import { renderWithProviders as render } from "../../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor, waitForElementToBeRemoved } from "@testing-library/react";
import KeyComboPicker from "../KeyComboPicker";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";

// Reset store before each test
beforeEach(() => {
  useWorkspaceStore.setState({
    recentCombos: [],
  });
});

describe("KeyComboPicker", () => {
  const defaultProps = {
    onInput: vi.fn(() => ({ status: "sent" as const, offset: 1 })),
    onFocusTerminal: vi.fn(),
  };

  it("renders trigger button", () => {
    render(<KeyComboPicker {...defaultProps} />);
    expect(screen.getByTestId("combo-picker-trigger")).toBeInTheDocument();
  });

  it("trigger button has tabIndex={-1}", () => {
    render(<KeyComboPicker {...defaultProps} />);
    expect(screen.getByTestId("combo-picker-trigger")).toHaveAttribute("tabindex", "-1");
  });

  it("clicking trigger opens bottom sheet", () => {
    render(<KeyComboPicker {...defaultProps} />);
    fireEvent.click(screen.getByTestId("combo-picker-trigger"));
    expect(screen.getByTestId("combo-picker")).toBeInTheDocument();
    expect(screen.getByTestId("combo-picker.backdrop")).toBeInTheDocument();
  });

  it("pressing the backdrop closes bottom sheet", async () => {
    render(<KeyComboPicker {...defaultProps} />);
    fireEvent.click(screen.getByTestId("combo-picker-trigger"));
    expect(screen.getByTestId("combo-picker")).toBeInTheDocument();

    // The backdrop dismisses on press, not on click, and the sheet stays
    // mounted for the length of its exit transition before it is removed.
    fireEvent.pointerDown(screen.getByTestId("combo-picker.backdrop"));
    await waitForElementToBeRemoved(() => screen.queryByTestId("combo-picker"));
  });

  it("combo items are rendered with correct labels", () => {
    render(<KeyComboPicker {...defaultProps} />);
    fireEvent.click(screen.getByTestId("combo-picker-trigger"));

    // Check a few known combos
    expect(screen.getByTestId("combo-item-ctrl-c")).toBeInTheDocument();
    expect(screen.getByTestId("combo-item-ctrl-d")).toBeInTheDocument();
    expect(screen.getByTestId("combo-item-ctrl-c-x2")).toBeInTheDocument();
  });

  it("tapping a combo calls onInput and closes sheet", async () => {
    const onInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
    render(<KeyComboPicker onInput={onInput} onFocusTerminal={vi.fn()} />);
    fireEvent.click(screen.getByTestId("combo-picker-trigger"));
    fireEvent.click(screen.getByTestId("combo-item-ctrl-c"));

    // Sheet should close — after its exit transition has run.
    await waitForElementToBeRemoved(() => screen.queryByTestId("combo-picker"));

    // onInput should have been called with Ctrl+C data and the
    // toolbar-key source tag.
    await waitFor(() => {
      expect(onInput).toHaveBeenCalledWith("\x03", "named_key");
    });
  });

  it("search filters visible combos", () => {
    render(<KeyComboPicker {...defaultProps} />);
    fireEvent.click(screen.getByTestId("combo-picker-trigger"));

    const searchInput = screen.getByTestId("combo-picker-search");
    fireEvent.change(searchInput, { target: { value: "suspend" } });

    // Only Ctrl+Z (Suspend) should remain
    expect(screen.getByTestId("combo-item-ctrl-z")).toBeInTheDocument();
    expect(screen.queryByTestId("combo-item-ctrl-c")).not.toBeInTheDocument();
  });

  it("recent combos section appears when store has entries", () => {
    useWorkspaceStore.setState({ recentCombos: ["ctrl-c", "ctrl-d"] });
    render(<KeyComboPicker {...defaultProps} />);
    fireEvent.click(screen.getByTestId("combo-picker-trigger"));

    expect(screen.getByTestId("combo-recent-ctrl-c")).toBeInTheDocument();
    expect(screen.getByTestId("combo-recent-ctrl-d")).toBeInTheDocument();
  });

  it("calls onFocusTerminal after selecting a combo", async () => {
    const onFocusTerminal = vi.fn();
    render(<KeyComboPicker onInput={vi.fn(() => ({ status: "sent" as const, offset: 1 }))} onFocusTerminal={onFocusTerminal} />);
    fireEvent.click(screen.getByTestId("combo-picker-trigger"));
    fireEvent.click(screen.getByTestId("combo-item-ctrl-c"));

    expect(onFocusTerminal).toHaveBeenCalled();
  });
});
