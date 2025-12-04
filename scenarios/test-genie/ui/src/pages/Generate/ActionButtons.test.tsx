import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ActionButtons } from "./ActionButtons";

// Mock clipboard API
const mockClipboard = {
  writeText: vi.fn()
};

Object.assign(navigator, {
  clipboard: mockClipboard
});

describe("ActionButtons", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClipboard.writeText.mockResolvedValue(undefined);
  });

  it("renders copy prompt button", () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={false} />);
    expect(screen.getByText("Copy prompt")).toBeInTheDocument();
  });

  it("renders spawn agent button", () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={false} />);
    expect(screen.getByText("Spawn agents")).toBeInTheDocument();
  });

  it("spawn agent button is always disabled", () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={false} />);
    const spawnButton = screen.getByText("Spawn agents").closest("button");
    expect(spawnButton).toBeDisabled();
  });

  it("disables copy button when disabled prop is true", () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={true} />);
    const copyButton = screen.getByText("Copy prompt").closest("button");
    expect(copyButton).toBeDisabled();
  });

  it("disables copy button when prompt is empty", () => {
    render(<ActionButtons prompt="" allPrompts={[]} disabled={false} />);
    const copyButton = screen.getByText("Copy prompt").closest("button");
    expect(copyButton).toBeDisabled();
  });

  it("enables copy button when prompt exists and not disabled", () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={false} />);
    const copyButton = screen.getByText("Copy prompt").closest("button");
    expect(copyButton).not.toBeDisabled();
  });

  it("copies prompt to clipboard when clicked", async () => {
    render(<ActionButtons prompt="test prompt content" allPrompts={["test prompt content"]} disabled={false} />);

    const copyButton = screen.getByText("Copy prompt").closest("button");
    fireEvent.click(copyButton!);

    expect(mockClipboard.writeText).toHaveBeenCalledWith("test prompt content");
  });

  it("shows copied feedback after clicking", async () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={false} />);

    const copyButton = screen.getByText("Copy prompt").closest("button");
    fireEvent.click(copyButton!);

    await waitFor(() => {
      expect(screen.getByText("Copied!")).toBeInTheDocument();
    });
  });

  it("has correct data-testid attributes", () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={false} />);

    expect(screen.getByTestId("test-genie-copy-prompt")).toBeInTheDocument();
    expect(screen.getByTestId("test-genie-spawn-agent")).toBeInTheDocument();
  });

  it("does not copy when disabled", async () => {
    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={true} />);

    const copyButton = screen.getByText("Copy prompt").closest("button");
    fireEvent.click(copyButton!);

    expect(mockClipboard.writeText).not.toHaveBeenCalled();
  });

  it("handles clipboard error gracefully", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockClipboard.writeText.mockRejectedValue(new Error("Clipboard failed"));

    render(<ActionButtons prompt="test prompt" allPrompts={["test prompt"]} disabled={false} />);

    const copyButton = screen.getByText("Copy prompt").closest("button");
    fireEvent.click(copyButton!);

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalled();
    });

    consoleSpy.mockRestore();
  });
});
