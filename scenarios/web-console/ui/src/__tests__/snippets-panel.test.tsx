import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../test-utils";
import type { SnippetDTO, UpsertSnippetInput } from "../api/snippets";

const api = vi.hoisted(() => ({
  rows: [] as SnippetDTO[],
  listSnippets: vi.fn(),
  upsertSnippet: vi.fn(),
  deleteSnippet: vi.fn(),
  touchSnippet: vi.fn(),
  promoteSnippet: vi.fn(),
}));

vi.mock("../api/snippets", async (loadOriginal) => {
  const original = await loadOriginal<typeof import("../api/snippets")>();
  return {
    ...original,
    listSnippets: api.listSnippets,
    upsertSnippet: api.upsertSnippet,
    deleteSnippet: api.deleteSnippet,
    touchSnippet: api.touchSnippet,
    promoteSnippet: api.promoteSnippet,
  };
});

import SnippetsPanel from "../components/settings/SnippetsPanel";
import { SETTINGS_TAB_IDS } from "../components/settings/tabs";
import { resetSnippetsCacheForTests } from "../hooks/useSnippets";

function row(id: string, overrides: Partial<SnippetDTO> = {}): SnippetDTO {
  return {
    id,
    name: id,
    body: "body",
    color: "#6366f1",
    pinned: false,
    use_count: 0,
    last_used_at: "",
    sort_order: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("SnippetsPanel", () => {
  beforeEach(() => {
    resetSnippetsCacheForTests();
    vi.clearAllMocks();
    api.rows = [
      row("seed", { name: "Example", use_count: 7, last_used_at: "2026-01-02T00:00:00Z" }),
      row("owned", { name: "Mine", use_count: 2, last_used_at: "2026-01-01T00:00:00Z" }),
    ];
    api.listSnippets.mockImplementation(async () => [...api.rows]);
    api.upsertSnippet.mockImplementation(async (input: UpsertSnippetInput) => {
      const current = api.rows.find((item) => item.id === input.id);
      const next = row(input.id ?? `new-${String(api.rows.length + 1)}`, {
        ...current,
        name: input.name,
        body: input.body,
        color: input.color ?? current?.color,
        pinned: input.pinned ?? current?.pinned,
        sort_order: input.sort_order ?? current?.sort_order,
      });
      api.rows = current
        ? api.rows.map((item) => item.id === current.id ? next : item)
        : [...api.rows, next];
      return next;
    });
    api.deleteSnippet.mockImplementation(async (id: string) => {
      api.rows = api.rows.filter((item) => item.id !== id);
      return true;
    });
    api.touchSnippet.mockImplementation(async (id: string) => api.rows.find((item) => item.id === id));
    api.promoteSnippet.mockResolvedValue("example-skill");
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  it("places snippets between templates and handoff rules as the tenth settings tab", () => {
    expect(SETTINGS_TAB_IDS).toHaveLength(10);
    expect(SETTINGS_TAB_IDS.slice(6, 9)).toEqual(["templates", "snippets", "handoff-rules"]);
  });

  it("creates a snippet through the New sheet and adds its row", async () => {
    render(<SnippetsPanel />);
    await screen.findByTestId("snippet-settings-row-seed");
    fireEvent.click(screen.getByTestId("snippets-create"));
    fireEvent.change(screen.getByTestId("snippet-save-name"), { target: { value: "Fresh" } });
    fireEvent.change(screen.getByTestId("snippet-save-body"), { target: { value: "fresh body" } });
    fireEvent.click(screen.getByTestId("snippet-save-submit"));
    expect(await screen.findByText("Fresh")).toBeInTheDocument();
  });

  it("edits content without changing the use count", async () => {
    render(<SnippetsPanel />);
    fireEvent.click(await screen.findByText("Example"));
    fireEvent.change(screen.getByTestId("snippet-settings-name"), { target: { value: "Renamed" } });
    fireEvent.change(screen.getByTestId("snippet-settings-body"), { target: { value: "new {{topic}} body" } });
    expect(screen.getByTestId("snippet-settings-variable-count")).toHaveTextContent("snippets.variableCount");
    fireEvent.click(screen.getByTestId("snippet-settings-save"));
    await waitFor(() => expect(within(screen.getByTestId("snippet-settings-row-seed")).getByText("Renamed")).toBeInTheDocument());
    expect(api.rows.find((item) => item.id === "seed")?.use_count).toBe(7);
    expect(within(screen.getByTestId("snippet-settings-row-seed")).getByText("snippets.settings.usedCount")).toBeInTheDocument();
  });

  it("gives seeded and authored rows the same controls", async () => {
    render(<SnippetsPanel />);
    const seed = await screen.findByTestId("snippet-settings-row-seed");
    const owned = screen.getByTestId("snippet-settings-row-owned");
    const controls = (container: HTMLElement, id: string) => within(container)
      .getAllByRole("button")
      .map((button) => button.dataset.testid?.replace(id, "<id>") ?? "select");
    expect(controls(seed, "seed")).toEqual(controls(owned, "owned"));
  });

  it("confirms deletion in a governed dialog and leaves a stated empty state", async () => {
    api.rows = [row("only", { name: "Only" })];
    render(<SnippetsPanel />);
    fireEvent.click(await screen.findByTestId("snippet-settings-delete-only"));
    // The row does not delete on the first press: an in-app dialog stands
    // between the press and the irreversible call, and no browser modal is
    // involved on any path.
    expect(api.deleteSnippet).not.toHaveBeenCalled();
    expect(window.confirm).not.toHaveBeenCalled();
    expect(await screen.findByTestId("snippet-delete-dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("snippet-delete-confirm"));
    expect(await screen.findByTestId("snippets-empty")).toBeInTheDocument();
    await waitFor(() => { expect(screen.queryByTestId("snippet-delete-dialog")).toBeNull(); });
  });

  it("does nothing when deletion is cancelled", async () => {
    render(<SnippetsPanel />);
    fireEvent.click(await screen.findByTestId("snippet-settings-delete-seed"));
    fireEvent.click(await screen.findByTestId("snippet-delete-cancel"));
    expect(api.deleteSnippet).not.toHaveBeenCalled();
    expect(screen.getByTestId("snippet-settings-row-seed")).toBeInTheDocument();
    await waitFor(() => { expect(screen.queryByTestId("snippet-delete-dialog")).toBeNull(); });
  });

  it("moves a newly pinned snippet to the top", async () => {
    render(<SnippetsPanel />);
    await screen.findByTestId("snippet-settings-row-seed");
    fireEvent.click(screen.getByTestId("snippet-settings-pin-owned"));
    await waitFor(() => {
      const rows = screen.getAllByTestId(/snippet-settings-row-/);
      expect(rows[0]).toHaveAttribute("data-testid", "snippet-settings-row-owned");
    });
  });

  it("states the one-way promotion contract and cancellation performs no call", async () => {
    render(<SnippetsPanel />);
    fireEvent.click(await screen.findByText("Example"));
    fireEvent.click(screen.getByTestId("snippet-settings-promote"));
    expect(screen.getByText("snippets.settings.promoteFactGoverned")).toBeInTheDocument();
    expect(screen.getByText("snippets.settings.promoteFactSnippetStays")).toBeInTheDocument();
    expect(screen.getByText("snippets.settings.promoteFactNoSync")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("snippet-promote-cancel"));
    expect(api.promoteSnippet).not.toHaveBeenCalled();
  });

  it("promotes one way and leaves the snippet row unchanged", async () => {
    render(<SnippetsPanel />);
    fireEvent.click(await screen.findByText("Example"));
    fireEvent.click(screen.getByTestId("snippet-settings-promote"));
    fireEvent.click(screen.getByTestId("snippet-promote-confirm"));
    expect(await screen.findByText("snippets.settings.promoteSuccess")).toBeInTheDocument();
    expect(api.promoteSnippet).toHaveBeenCalledWith("seed");
    expect(within(screen.getByTestId("snippet-settings-row-seed")).getByText("Example")).toBeInTheDocument();
    expect(within(screen.getByTestId("snippet-settings-row-seed")).getByText("snippets.settings.usedCount")).toBeInTheDocument();
  });
});
