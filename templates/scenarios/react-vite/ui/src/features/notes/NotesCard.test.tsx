/**
 * NotesCard tests — focused on the notes-card surface only.
 *
 * Renders <NotesCard /> directly so failures point at notes-feature
 * behaviour, not shell composition. Follows the canonical
 * mock-builder pattern from `@/test-utils`.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeListNotesResponse, makeNote } from "./mocks/factories";
import { makeNotesMocks } from "./mocks/notes";

vi.mock("../../lib/notes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/notes")>();
  return { ...actual, ...makeNotesMocks() };
});

import { NotesCard } from "./NotesCard";
import { selectors } from "../../consts/selectors";

describe("NotesCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when listNotes resolves with no notes", async () => {
    const { listNotes } = await import("../../lib/notes");
    vi.mocked(listNotes).mockResolvedValueOnce(makeListNotesResponse());

    renderWithProviders(<NotesCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.notes.list)).not.toBeInTheDocument();
  });

  it("renders the list when listNotes returns items", async () => {
    const { listNotes } = await import("../../lib/notes");
    vi.mocked(listNotes).mockResolvedValueOnce(
      makeListNotesResponse({
        notes: [
          makeNote({ id: "a", title: "First persisted note" }),
          makeNote({ id: "b", title: "Second persisted note" }),
        ],
      }),
    );

    renderWithProviders(<NotesCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.notes.list);
    expect(list.textContent).toContain("First persisted note");
    expect(list.textContent).toContain("Second persisted note");
  });

  it("invokes createNote when the create button is clicked", async () => {
    const { createNote, listNotes } = await import("../../lib/notes");
    vi.mocked(listNotes).mockResolvedValue(makeListNotesResponse());
    vi.mocked(createNote).mockResolvedValueOnce(makeNote({ id: "new" }));

    const user = userEvent.setup();
    renderWithProviders(<NotesCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.createButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.notes.createButton));

    await waitFor(() => {
      expect(createNote).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(createNote).mock.calls[0]?.[0]).toMatchObject({ title: expect.any(String) });
  });
});
