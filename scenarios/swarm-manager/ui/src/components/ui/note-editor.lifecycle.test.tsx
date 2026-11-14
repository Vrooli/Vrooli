import { act, cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { NoteEditor } from "./note-editor";

// Component lifecycle draft: actual NoteEditor; callback is its owner boundary.
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
function edit(value: string) { fireEvent.click(screen.getByRole("button", { name: "Original note" })); fireEvent.change(screen.getByRole("textbox"), { target: { value } }); }
describe("NoteEditor save lifecycle", () => {
  it("keeps optimistic success visible until the owner prop catches up", async () => {
    const save = vi.fn<(_: string) => Promise<void>>().mockResolvedValue(undefined);
    const view = renderWithProviders(<NoteEditor note="Original note" onSave={save} />);
    edit("Saved draft"); fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("button", { name: "Saved draft" })).toBeVisible(); expect(save.mock.calls).toEqual([["Saved draft"]]);
    view.rerender(<NoteEditor note="Saved draft" onSave={save} />);
    expect(screen.getByRole("button", { name: "Saved draft" })).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Saved draft" })); expect(screen.getByRole("textbox")).toHaveValue("Saved draft");
  });
  it("drains one exact pending save and prevents same-turn duplicates, cancel and editing", async () => {
    let finish!: () => void; const save = vi.fn<(_: string) => Promise<void>>(() => new Promise(resolve => { finish = resolve; }));
    renderWithProviders(<NoteEditor note="Original note" onSave={save} />); edit("Pending draft"); const button = screen.getByRole("button", { name: "Save" });
    act(() => { button.click(); button.click(); }); expect(save.mock.calls).toEqual([["Pending draft"]]);
    expect(screen.getByRole("button", { name: "Saving note" })).toBeDisabled(); expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled(); expect(screen.getByRole("textbox")).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(screen.getByRole("textbox")).toHaveValue("Pending draft");
    await act(async () => finish()); expect(await screen.findByRole("button", { name: "Pending draft" })).toBeVisible(); expect(save).toHaveBeenCalledOnce();
  });
  it("handles refusal, preserves the draft and offers only an explicit manual retry", async () => {
    const save = vi.fn<(_: string) => Promise<void>>().mockRejectedValueOnce(new Error("Owner refused note")).mockResolvedValueOnce(undefined);
    renderWithProviders(<NoteEditor note="Original note" onSave={save} />); edit("Unaccepted draft"); fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Owner refused note"); expect(screen.getByRole("textbox")).toHaveValue("Unaccepted draft"); expect(save).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Save" })).not.toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("button", { name: "Unaccepted draft" })).toBeVisible(); expect(screen.queryByRole("alert")).toBeNull(); expect(save.mock.calls).toEqual([["Unaccepted draft"], ["Unaccepted draft"]]);
  });
  it("clears a refused draft only on explicit cancel, then edits the original owner value", async () => {
    const save = vi.fn<(_: string) => Promise<void>>().mockRejectedValue(new Error("Owner refusal"));
    renderWithProviders(<NoteEditor note="Original note" onSave={save} />); edit("Refused draft"); fireEvent.click(screen.getByRole("button", { name: "Save" })); await screen.findByRole("alert");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(screen.queryByRole("alert")).toBeNull(); expect(screen.getByRole("button", { name: "Original note" })).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Original note" })); expect(screen.getByRole("textbox")).toHaveValue("Original note"); expect(save).toHaveBeenCalledOnce();
  });
  it("honors external saving while idle and editing without invoking its owner", async () => {
    const save = vi.fn<(_: string) => Promise<void>>();
    const view = renderWithProviders(<NoteEditor note="Original note" onSave={save} saving />);
    expect(screen.getByRole("button", { name: "Original note" })).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Original note" })); expect(screen.queryByRole("textbox")).toBeNull();
    view.rerender(<NoteEditor note="Original note" onSave={save} />); edit("External pending draft");
    view.rerender(<NoteEditor note="Original note" onSave={save} saving />);
    expect(screen.getByRole("textbox")).toBeDisabled(); expect(screen.getByRole("button", { name: "Saving note" })).toBeDisabled(); expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled(); expect(save).not.toHaveBeenCalled();
    view.rerender(<NoteEditor note="Original note" onSave={save} />); expect(screen.getByRole("textbox")).toHaveValue("External pending draft");
  });
  it("replaces accepted optimistic text with a different authoritative canonical value", async () => {
    const save = vi.fn<(_: string) => Promise<void>>().mockResolvedValue(undefined);
    const view = renderWithProviders(<NoteEditor note="Original note" onSave={save} />);
    edit("Submitted note"); fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("button", { name: "Submitted note" })).toBeVisible();
    view.rerender(<NoteEditor note="Canonical owner note" onSave={save} />);
    expect(await screen.findByRole("button", { name: "Canonical owner note" })).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Canonical owner note" }));
    expect(screen.getByRole("textbox")).toHaveValue("Canonical owner note"); expect(save).toHaveBeenCalledOnce();
  });
  it("preserves an in-progress draft across owner prop changes and cancels to the new owner value", async () => {
    const save = vi.fn<(_: string) => Promise<void>>();
    const view = renderWithProviders(<NoteEditor note="Original note" onSave={save} />);
    edit("Unsubmitted draft"); view.rerender(<NoteEditor note="New authoritative note" onSave={save} />);
    expect(screen.getByRole("textbox")).toHaveValue("Unsubmitted draft");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(await screen.findByRole("button", { name: "New authoritative note" })).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "New authoritative note" }));
    expect(screen.getByRole("textbox")).toHaveValue("New authoritative note"); expect(save).not.toHaveBeenCalled();
  });
  it("keeps an authoritative canonical prop published before the save callback resolves", async () => {
    let complete!: () => void;
    const save = vi.fn<(_: string) => Promise<void>>(() => new Promise(resolve => { complete = resolve; }));
    const view = renderWithProviders(<NoteEditor note="Original note" onSave={save} />);
    edit("Submitted draft"); fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(save.mock.calls).toEqual([["Submitted draft"]]);
    view.rerender(<NoteEditor note="Canonical before completion" onSave={save} />);
    expect(screen.getByRole("textbox")).toHaveValue("Submitted draft");
    expect(screen.getByRole("button", { name: "Saving note" })).toBeDisabled();
    await act(async () => complete());
    expect(await screen.findByRole("button", { name: "Canonical before completion" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Submitted draft" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Canonical before completion" }));
    expect(screen.getByRole("textbox")).toHaveValue("Canonical before completion"); expect(save).toHaveBeenCalledOnce();
  });
  it.each([null, { code: "opaque" }])("handles unknown refusal %s with a bounded domain fallback", async cause => {
    const save = vi.fn<(_: string) => Promise<void>>().mockRejectedValue(cause);
    renderWithProviders(<NoteEditor note="Original note" onSave={save} />); edit("Retained draft"); fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Unable to save this note.")); expect(screen.getByRole("textbox")).toHaveValue("Retained draft"); expect(save).toHaveBeenCalledOnce();
  });
});
