import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FilePath } from "../../../library/components/FilePath/versions/1.0.6/FilePath";
import { renderWithProviders } from "../test-utils/renderWithProviders";

const originalClipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
const originalExecCommand = Object.getOwnPropertyDescriptor(document, "execCommand");

afterEach(() => {
  vi.restoreAllMocks();
  if (originalClipboard) Object.defineProperty(navigator, "clipboard", originalClipboard);
  else Reflect.deleteProperty(navigator, "clipboard");
  if (originalExecCommand) Object.defineProperty(document, "execCommand", originalExecCommand);
  else Reflect.deleteProperty(document, "execCommand");
});

describe("FilePath draft", () => {
  it("renders a wrapping inline link while copying the canonical path", async () => {
    const canonicalPath = "/workspace/research/topics/example/MISSION.md";
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });

    renderWithProviders(
      <FilePath
        path={canonicalPath}
        presentation="inline"
        displayText="~research/topics/example/MISSION.md"
        href="/files/topics/example/MISSION.md"
        showCopyButton
      />,
    );

    const link = screen.getByRole("link", { name: "~research/topics/example/MISSION.md" });
    expect(link.closest("[data-rcl-file-path]")?.tagName).toBe("SPAN");
    expect(link).toHaveAttribute("href", "/files/topics/example/MISSION.md");
    expect(link).toHaveAttribute("data-rcl-file-path-inline-label");
    expect(screen.getByTestId("file-path.copy")).toHaveAttribute("aria-label", "Copy path");
    expect(screen.queryByRole("button", { name: canonicalPath })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Copy path" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(canonicalPath));
  });

  it("keeps its gesture-bound fallback when the asynchronous clipboard is unavailable", async () => {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
    const execCommand = vi.fn(() => {
      expect(document.querySelector<HTMLTextAreaElement>("textarea[readonly]")?.value).toBe(
        "/workspace/reports/today.csv",
      );
      return true;
    });
    Object.defineProperty(document, "execCommand", { configurable: true, value: execCommand });

    renderWithProviders(<FilePath path="/workspace/reports/today.csv" />);
    fireEvent.click(screen.getByRole("button", { name: "/workspace/reports/today.csv" }));
    fireEvent.click(screen.getByRole("button", { name: "Copy path" }));

    await waitFor(() => expect(execCommand).toHaveBeenCalledWith("copy"));
    expect(document.querySelector("textarea[readonly]")).toBeNull();
  });
});
