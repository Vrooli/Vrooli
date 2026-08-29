import { fireEvent, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../test-utils";
import type { SnippetDTO } from "../api/snippets";
import FullScreenComposer from "../components/FullScreenComposer";
import { useComposerDraft } from "../hooks/useComposerDraft";

const hook = vi.hoisted(() => ({ useSnippets: vi.fn() }));
vi.mock("../hooks/useSnippets", () => hook);

const rows: SnippetDTO[] = [
  { id: "plain", name: "Plain", body: "BODY", color: "", pinned: false, use_count: 0, last_used_at: "", sort_order: 0, created_at: "", updated_at: "" },
  { id: "vars", name: "Variables", body: "{{first}} and {{second}}", color: "", pinned: false, use_count: 0, last_used_at: "", sort_order: 0, created_at: "", updated_at: "" },
];

function Harness() {
  const draft = useComposerDraft("session-a");
  const [open, setOpen] = useState(true);
  return <FullScreenComposer open={open} onClose={() => setOpen(false)} draft={draft} onInput={() => ({ status: "sent", offset: 1 })} />;
}

describe("full-screen composer snippets", () => {
  const touch = vi.fn();
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    touch.mockResolvedValue(undefined);
    hook.useSnippets.mockReturnValue({ snippets: rows, status: "ready", touch, save: vi.fn() });
  });

  it("inserts a plain snippet at the recorded caret without replacing surrounding text", async () => {
    render(<Harness />);
    const input = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "hello world" } });
    input.setSelectionRange(6, 6);
    fireEvent.select(input);
    fireEvent.click(screen.getByTestId("composer-open-snippets"));
    fireEvent.click(screen.getByTestId("snippet-row-plain"));
    await waitFor(() => expect(input.value).toBe("hello BODY world"));
    expect(touch).toHaveBeenCalledWith("plain");
  });

  it("detours two unresolved variables to exactly two fields and one preview", () => {
    render(<Harness />);
    fireEvent.click(screen.getByTestId("composer-open-snippets"));
    fireEvent.click(screen.getByTestId("snippet-row-vars"));
    expect(screen.getByTestId("snippet-variable-input-first")).toBeTruthy();
    expect(screen.getByTestId("snippet-variable-input-second")).toBeTruthy();
    expect(screen.getAllByTestId("snippet-variable-preview")).toHaveLength(1);
    expect(touch).not.toHaveBeenCalled();
  });

  it("closes the nested snippet picker without closing the composer or inserting text", async () => {
    render(<Harness />);
    fireEvent.click(screen.getByTestId("composer-open-snippets"));
    fireEvent.click(screen.getByTestId("snippet-picker.close"));
    await waitFor(() => expect(screen.queryByTestId("snippet-picker")).toBeNull());
    expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
    expect(screen.getByTestId("composer-input")).toHaveValue("");
    expect(touch).not.toHaveBeenCalled();
  });
});
