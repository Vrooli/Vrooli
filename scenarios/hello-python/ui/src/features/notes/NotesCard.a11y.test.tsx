/**
 * NotesCard accessibility regression tests.
 *
 * Notes owns its query and mutation states, so the a11y waits and mocks live
 * with the feature instead of leaking into `App.a11y.test.tsx`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeListNotesResponse, makeNote } from "./mocks/factories";
import { makeNotesMocks } from "./mocks/notes";
import { NotesCard } from "./NotesCard";

vi.mock("../../api/notes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/notes")>();
  return { ...actual, ...makeNotesMocks() };
});

describe("NotesCard accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state without axe violations", async () => {
    const { notesClient } = await import("../../api/notes");
    vi.mocked(notesClient.listNotes).mockResolvedValueOnce(makeListNotesResponse());

    const { container } = renderWithProviders(<NotesCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.empty)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the list state without axe violations", async () => {
    const { notesClient } = await import("../../api/notes");
    vi.mocked(notesClient.listNotes).mockResolvedValueOnce(
      makeListNotesResponse({
        notes: [
          makeNote({ id: "a", title: "First note", attachmentKeys: ["a.txt"] }),
          makeNote({ id: "b", title: "Second note" }),
        ],
      }),
    );

    const { container } = renderWithProviders(<NotesCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.list)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the error state without axe violations", async () => {
    const { notesClient } = await import("../../api/notes");
    vi.mocked(notesClient.listNotes).mockRejectedValueOnce(new Error("notes unavailable"));

    const { container } = renderWithProviders(<NotesCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.error)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
