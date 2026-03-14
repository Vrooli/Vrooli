import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import TerminalContextMenu from "../components/TerminalContextMenu";

const defaultProps = () => ({
  position: { x: 200, y: 300 },
  hasSelection: false,
  onCopy: vi.fn(),
  onPaste: vi.fn(),
  onSelectAll: vi.fn(),
  onClear: vi.fn(),
  onUploadImage: vi.fn(),
  onClose: vi.fn(),
});

describe("TerminalContextMenu", () => {
  beforeEach(() => {
    // Default: clipboard.readText succeeds
    Object.assign(navigator, {
      clipboard: {
        readText: vi.fn().mockResolvedValue("pasted text"),
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders Paste, Select All, and Clear items", () => {
    render(<TerminalContextMenu {...defaultProps()} />);
    expect(screen.getByTestId("ctx-paste")).toBeInTheDocument();
    expect(screen.getByTestId("ctx-select-all")).toBeInTheDocument();
    expect(screen.getByTestId("ctx-clear")).toBeInTheDocument();
  });

  it("hides Copy when hasSelection is false", () => {
    render(<TerminalContextMenu {...defaultProps()} />);
    expect(screen.queryByTestId("ctx-copy")).not.toBeInTheDocument();
  });

  it("shows Copy when hasSelection is true", () => {
    render(<TerminalContextMenu {...defaultProps()} hasSelection={true} />);
    expect(screen.getByTestId("ctx-copy")).toBeInTheDocument();
  });

  it("calls onCopy when Copy is clicked", () => {
    const props = defaultProps();
    render(<TerminalContextMenu {...props} hasSelection={true} />);
    fireEvent.click(screen.getByTestId("ctx-copy"));
    expect(props.onCopy).toHaveBeenCalledOnce();
  });

  it("reads clipboard and calls onPaste on Paste click", async () => {
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    await act(async () => {
      fireEvent.click(screen.getByTestId("ctx-paste"));
    });
    expect(navigator.clipboard.readText).toHaveBeenCalled();
    expect(props.onPaste).toHaveBeenCalledWith("pasted text");
    expect(props.onClose).toHaveBeenCalled();
  });

  it("shows fallback text when clipboard read fails", async () => {
    vi.spyOn(navigator.clipboard, "readText").mockRejectedValue(
      new DOMException("denied"),
    );
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    await act(async () => {
      fireEvent.click(screen.getByTestId("ctx-paste"));
    });
    expect(props.onPaste).not.toHaveBeenCalled();
    expect(screen.getByTestId("ctx-paste").textContent).toBe(
      "Use Ctrl+V to paste",
    );
  });

  it("calls onSelectAll when Select All is clicked", () => {
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    fireEvent.click(screen.getByTestId("ctx-select-all"));
    expect(props.onSelectAll).toHaveBeenCalledOnce();
  });

  it("calls onClear when Clear Terminal is clicked", () => {
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    fireEvent.click(screen.getByTestId("ctx-clear"));
    expect(props.onClear).toHaveBeenCalledOnce();
  });

  it("dismisses on Escape key", () => {
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(props.onClose).toHaveBeenCalledOnce();
  });

  it("dismisses on backdrop click", () => {
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    fireEvent.click(screen.getByTestId("ctx-backdrop"));
    expect(props.onClose).toHaveBeenCalledOnce();
  });

  it("renders all four actions when hasSelection is true", () => {
    render(<TerminalContextMenu {...defaultProps()} hasSelection={true} />);
    expect(screen.getByTestId("ctx-copy")).toBeInTheDocument();
    expect(screen.getByTestId("ctx-paste")).toBeInTheDocument();
    expect(screen.getByTestId("ctx-select-all")).toBeInTheDocument();
    expect(screen.getByTestId("ctx-clear")).toBeInTheDocument();
  });

  it("does not call onPaste when clipboard returns empty string", async () => {
    vi.spyOn(navigator.clipboard, "readText").mockResolvedValue("");
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    await act(async () => {
      fireEvent.click(screen.getByTestId("ctx-paste"));
    });
    expect(props.onPaste).not.toHaveBeenCalled();
    expect(props.onClose).toHaveBeenCalled();
  });

  it("positions the menu at the given coordinates", () => {
    render(<TerminalContextMenu {...defaultProps()} />);
    const menu = screen.getByTestId("terminal-context-menu");
    // Before measurement, menu renders at position with opacity 0
    expect(menu.style.left).toBe("200px");
    expect(menu.style.top).toBe("300px");
  });

  it("renders Upload Image button when onUploadImage is provided", () => {
    render(<TerminalContextMenu {...defaultProps()} />);
    expect(screen.getByTestId("ctx-upload-image")).toBeInTheDocument();
  });

  it("calls onUploadImage when Upload Image is clicked", () => {
    const props = defaultProps();
    render(<TerminalContextMenu {...props} />);
    fireEvent.click(screen.getByTestId("ctx-upload-image"));
    expect(props.onUploadImage).toHaveBeenCalledOnce();
  });

  it("hides Upload Image when onUploadImage is undefined", () => {
    const props = defaultProps();
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { onUploadImage: _, ...propsWithout } = props;
    render(<TerminalContextMenu {...propsWithout} />);
    expect(screen.queryByTestId("ctx-upload-image")).not.toBeInTheDocument();
  });
});
