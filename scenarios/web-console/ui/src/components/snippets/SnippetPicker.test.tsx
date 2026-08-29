import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils";
import type { SnippetDTO } from "../../api/snippets";

const hook = vi.hoisted(() => ({ useSnippets: vi.fn() }));
vi.mock("../../hooks/useSnippets", () => hook);

import { SnippetPicker } from "./SnippetPicker";

const snippets: SnippetDTO[] = [
  { id: "plain", name: "Plain", body: "Demand exact evidence", color: "#38d9c0", pinned: false, use_count: 2, last_used_at: "", sort_order: 0, created_at: "", updated_at: "" },
  { id: "vars", name: "Investigate", body: "Check {{scenario}} with {{owner}}", color: "#4dabf7", pinned: true, use_count: 1, last_used_at: "", sort_order: 0, created_at: "", updated_at: "" },
];

describe("SnippetPicker", () => {
  const touch = vi.fn();
  beforeEach(() => {
    vi.clearAllMocks();
    touch.mockResolvedValue(undefined);
    hook.useSnippets.mockReturnValue({ snippets, status: "ready", touch });
  });

  it("filters by body text and keeps rows at the 44px floor", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    fireEvent.change(screen.getByTestId("snippet-filter"), { target: { value: "exact evidence" } });
    const row = screen.getByTestId("snippet-row-plain");
    expect(row.className).toContain("min-h-11");
    expect(screen.queryByTestId("snippet-row-vars")).toBeNull();
  });

  it("inserts exact plain text and touches only after insertion resolves", async () => {
    let resolve!: () => void;
    const onInsert = vi.fn(() => new Promise<void>((done) => { resolve = done; }));
    render(<SnippetPicker open onClose={vi.fn()} onInsert={onInsert} />);
    fireEvent.click(screen.getByTestId("snippet-row-plain"));
    expect(onInsert).toHaveBeenCalledWith("Demand exact evidence", snippets[0]);
    expect(touch).not.toHaveBeenCalled();
    resolve();
    await waitFor(() => { expect(touch).toHaveBeenCalledWith("plain"); });
  });

  it("detours unresolved names through the variable sheet", () => {
    const onInsert = vi.fn();
    render(<SnippetPicker open onClose={vi.fn()} onInsert={onInsert} autoValues={{ scenario: "web-console" }} />);
    fireEvent.click(screen.getByTestId("snippet-row-vars"));
    expect(onInsert).not.toHaveBeenCalled();
    expect(screen.getByTestId("snippet-variable-readonly-scenario")).toHaveTextContent("web-console");
    expect(screen.getByTestId("snippet-variable-input-owner")).toBeTruthy();
  });

  it("does not touch when dismissed", () => {
    const onClose = vi.fn();
    render(<SnippetPicker open onClose={onClose} onInsert={vi.fn()} />);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(touch).not.toHaveBeenCalled();
  });
});
