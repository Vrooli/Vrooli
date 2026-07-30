import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { listJournalEntries } from "../../api/journal";
import { selectors } from "../../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { JournalTimeline } from "./JournalTimeline";

vi.mock("../../api/journal", () => ({ listJournalEntries: vi.fn() }));

describe("[REQ:VMEM-P1-006] JournalTimeline", () => {
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("renders empty, ready, and error states accessibly in light and dark themes", async () => {
    vi.mocked(listJournalEntries).mockResolvedValueOnce([]);
    const empty = renderWithProviders(<JournalTimeline />);
    await screen.findByTestId(selectors.journal.empty);
    await expectNoA11yViolations(empty.container);
    cleanup();

    vi.mocked(listJournalEntries).mockResolvedValueOnce([{ id: "entry", body: "Durable shared memory", facetId: "episode", facetTexts: [], kind: "import" } as never]);
    const ready = renderWithProviders(<JournalTimeline />, { initialTheme: "dark" });
    await screen.findByTestId(selectors.journal.list);
    await expectNoA11yViolations(ready.container);
    cleanup();

    vi.mocked(listJournalEntries).mockRejectedValueOnce(new Error("journal unavailable"));
    renderWithProviders(<JournalTimeline />);
    await waitFor(() => expect(screen.getByTestId(selectors.journal.error)).toBeInTheDocument());
  });
});
