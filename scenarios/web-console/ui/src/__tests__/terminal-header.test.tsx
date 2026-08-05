import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import TerminalHeader from "../components/TerminalHeader";

// Mock the workspace store
const mockRenamePaneById = vi.fn();
const mockSetAppearanceModalPane = vi.fn();

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) => {
    const state = {
      // The header reads panes/groups to resolve its accent (own color first,
      // then the group's), so both must exist for the selector to run.
      panes: [],
      groups: [],
      renamePaneById: mockRenamePaneById,
      setAppearanceModalPane: mockSetAppearanceModalPane,
    };
    return selector(state);
  },
}));

describe("TerminalHeader", () => {
  const defaultProps = {
    sessionId: "sess-1",
    name: "bash",
    headerColor: "transparent",
    isActive: false,
    onClose: vi.fn(),
    onFocus: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the pane name", () => {
    render(<TerminalHeader {...defaultProps} />);
    expect(screen.getByTestId("terminal-header-name-sess-1")).toBeTruthy();
    expect(screen.getByText("bash")).toBeTruthy();
  });

  it("enters edit mode on name click", () => {
    render(<TerminalHeader {...defaultProps} />);
    fireEvent.click(screen.getByTestId("terminal-header-name-sess-1"));
    expect(screen.getByTestId("terminal-header-name-input-sess-1")).toBeTruthy();
  });

  it("commits rename on blur", () => {
    render(<TerminalHeader {...defaultProps} />);
    fireEvent.click(screen.getByTestId("terminal-header-name-sess-1"));

    const input = screen.getByTestId("terminal-header-name-input-sess-1") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "my-server" } });
    fireEvent.blur(input);

    expect(mockRenamePaneById).toHaveBeenCalledWith("sess-1", "my-server");
  });

  it("commits rename on Enter key", () => {
    render(<TerminalHeader {...defaultProps} />);
    fireEvent.click(screen.getByTestId("terminal-header-name-sess-1"));

    const input = screen.getByTestId("terminal-header-name-input-sess-1") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "new-name" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(mockRenamePaneById).toHaveBeenCalledWith("sess-1", "new-name");
  });

  it("cancels rename on Escape key", () => {
    render(<TerminalHeader {...defaultProps} />);
    fireEvent.click(screen.getByTestId("terminal-header-name-sess-1"));

    const input = screen.getByTestId("terminal-header-name-input-sess-1") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "new-name" } });
    fireEvent.keyDown(input, { key: "Escape" });

    // Should exit edit mode without calling rename
    expect(mockRenamePaneById).not.toHaveBeenCalled();
    expect(screen.getByTestId("terminal-header-name-sess-1")).toBeTruthy();
  });

  it("clicking appearance button calls setAppearanceModalPane", () => {
    render(<TerminalHeader {...defaultProps} />);
    fireEvent.click(screen.getByTestId("terminal-header-appearance-sess-1"));
    expect(mockSetAppearanceModalPane).toHaveBeenCalledWith("sess-1");
  });

  it("calls onClose when close button is clicked", () => {
    render(<TerminalHeader {...defaultProps} />);
    fireEvent.click(screen.getByTestId("terminal-close-sess-1"));
    expect(defaultProps.onClose).toHaveBeenCalledOnce();
  });

  it("applies active border when isActive", () => {
    const { container } = render(
      <TerminalHeader {...defaultProps} isActive={true} />,
    );
    const header = container.firstChild as HTMLElement;
    expect(header.className).toContain("border-b-2");
  });

  it("applies header color as background", () => {
    const { container } = render(
      <TerminalHeader {...defaultProps} headerColor="#ff7a7a" />,
    );
    const header = container.firstChild as HTMLElement;
    expect(header.style.backgroundColor).toBe("rgb(255, 122, 122)");
  });
});
