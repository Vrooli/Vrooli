import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { JournalPage } from "./JournalPage";

vi.mock("../../api/tokenEconomy", () => ({
  minterClient: {
    listJournalEvents: vi.fn().mockResolvedValue({ events: [
      { id: "event-1", kind: 2, amount: 8n, reason: "Weekly chores", actorIdentity: "parent", actorVerificationStatus: 1 },
      { id: "event-2", kind: 3, amount: -3n, reason: "Movie night", actorIdentity: "", actorVerificationStatus: 2 },
    ] }),
    exportJournal: vi.fn().mockResolvedValue({ events: [{ id: "event-1", amount: 8n }] }),
  },
}));

describe("JournalPage", () => {
  afterEach(() => cleanup());
  it("renders append-only evidence and export accessibly", async () => {
    const user = userEvent.setup();
    const createObjectURL = vi.fn(() => "blob:test");
    const revokeObjectURL = vi.fn();
    Object.defineProperty(URL, "createObjectURL", { configurable: true, value: createObjectURL });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: revokeObjectURL });
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
    const { container } = renderWithProviders(<JournalPage />);
    await waitFor(() => expect(screen.getByTestId("journal-event-table")).toBeInTheDocument());
    expect(screen.getByTestId("journal-export")).toBeInTheDocument();
    await user.type(screen.getByTestId("journal-holder-filter"), "sam");
    await user.type(screen.getByTestId("journal-token-filter"), "chores");
    await user.click(screen.getByTestId("journal-export"));
    const { minterClient } = await import("../../api/tokenEconomy");
    await waitFor(() => expect(minterClient.exportJournal).toHaveBeenCalledOnce());
    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:test");
    expect(anchorClick).toHaveBeenCalledOnce();
    await expectNoA11yViolations(container);
  });
});
