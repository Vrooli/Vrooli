import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { clearHistory, recordSearch } from "../../lib/searchHistory";
import { HistoryPanel } from "./HistoryPanel";

describe("HistoryPanel", () => {
  beforeEach(() => {
    window.localStorage.clear();
    clearHistory();
  });
  afterEach(() => {
    cleanup();
  });

  it("shows the empty state when there is no history", () => {
    renderWithProviders(<HistoryPanel onReplay={vi.fn()} />);
    expect(screen.getByTestId(selectors.history.empty)).toHaveTextContent(strings.history.empty);
  });

  it("lists recorded searches and replays on click", () => {
    recordSearch("vrooli", "live");
    const onReplay = vi.fn();
    renderWithProviders(<HistoryPanel onReplay={onReplay} />);

    const items = screen.getAllByTestId(selectors.history.item);
    expect(items).toHaveLength(1);
    fireEvent.click(items[0] as HTMLElement);
    expect(onReplay).toHaveBeenCalledWith("vrooli", "live");
  });

  it("clears history when the clear button is pressed", () => {
    recordSearch("a", "live");
    recordSearch("b", "learnings");
    renderWithProviders(<HistoryPanel onReplay={vi.fn()} />);

    expect(screen.getAllByTestId(selectors.history.item)).toHaveLength(2);
    fireEvent.click(screen.getByTestId(selectors.history.clear));
    expect(screen.getByTestId(selectors.history.empty)).toBeInTheDocument();
  });
});
