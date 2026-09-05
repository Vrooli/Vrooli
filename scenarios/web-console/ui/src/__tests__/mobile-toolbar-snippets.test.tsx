import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../test-utils";
import type { SnippetDTO } from "../api/snippets";

const hook = vi.hoisted(() => ({ useSnippets: vi.fn() }));
vi.mock("../hooks/useSnippets", () => hook);

import MobileToolbar from "../components/MobileToolbar";

const rows: SnippetDTO[] = Array.from({ length: 10 }, (_, index) => ({
  id: `s${index}`,
  name: `Snippet ${index}`,
  body: index === 1 ? "Check {{owner}}" : `body-${index}`,
  color: "#38d9c0",
  pinned: false,
  use_count: 10 - index,
  last_used_at: "",
  sort_order: 0,
  created_at: "",
  updated_at: "",
}));

function mount() {
  return render(<MobileToolbar onInput={() => ({ status: "sent", offset: 1 })} activeSessionId="session-a" onFocusTerminal={vi.fn()} />);
}

describe("mobile toolbar snippets", () => {
  const touch = vi.fn();
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    touch.mockResolvedValue(undefined);
    hook.useSnippets.mockReturnValue({ snippets: rows, status: "ready", touch });
  });

  it("opens the full picker directly through the governed toolbar control", () => {
    mount();
    fireEvent.click(screen.getByTestId("toolbar-snippets"));
    expect(screen.getByTestId("snippet-picker")).toBeTruthy();
    expect(screen.queryByTestId("snippet-rail")).toBeNull();
  });

  it("inserts a plain picker selection through the shared draft", async () => {
    mount();
    const input = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "before" } });
    fireEvent.click(screen.getByTestId("toolbar-snippets"));
    fireEvent.click(screen.getByTestId("snippet-row-s0"));
    await waitFor(() => expect(input.value).toBe("before body-0"));
    await waitFor(() => expect(touch).toHaveBeenCalledWith("s0"));
  });

  it("detours an unresolved variable without changing the draft", () => {
    mount();
    const input = screen.getByTestId("mobile-command-input") as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: "before" } });
    fireEvent.click(screen.getByTestId("toolbar-snippets"));
    fireEvent.click(screen.getByTestId("snippet-row-s1"));
    expect(screen.getByTestId("snippet-variable-input-owner")).toBeTruthy();
    expect(input.value).toBe("before");
    expect(touch).not.toHaveBeenCalled();
  });

  it("does not persist the picker across mounts", () => {
    const first = mount();
    fireEvent.click(screen.getByTestId("toolbar-snippets"));
    expect(screen.getByTestId("snippet-picker")).toBeTruthy();
    first.unmount();
    mount();
    expect(screen.queryByTestId("snippet-picker")).toBeNull();
  });
});
