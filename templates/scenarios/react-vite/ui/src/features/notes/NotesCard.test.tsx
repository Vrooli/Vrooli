/**
 * NotesCard tests — focused on the notes-card surface only.
 *
 * Renders <NotesCard /> directly so failures point at notes-feature
 * behaviour, not shell composition. Follows the canonical
 * mock-builder pattern from `@/test-utils`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeCreateNoteResponse, makeListNotesResponse, makeNote } from "./mocks/factories";
import { makeNotesMocks } from "./mocks/notes";

vi.mock("../../api/notes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/notes")>();
  return { ...actual, ...makeNotesMocks() };
});

import { NotesCard } from "./NotesCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("NotesCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when listNotes resolves with no notes", async () => {
    const { notesClient } = await import("../../api/notes");
    vi.mocked(notesClient.listNotes).mockResolvedValueOnce(makeListNotesResponse());

    renderWithProviders(<NotesCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.empty)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.notes.surface)).toHaveAttribute("data-experience-state", "empty");
    expect(screen.queryByTestId(selectors.notes.list)).not.toBeInTheDocument();
  });

  it("renders the list when listNotes returns items", async () => {
    const { notesClient } = await import("../../api/notes");
    vi.mocked(notesClient.listNotes).mockResolvedValueOnce(
      makeListNotesResponse({
        notes: [
          makeNote({ id: "a", title: "First persisted note", attachmentKeys: ["a.txt"] }),
          makeNote({ id: "b", title: "Second persisted note" }),
        ],
      }),
    );

    renderWithProviders(<NotesCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.notes.list);
    expect(screen.getByTestId(selectors.notes.surface)).toHaveAttribute("data-experience-state", "ready");
    expect(list.textContent).toContain("First persisted note");
    expect(list.textContent).toContain("Second persisted note");
    expect(screen.getAllByTestId(selectors.notes.createdAt)).toHaveLength(2);
    expect(screen.getAllByTestId(selectors.notes.attachmentCount)[0]?.textContent).toContain("1");
  });

  it("reports loading and request failure through the semantic surface", async () => {
    const { notesClient } = await import("../../api/notes");
    let reject!: (reason?: unknown) => void;
    vi.mocked(notesClient.listNotes).mockReturnValueOnce(new Promise((_, fail) => { reject = fail; }));

    renderWithProviders(<NotesCard />);
    expect(await screen.findByTestId(selectors.notes.surface)).toHaveAttribute("data-experience-state", "loading");

    reject(new Error("notes unavailable"));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.surface)).toHaveAttribute("data-experience-state", "error");
    });
  });

  it("invokes createNote when the create button is clicked", async () => {
    const { notesClient } = await import("../../api/notes");
    vi.mocked(notesClient.listNotes).mockResolvedValue(makeListNotesResponse());
    vi.mocked(notesClient.createNote).mockResolvedValueOnce(
      makeCreateNoteResponse({ note: makeNote({ id: "new" }) }),
    );

    const user = userEvent.setup();
    renderWithProviders(<NotesCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.createButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.notes.createButton));

    await waitFor(() => {
      expect(notesClient.createNote).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(notesClient.createNote).mock.calls[0]?.[0]).toMatchObject({ title: expect.any(String) });
  });
});
