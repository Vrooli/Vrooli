import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Layers } from "lucide-react";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { selectors } from "../../../../consts/selectors";

describe("SidebarEmptyState", () => {
  it("renders the title and hint when no query is set", () => {
    render(
      <SidebarEmptyState
        icon={Layers}
        title="No operating modes registered."
        hint="Modes appear here as the system learns new methodologies."
      />,
    );
    expect(screen.getByTestId(selectors.sidebar.emptyStateTitle)).toHaveTextContent(
      "No operating modes registered.",
    );
    expect(
      screen.getByText("Modes appear here as the system learns new methodologies."),
    ).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.sidebar.emptyStateClear)).toBeNull();
  });

  it("swaps the title and hides the hint when a query is set", () => {
    render(
      <SidebarEmptyState
        icon={Layers}
        title="No operating modes registered."
        hint="Some hint that should be suppressed."
        query="alpha"
        onClearSearch={() => {}}
      />,
    );
    expect(screen.getByTestId(selectors.sidebar.emptyStateTitle)).toHaveTextContent(
      'No matches for "alpha"',
    );
    expect(screen.queryByText("Some hint that should be suppressed.")).toBeNull();
  });

  it("only renders the Clear search button when both query and onClearSearch are set", async () => {
    const onClearSearch = vi.fn();
    const { rerender } = render(
      <SidebarEmptyState
        icon={Layers}
        title="empty"
        query="alpha"
        onClearSearch={onClearSearch}
      />,
    );
    const clear = screen.getByTestId(selectors.sidebar.emptyStateClear);
    expect(clear).toBeEnabled();
    await userEvent.click(clear);
    expect(onClearSearch).toHaveBeenCalledTimes(1);

    // Without onClearSearch the button is hidden even when a query is set.
    rerender(<SidebarEmptyState icon={Layers} title="empty" query="alpha" />);
    expect(screen.queryByTestId(selectors.sidebar.emptyStateClear)).toBeNull();

    // With onClearSearch but no query, the button is also hidden.
    rerender(
      <SidebarEmptyState icon={Layers} title="empty" onClearSearch={onClearSearch} />,
    );
    expect(screen.queryByTestId(selectors.sidebar.emptyStateClear)).toBeNull();
  });

  it("treats whitespace-only queries as empty", () => {
    render(
      <SidebarEmptyState
        icon={Layers}
        title="No matches yet."
        query="   "
        onClearSearch={() => {}}
      />,
    );
    expect(screen.getByTestId(selectors.sidebar.emptyStateTitle)).toHaveTextContent("No matches yet.");
    expect(screen.queryByTestId(selectors.sidebar.emptyStateClear)).toBeNull();
  });
});
