import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { Composer } from "./Composer";
import { formatBytes } from "./Transcript";

describe("Composer", () => {
  it("sends on Enter and clears on success", async () => {
    const onSend = vi.fn(() => true);
    renderWithProviders(<Composer onSend={onSend} />);
    const input = screen.getByTestId("conversations-composer-input");
    fireEvent.change(input, { target: { value: "hi" } });
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() => expect(onSend).toHaveBeenCalledWith("hi", undefined));
    await waitFor(() => expect(input).toHaveValue(""));
  });

  it("keeps the draft and shows an error when the send fails", async () => {
    const onSend = vi.fn(() => false);
    renderWithProviders(<Composer onSend={onSend} />);
    const input = screen.getByTestId("conversations-composer-input");
    fireEvent.change(input, { target: { value: "still here" } });
    fireEvent.click(screen.getByTestId("conversations-send"));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(input).toHaveValue("still here");
  });

  it("does not send on Shift+Enter and disables send with no content", () => {
    const onSend = vi.fn(() => true);
    renderWithProviders(<Composer onSend={onSend} />);
    const input = screen.getByTestId("conversations-composer-input");
    expect(screen.getByTestId("conversations-send")).toBeDisabled();
    fireEvent.change(input, { target: { value: "line" } });
    fireEvent.keyDown(input, { key: "Enter", shiftKey: true });
    expect(onSend).not.toHaveBeenCalled();
  });

  it("attaches a file, lets it be removed, and passes it on send", async () => {
    const onSend = vi.fn(() => true);
    renderWithProviders(<Composer onSend={onSend} />);
    const file = new File(["abc"], "notes.txt", { type: "text/plain" });
    const fileInput = document.querySelector<HTMLInputElement>('input[type="file"]');
    expect(fileInput).not.toBeNull();
    if (fileInput) fireEvent.change(fileInput, { target: { files: [file] } });
    expect(screen.getByTestId("conversations-composer")).toHaveTextContent("notes.txt");
    fireEvent.click(screen.getByTestId("conversations-send"));
    await waitFor(() => expect(onSend).toHaveBeenCalledWith("", file));
  });

  it("renders the disabled reason instead of a textbox", () => {
    renderWithProviders(<Composer onSend={() => true} disabledReason="elsewhere" />);
    expect(screen.getByTestId("conversations-composer")).toHaveTextContent("elsewhere");
    expect(screen.queryByTestId("conversations-composer-input")).not.toBeInTheDocument();
  });

  it("formats byte sizes", () => {
    expect(formatBytes(512)).toMatch(/512/);
    expect(formatBytes(2048)).toMatch(/KB/);
    expect(formatBytes(3 * 1024 * 1024)).toMatch(/MB/);
  });
});
