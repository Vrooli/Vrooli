import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { recallMemory } from "../../api/recall";
import { selectors } from "../../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { RecallPanel } from "./RecallPanel";

vi.mock("../../api/recall", () => ({ recallMemory: vi.fn() }));

describe("[REQ:VMEM-P1-003] RecallPanel", () => {
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("searches, labels a summary, and renders accessibly in light and dark themes", async () => {
    vi.mocked(recallMemory).mockResolvedValueOnce([{ nodeId: "summary", entryId: "entry", facetId: "episode", text: "Compressed memory", score: 0.9, depth: 1, summary: true, span: 3 } as never]);
    const user = userEvent.setup();
    const ready = renderWithProviders(<RecallPanel />);
    await user.type(screen.getByRole("textbox"), "shared memory");
    await user.click(screen.getByRole("button"));
    await screen.findByTestId(selectors.recall.list);
    expect(recallMemory).toHaveBeenCalledWith("shared memory", 10);
    await expectNoA11yViolations(ready.container);
    cleanup();

    vi.mocked(recallMemory).mockResolvedValueOnce([]);
    const empty = renderWithProviders(<RecallPanel />, { initialTheme: "dark" });
    await user.type(screen.getByRole("textbox"), "missing");
    await user.click(screen.getByRole("button"));
    await screen.findByTestId(selectors.recall.empty);
    await expectNoA11yViolations(empty.container);

    cleanup();
    vi.mocked(recallMemory).mockRejectedValueOnce(new Error("recall unavailable"));
    renderWithProviders(<RecallPanel />);
    await user.type(screen.getByRole("textbox"), "failure");
    await user.click(screen.getByRole("button"));
    await waitFor(() => expect(screen.getByTestId(selectors.recall.error)).toBeInTheDocument());
  });
});
