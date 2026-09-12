/**
 * NotesMeasureCard tests — the measure-result card surface only.
 *
 * Renders <NotesMeasureCard /> directly so failures point at the
 * measure-card behaviour. Follows the canonical mock-builder pattern
 * from `./mocks/notes`: the standalone `countNotesInWindow` API function
 * is substituted, so the card never touches the network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeNotesMocks } from "./mocks/notes";

vi.mock("../../api/notes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/notes")>();
  return { ...actual, ...makeNotesMocks() };
});

import { NotesMeasureCard } from "./NotesMeasureCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("NotesMeasureCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the resolved count once the measure resolves", async () => {
    const { countNotesInWindow } = await import("../../api/notes");
    vi.mocked(countNotesInWindow).mockResolvedValueOnce(5);

    renderWithProviders(<NotesMeasureCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.measure.value)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.notes.measure.value)).toHaveTextContent("5 notes created this week");
    expect(countNotesInWindow).toHaveBeenCalledTimes(1);
  });

  it("renders the error state when the measure call rejects", async () => {
    const { countNotesInWindow } = await import("../../api/notes");
    vi.mocked(countNotesInWindow).mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<NotesMeasureCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.measure.error)).toBeInTheDocument();
    });
  });
});
