import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import WorkspaceSection from "../components/settings/WorkspaceSection";

const mockStoreState: Record<string, unknown> = {
  isMinimapVisible: true,
  setMinimapVisible: vi.fn(),
  displayMode: "grid",
  setDisplayMode: vi.fn(),
  toolbarLayout: "expanded",
  setToolbarLayout: vi.fn(),
  keepScreenAwake: true,
  setKeepScreenAwake: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) => selector(mockStoreState),
}));

describe("WorkspaceSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStoreState.keepScreenAwake = true;
  });

  it("renders keep-screen-awake toggle checked when enabled", () => {
    render(<WorkspaceSection />);
    const toggle = screen.getByTestId("keep-screen-awake-toggle");
    expect(toggle).toHaveAttribute("aria-checked", "true");
  });

  it("renders keep-screen-awake toggle unchecked when disabled", () => {
    mockStoreState.keepScreenAwake = false;
    render(<WorkspaceSection />);
    const toggle = screen.getByTestId("keep-screen-awake-toggle");
    expect(toggle).toHaveAttribute("aria-checked", "false");
  });

  it("calls setKeepScreenAwake when toggle is clicked", () => {
    render(<WorkspaceSection />);
    const toggle = screen.getByTestId("keep-screen-awake-toggle");
    fireEvent.click(toggle);
    expect(mockStoreState.setKeepScreenAwake).toHaveBeenCalledWith(false);
  });
});
