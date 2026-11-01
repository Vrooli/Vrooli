import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import userEvent from "@testing-library/user-event";

import { listJournalEntries } from "../../api/journal";
import { assignFacet, listFacets, setPin } from "../../api/operator";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { ThemeProvider } from "../../theme/ThemeProvider";
import { JournalTimeline } from "./JournalTimeline";

vi.mock("../../api/journal", () => ({ listJournalEntries: vi.fn() }));
vi.mock("../../api/operator", () => ({ assignFacet: vi.fn(), listFacets: vi.fn(), setPin: vi.fn() }));

describe("[REQ:VMEM-P1-006] JournalTimeline", () => {
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("renders empty, ready, and error states accessibly in light and dark themes", async () => {
    vi.mocked(listJournalEntries).mockResolvedValueOnce([]);
    const empty = renderWithProviders(<JournalTimeline />);
    await screen.findByTestId(selectors.journal.empty);
    await expectNoA11yViolations(empty.container);
    empty.unmount();

    vi.mocked(listJournalEntries).mockResolvedValueOnce([{ id: "entry", body: "Durable shared memory", facetId: "episode", facetTexts: [], kind: "import" } as never]);
    const ready = renderWithProviders(<JournalTimeline />, {
      extraProviders: (children) => <ThemeProvider initialChoice="dark">{children}</ThemeProvider>,
    });
    await screen.findByTestId(selectors.journal.list);
    await expectNoA11yViolations(ready.container);
    ready.unmount();

    vi.mocked(listJournalEntries).mockRejectedValueOnce(new Error("journal unavailable"));
    renderWithProviders(<JournalTimeline />);
    await waitFor(() => expect(screen.getByTestId(selectors.journal.error)).toBeInTheDocument());
  });

  it("corrects a facet, toggles a pin after acknowledgement, and preserves it on rejection", async () => {
    vi.mocked(listJournalEntries).mockResolvedValue([{ id: "entry", body: "Immutable journal source", facetId: "episode", facetTexts: [], kind: "import" } as never]);
    vi.mocked(listFacets).mockResolvedValue([{ id: "rule", label: "Rule" } as never]);
    vi.mocked(assignFacet).mockResolvedValue(undefined);
    vi.mocked(setPin).mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderWithProviders(<JournalTimeline />);
    await screen.findByText("Immutable journal source");
    const select = screen.getByRole("button", { name: strings.journal.facetCorrection });
    await waitFor(() => expect(select).toBeEnabled());
    await user.click(select);
    await user.click(await screen.findByRole("option", { name: "Rule" }));
    await waitFor(() => expect(assignFacet).toHaveBeenCalledWith("entry", "rule"));
    await user.click(screen.getByRole("button", { name: strings.journal.pin }));
    await screen.findByRole("button", { name: strings.journal.unpin });
    expect(setPin).toHaveBeenLastCalledWith("entry", true);
    await user.click(screen.getByRole("button", { name: strings.journal.unpin }));
    await screen.findByRole("button", { name: strings.journal.pin });
    expect(setPin).toHaveBeenLastCalledWith("entry", false);
    vi.mocked(setPin).mockRejectedValueOnce(new Error("pin rejected"));
    await user.click(screen.getByRole("button", { name: strings.journal.pin }));
    await screen.findByText("pin rejected");
    expect(screen.getByRole("button", { name: strings.journal.pin })).toBeEnabled();
  });
});
