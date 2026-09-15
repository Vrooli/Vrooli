import { useState } from "react";
import { fireEvent, render, screen, waitFor, act, cleanup } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PromptComposer, type PromptComposerProps } from "@vrooli/react-component-library/PromptComposer/1.0.2";
import { ComposerAttachmentTray } from "@vrooli/react-component-library/ComposerAttachmentTray/1.0.2";

afterEach(cleanup);
function Harness(props: Omit<PromptComposerProps, "value" | "onValueChange">) {
  const [value, setValue] = useState("Please review the rollout.");
  return <PromptComposer {...props} value={value} onValueChange={setValue} />;
}
describe("conversation send contract", () => {
  it("reorders attachments through keyboard-accessible actions", () => {
    const onReorder = vi.fn();
    render(<ComposerAttachmentTray items={[{ id: "first", name: "Brief.pdf" }, { id: "second", name: "Notes.md" }]} onReorder={onReorder} />);
    fireEvent.click(screen.getByRole("button", { name: "Move Notes.md earlier" }));
    expect(onReorder).toHaveBeenCalledWith(["second", "first"]);
    expect(screen.getByText("Notes.md moved to position 1 of 2.")).toHaveAttribute("role", "status");
  });
  it("retains the draft after failure and clears it only after an accepted retry", async () => {
    const onSend = vi.fn().mockResolvedValueOnce(false).mockResolvedValueOnce(true);
    render(<Harness onSend={onSend} />);
    fireEvent.click(screen.getByRole("button", { name: "Send message" }));
    await screen.findByText(/Couldn’t send/);
    expect(screen.getByRole("textbox")).toHaveValue("Please review the rollout.");
    fireEvent.click(screen.getByRole("button", { name: "Retry send" }));
    await waitFor(() => expect(screen.getByRole("textbox")).toHaveValue(""));
    expect(onSend).toHaveBeenCalledTimes(2);
  });
  it("does not erase newer typing when an earlier submission completes", async () => {
    let resolve!: (accepted: boolean) => void;
    const onSend = vi.fn(() => new Promise<boolean>(done => { resolve = done; }));
    render(<Harness onSend={onSend} />);
    const input = screen.getByRole("textbox");
    fireEvent.keyDown(input, { key: "Enter" });
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() => expect(onSend).toHaveBeenCalledTimes(1));
    fireEvent.change(input, { target: { value: "A follow-up thought." } });
    await act(async () => resolve(true));
    expect(input).toHaveValue("A follow-up thought.");
  });
  it("keeps IME confirmation and newline entry separate from submission", () => {
    const onSend = vi.fn().mockResolvedValue(true);
    render(<Harness onSend={onSend} />);
    const input = screen.getByRole("textbox");
    fireEvent.keyDown(input, { key: "Enter", shiftKey: true });
    fireEvent.keyDown(input, { key: "Enter", isComposing: true });
    fireEvent.keyDown(input, { key: "Enter", keyCode: 229 });
    expect(onSend).not.toHaveBeenCalled();
  });
  it("explains offline state and keeps the draft editable", () => {
    const onSend = vi.fn();
    render(<Harness onSend={onSend} offline />);
    expect(screen.getByRole("button", { name: "Send message" })).toBeDisabled();
    expect(screen.getByText(/You’re offline/)).toHaveAttribute("role", "status");
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "Still writing." } });
    expect(screen.getByRole("textbox")).toHaveValue("Still writing.");
    expect(onSend).not.toHaveBeenCalled();
  });
  it("keeps failed and healthy attachments until the caller resolves them", () => {
    render(<Harness onSend={vi.fn()} attachments={[{ id: "ok", name: "Brief.pdf", status: "success" }, { id: "failed", name: "Notes.md", status: "error" }]} onAttachmentRetry={vi.fn()} />);
    expect(screen.getByText("Brief.pdf")).toBeVisible();
    expect(screen.getByText("Notes.md")).toBeVisible();
    expect(screen.getByRole("button", { name: "Send message" })).toBeDisabled();
    expect(screen.getByText("Resolve unfinished attachments before sending.")).toBeVisible();
  });
});
